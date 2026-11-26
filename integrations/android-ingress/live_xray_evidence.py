"""Registrar's read-only Xray and Decapsulator TCP observation."""
import ipaddress
import json
import os
from pathlib import Path
import re
import subprocess
import sys

from check_panel_xray_bridge_routes import AuditBlocked, get_xray_live


def command(*args):
    return subprocess.check_output(args, text=True, stderr=subprocess.DEVNULL, timeout=5)


def process_identity(pid):
    proc = Path(f"/proc/{pid}")
    return (proc / "stat").read_text().rsplit(")", 1)[1].split()[19]


def find_xray(unit, executable):
    parent = int(command("systemctl", "show", "--property=MainPID", "--value", "--", unit))
    if parent <= 1:
        raise ValueError("unit inactive")
    matches = []
    for row in command("ps", "-eo", "pid=,ppid=").splitlines():
        pid, ppid = map(int, row.split())
        if ppid != parent:
            continue
        try:
            if Path(os.readlink(f"/proc/{pid}/exe")).name == executable:
                matches.append((parent, pid, process_identity(pid)))
        except FileNotFoundError:
            continue
    if len(matches) != 1:
        raise ValueError("ambiguous Xray process")
    return matches[0]


def verify_listener(pid, address, port, allow_wildcard=False):
    endpoint = f"[{address}]:{port}" if ":" in address else f"{address}:{port}"
    endpoints = {endpoint}
    if allow_wildcard:
        endpoints.add(f"*:{port}")
        endpoints.add(f"[::]:{port}" if ":" in address else f"0.0.0.0:{port}")
    matches = []
    for row in command("ss", "-H", "-ltnp", f"sport = :{port}").splitlines():
        fields = row.split()
        if len(fields) >= 5 and fields[3] in endpoints:
            matches.append(set(re.findall(r"\bpid=(\d+)\b", row)))
    if not matches or any(owners != {str(pid)} for owners in matches):
        raise ValueError("Xray listener ownership unverified")


def backend_target(pid, alias):
    # Native services can use an explicit backend IP. Listener ownership and
    # reachability are checked the same way as for a Docker host alias.
    try:
        return str(ipaddress.ip_address(alias))
    except ValueError:
        pass
    addresses = set()
    for line in Path(f"/proc/{pid}/root/etc/hosts").read_text().splitlines():
        fields = line.split("#", 1)[0].split()
        if len(fields) >= 2 and alias in fields[1:]:
            addresses.add(str(ipaddress.ip_address(fields[0])))
    if len(addresses) != 1:
        raise ValueError("ambiguous backend alias")
    return addresses.pop()


def reachable_ports(pid, address, ports):
    if not ports:
        return set()
    # Use a single namespace entry. Bound the total observation time.
    result = command("nsenter", "-t", str(pid), "-n", "--", sys.executable, "-IB", "-c", """
import concurrent.futures
import json
import socket
import sys

def connect(port):
    try:
        with socket.create_connection((sys.argv[1], port), timeout=1):
            return port
    except OSError:
        return None

ports = json.loads(sys.argv[2])
with concurrent.futures.ThreadPoolExecutor(max_workers=16) as pool:
    print(json.dumps([port for port in pool.map(connect, ports) if port is not None]))
""", address, json.dumps(ports))
    return set(json.loads(result))


def observe(unit, executable, pid, alias):
    before = find_xray(unit, executable)
    container_started = process_identity(pid)
    applied = get_xray_live(before[1], executable, verify_listener)
    eligible = {
        port: (protocol, network)
        for port, (protocol, network) in applied.items()
        if protocol in {"vless", "trojan", "shadowsocks"}
        and network in {"tcp", "ws", "grpc", "httpupgrade", "http", "h2", "xhttp"}
    }
    address = backend_target(pid, alias)
    for port in eligible:
        verify_listener(before[1], address, port, allow_wildcard=True)
    reachable = reachable_ports(pid, address, list(eligible))
    if find_xray(unit, executable) != before or process_identity(pid) != container_started:
        raise ValueError("process identity changed")
    if backend_target(pid, alias) != address:
        raise ValueError("backend alias changed")
    if reachable != eligible.keys():
        raise ValueError("Xray backend reachability incomplete")
    return {
        "Address": address,
        "Inbounds": [
            {"Port": port, "Protocol": protocol, "Network": network}
            for port, (protocol, network) in sorted(eligible.items())
        ],
    }


if __name__ == "__main__":
    try:
        unit, executable, pid, alias = sys.argv[1:]
        print(json.dumps(observe(unit, executable, int(pid), alias)))
    except (AuditBlocked, KeyError, OSError, TypeError, ValueError, subprocess.SubprocessError):
        sys.exit(2)
