#!/usr/bin/env python3
"""Read-only candidate builder for optional Trojan TCP/XHTTP Bridge routes.

Only the existing JSON configuration and actual Trojan inbound ports are used.
Does not change any active Xray/Bridge service, ACL, Docker, or Android app.
"""
import argparse
import json
import os
from pathlib import Path
import sys

TRANSPORTS = {
    "tcp": "vpn-trojan",
    "xhttp": "vpn-trojan-xhttp",
}

def candidate(xray, bridge):
    if bridge.get("mode") != "server" or not isinstance(bridge.get("routes"), dict):
        raise ValueError("invalid Bridge server mode/routes")
    routes = bridge["routes"]
    vpn = routes.get("vpn")
    if not isinstance(vpn, list) or len(vpn) != 1:
        raise ValueError("one existing Bridge TCP backend required")
    current = vpn[0]
    gateway = current.get("Address", current.get("address", ""))
    if not isinstance(gateway, str) or ":" not in gateway:
        raise ValueError("invalid existing Docker backend address")
    gateway_host, gateway_port = gateway.rsplit(":", 1)
    if not gateway_host or not gateway_port.isdecimal() or int(gateway_port) not in range(1,65536):
        raise ValueError("invalid existing Docker host gateway")
    matches = {}
    for record in xray.get("inbounds", []):
        if record.get("protocol") != "trojan":
            continue
        stream = record.get("streamSettings") or {}
        name = str(stream.get("network") or "tcp").lower()
        if name in ("raw", ""):
            name = "tcp"
        if name == "splithttp":
            name = "xhttp"
        if name not in TRANSPORTS:
            continue
        if stream.get("security") != "tls":
            continue
        if str(record.get("listen") or "0.0.0.0") in ("127.0.0.1", "::1", "localhost"):
            raise ValueError("Trojan backend bound only on loopback")
        port = record.get("port")
        if type(port) is not int or port not in range(1,65536):
            raise ValueError("invalid Trojan inbound port")
        matches.setdefault(name, []).append(port)
    if not matches or any(len(found) != 1 for found in matches.values()):
        raise ValueError("Trojan inbound missing or ambiguous")
    output = json.loads(json.dumps(bridge))
    for transport, ports in matches.items():
        service = TRANSPORTS[transport]
        if service in routes:
            raise ValueError("existing Trojan Bridge service must not be overwritten")
        output["routes"][service] = [{"Name": "trojan-" + transport,
                                     "Address": f"{gateway_host}:{ports[0]}"}]
    acl = output.get("authorized")
    if not isinstance(acl, dict):
        raise ValueError("Bridge ACL absent")
    for fp, allowed in acl.items():
        if not isinstance(allowed, list):
            raise ValueError("invalid ACL entry")
        if "vpn" in allowed:
            for transport in matches:
                service = TRANSPORTS[transport]
                if service not in allowed:
                    allowed.append(service)
    return output

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--xray-config", type=Path, required=True)
    parser.add_argument("--bridge-config", type=Path, required=True)
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--check", action="store_true")
    group.add_argument("--stage", type=Path)
    a = parser.parse_args()
    try:
        xray = json.loads(a.xray_config.read_text())
        bridge = json.loads(a.bridge_config.read_text())
        output = candidate(xray, bridge)
        print("M6_TROJAN_BACKEND_CANDIDATE=VALID")
        print("TROJAN_TRANSPORTS=" + ",".join(sorted(
            k for k,v in TRANSPORTS.items() if v in output["routes"])))
        if a.check:
            print("LIVE_SERVICES=UNCHANGED")
            return 0
        target = a.stage
        if not target.is_absolute() or target.is_symlink():
            raise ValueError("stage candidate must be new absolute file")
        data = (json.dumps(output, sort_keys=True, indent=2) + "\n").encode()
        fd = os.open(target, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
        try:
            with os.fdopen(fd, "wb") as stream:
                stream.write(data)
                stream.flush()
                os.fsync(stream.fileno())
        except Exception:
            target.unlink(missing_ok=True)
            raise
        print("M6_TROJAN_CANDIDATE=STAGED_PRIVATE")
        print("LIVE_SERVICES=UNCHANGED")
        return 0
    except (OSError, ValueError, TypeError, json.JSONDecodeError):
        print("M6_TROJAN_BACKEND_CANDIDATE=BLOCKED")
        return 1

if __name__ == "__main__":
    sys.exit(main())
