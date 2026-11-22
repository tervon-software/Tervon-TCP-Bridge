#!/usr/bin/env python3
"""M6 read-only panel ↔ live-Xray ↔ approved Bridge routes audit.

No production mutations. No raw panel or Xray API responses are printed.
The only route authorization source is the existing server.json route set.
"""
import argparse
import json
import os
import stat
import subprocess
import sys
from pathlib import Path
from urllib.parse import urlsplit, urlunsplit

from check_xray_grpc_runtime import KNOWN_PID, extract

MAX_RESPONSE = 2 << 20
TCP = {"tcp", "raw", "ws", "grpc", "httpupgrade", "http", "h2", "xhttp", "splithttp"}


class AuditBlocked(Exception):
    pass


def normalize(net):
    net = net.lower()
    return {"raw": "tcp", "splithttp": "xhttp"}.get(net, net)


def approved_routes(path):
    cfg = json.loads(Path(path).read_text(encoding="utf-8"))
    if cfg.get("mode") != "server" or not isinstance(cfg.get("routes"), dict):
        raise AuditBlocked("BRIDGE_ROUTES_INVALID")
    routes = cfg["routes"]
    if not 1 <= len(routes) <= 256:
        raise AuditBlocked("BRIDGE_ROUTES_INVALID")
    by_port = {}
    for service, backends in routes.items():
        if not isinstance(service, str) or not service or not isinstance(backends, list) or not backends:
            raise AuditBlocked("BRIDGE_ROUTES_INVALID")
        ports = set()
        for backend in backends:
            if not isinstance(backend, dict):
                raise AuditBlocked("BRIDGE_BACKEND_INVALID")
            address = backend.get("Address", backend.get("address"))
            if not isinstance(address, str):
                raise AuditBlocked("BRIDGE_BACKEND_INVALID")
            # net.SplitHostPort equivalents for IPv4/domain and bracketed IPv6.
            if address.startswith("["):
                closing = address.find("]")
                if closing < 2 or address[closing + 1:closing + 2] != ":":
                    raise AuditBlocked("BRIDGE_BACKEND_INVALID")
                host, number = address[1:closing], address[closing + 2:]
            else:
                if address.count(":") != 1:
                    raise AuditBlocked("BRIDGE_BACKEND_INVALID")
                host, number = address.rsplit(":", 1)
            if not host or not number.isascii() or not number.isdecimal():
                raise AuditBlocked("BRIDGE_BACKEND_INVALID")
            port = int(number)
            if not 1 <= port <= 65535:
                raise AuditBlocked("BRIDGE_BACKEND_INVALID")
            ports.add(port)
        if len(ports) != 1:
            raise AuditBlocked("BRIDGE_BACKEND_AMBIGUOUS")
        port = next(iter(ports))
        if port in by_port:
            raise AuditBlocked("BRIDGE_DUPLICATE_PORT")
        by_port[port] = service
    return by_port


def parse_panel(payload):
    if not isinstance(payload, dict) or payload.get("success") is not True:
        raise AuditBlocked("PANEL_ENVELOPE_INVALID")
    entries = payload.get("obj")
    if not isinstance(entries, list) or len(entries) > 4096:
        raise AuditBlocked("PANEL_LIST_INVALID")
    by_port = {}
    ids = set()
    for row in entries:
        if not isinstance(row, dict):
            raise AuditBlocked("PANEL_INBOUND_INVALID")
        uid, port, enabled = row.get("id"), row.get("port"), row.get("enable")
        protocol = row.get("protocol")
        if type(uid) is not int or uid <= 0 or type(port) is not int or not 1 <= port <= 65535 or type(enabled) is not bool or not isinstance(protocol, str) or not protocol:
            raise AuditBlocked("PANEL_INBOUND_INVALID")
        if uid in ids or port in by_port:
            raise AuditBlocked("PANEL_INBOUND_DUPLICATE")
        ids.add(uid)
        settings = row.get("streamSettings")
        if isinstance(settings, str):
            try:
                settings = json.loads(settings)
            except (ValueError, TypeError):
                raise AuditBlocked("PANEL_STREAM_INVALID") from None
        if not isinstance(settings, dict):
            raise AuditBlocked("PANEL_STREAM_INVALID")
        network = settings.get("network") or "tcp"
        if not isinstance(network, str) or not network:
            raise AuditBlocked("PANEL_STREAM_INVALID")
        by_port[port] = (protocol.lower(), normalize(network), enabled)
    return by_port


def get_panel(base_url, header_file):
    if not isinstance(base_url, str) or not base_url:
        raise AuditBlocked("PANEL_URL_NOT_SUPPLIED")
    parsed = urlsplit(base_url)
    if parsed.scheme != "https" or not parsed.hostname or parsed.username or parsed.password or parsed.query or parsed.fragment:
        raise AuditBlocked("PANEL_URL_INVALID")
    try:
        port = parsed.port or 443
    except ValueError:
        raise AuditBlocked("PANEL_URL_INVALID") from None
    if not 1 <= port <= 65535:
        raise AuditBlocked("PANEL_URL_INVALID")
    basepath = parsed.path.rstrip("/")
    if "//" in basepath or "/../" in basepath or basepath.endswith("/.."):
        raise AuditBlocked("PANEL_URL_INVALID")
    endpoint = urlunsplit(("https", parsed.netloc, basepath + "/panel/api/inbounds/list/slim", "", ""))
    credential = Path(header_file)
    mode = credential.stat().st_mode
    if not stat.S_ISREG(mode) or (mode & 0o077):
        raise AuditBlocked("PANEL_CREDENTIAL_PERMISSIONS")
    # Curl honors the existing header file without exposing its content as argv.
    # --connect-to preserves TLS hostname verification while reaching local panel.
    host = parsed.hostname
    connect = f"{host}:{port}:127.0.0.1:{port}"
    process = subprocess.run(
        ["curl", "--silent", "--show-error", "--fail", "--max-time", "12",
         "--noproxy", "*", "--connect-to", connect, "--header", "@" + str(credential),
         "--header", "Accept: application/json",
         "--header", "X-Requested-With: XMLHttpRequest", endpoint],
        capture_output=True, timeout=16, check=False,
    )
    if process.returncode or len(process.stdout) > MAX_RESPONSE:
        raise AuditBlocked("PANEL_REQUEST_FAILED")
    try:
        return parse_panel(json.loads(process.stdout))
    except (UnicodeError, ValueError, TypeError):
        raise AuditBlocked("PANEL_RESPONSE_INVALID") from None


def get_xray_live():
    proc = Path(f"/proc/{KNOWN_PID}")
    try:
        if Path(os.readlink(proc / "exe")).name != "xray-linux-amd64":
            raise AuditBlocked("XRAY_PROCESS_CHANGED")
        args = [x.decode("utf-8") for x in (proc / "cmdline").read_bytes().split(b"\0") if x]
        paths = []
        for i, item in enumerate(args):
            if item in ("-config", "--config", "-c") and i + 1 < len(args):
                paths.append(args[i + 1])
            elif item.startswith(("-config=", "--config=")):
                paths.append(item.split("=", 1)[1])
        if len(paths) != 1:
            raise AuditBlocked("XRAY_CONFIG_AMBIGUOUS")
        path = Path(paths[0])
        if not path.is_absolute():
            path = Path(os.readlink(proc / "cwd")) / path
        cfg = json.loads(path.read_text(encoding="utf-8"))
        api = [v for v in cfg.get("inbounds", []) if isinstance(v, dict) and v.get("tag") == "api"]
        if len(api) != 1 or api[0].get("listen") not in ("127.0.0.1", "::1"):
            raise AuditBlocked("XRAY_API_UNAVAILABLE")
        address, port = api[0]["listen"], api[0].get("port")
        if type(port) is not int or not 1 <= port <= 65535:
            raise AuditBlocked("XRAY_API_UNAVAILABLE")
        endpoint = f"[{address}]:{port}" if ":" in address else f"{address}:{port}"
        process = subprocess.run(
            [str(proc / "exe"), "api", "lsi", "--server=" + endpoint,
             "--isOnlyTags=false", "--timeout=3"],
            capture_output=True, timeout=8, check=False,
        )
        if process.returncode or len(process.stdout) > 8 << 20:
            raise AuditBlocked("XRAY_GRPC_QUERY_FAILED")
        rows = extract(json.loads(process.stdout).get("inbounds"))
    except AuditBlocked:
        raise
    except (OSError, TypeError, ValueError, UnicodeError, subprocess.TimeoutExpired):
        raise AuditBlocked("XRAY_RUNTIME_UNAVAILABLE") from None
    mapped = {}
    for tag, port, protocol, network in rows:
        if tag == "api":
            continue
        if port in mapped or protocol == "unknown":
            raise AuditBlocked("XRAY_INBOUND_AMBIGUOUS")
        mapped[port] = (protocol, normalize(network))
    return mapped


def compare(approved, panel, runtime):
    outcomes = []
    for port in sorted(approved):
        p, live = panel.get(port), runtime.get(port)
        if p is None:
            status = "NOT_IN_PANEL"
        elif not p[2]:
            status = "DISABLED_IN_PANEL"
        elif live is None:
            status = "NOT_IN_XRAY_RUNTIME"
        elif p[:2] != live:
            status = "PANEL_XRAY_MISMATCH"
        elif live[1] not in {normalize(x) for x in TCP}:
            status = "UNSUPPORTED_TRANSPORT"
        else:
            status = "MATCH"
        outcomes.append((port, p[0] if p else "unavailable",
                         p[1] if p else "unavailable", status))
    return outcomes


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--panel-base-url", default=os.environ.get("M6_PANEL_BASE_URL"))
    parser.add_argument("--panel-header", default="/etc/tervon/bridge-android/private/xui-api-header")
    parser.add_argument("--bridge-config", default="/etc/tervon/bridge-android/config/server.json")
    args = parser.parse_args()
    try:
        approved = approved_routes(args.bridge_config)
        panel = get_panel(args.panel_base_url, args.panel_header)
        runtime = get_xray_live()
        results = compare(approved, panel, runtime)
    except (AuditBlocked, OSError, TypeError, ValueError, json.JSONDecodeError) as ex:
        reason = ex.args[0] if isinstance(ex, AuditBlocked) else "INPUT_OR_SOURCE_INVALID"
        print("M6_PANEL_RUNTIME_AUDIT=BLOCKED")
        print("REASON=" + reason)
        return 2
    for port, protocol, network, status in results:
        print(f"M6_APPROVED_ROUTE port={port} protocol={protocol} network={network} status={status}")
    ok = bool(results) and all(row[3] == "MATCH" for row in results)
    print("M6_APPROVED_ROUTE_COUNT=" + str(len(results)))
    print("M6_MATCHED_ROUTE_COUNT=" + str(sum(row[3] == "MATCH" for row in results)))
    print("M6_PANEL_SOURCE=LIVE_3XUI_API")
    print("M6_XRAY_SOURCE=LIVE_GRPC")
    print("M6_ALLOWED_SOURCE=EXISTING_SERVER_JSON")
    print("M6_PRODUCTION_POLICY_ACK=NOT_TESTED")
    print("SECRETS=NOT_PRINTED")
    print("RUNNING_SERVICES=UNCHANGED")
    print("M6_PANEL_RUNTIME_AUDIT=" + ("PASS_READ_ONLY" if ok else "FAIL_CLOSED"))
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
