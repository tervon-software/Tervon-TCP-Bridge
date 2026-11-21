#!/usr/bin/env python3
"""Read-only M6 check: live Xray TCP inbounds vs persisted Bridge routes.
No hardcoded domains, ports, backend names, or network destinations.
"""
import argparse
import json
import sys
from pathlib import Path

def read(path):
    if not path.is_absolute() or path.is_symlink() or not path.is_file():
        raise ValueError("unsafe or missing config")
    return json.loads(path.read_text(encoding="utf-8"))

def xray_tcp_ports(xray):
    result = {}
    for inbound in xray.get("inbounds", []):
        if not isinstance(inbound, dict):
            continue
        stream = inbound.get("streamSettings") or {}
        if not isinstance(stream, dict):
            continue
        network = str(stream.get("network") or "tcp").lower()
        if network in ("kcp", "quic", "hysteria", "hysteria2"):
            continue
        if str(inbound.get("listen") or "0.0.0.0") in ("127.0.0.1", "::1", "localhost"):
            continue
        port = inbound.get("port")
        if type(port) is int and 1 <= port <= 65535:
            result.setdefault(port, []).append(inbound.get("protocol"))
    return result

def bridge_ports(bridge):
    if bridge.get("mode") != "server":
        raise ValueError("unexpected Bridge mode")
    routes = bridge.get("routes")
    if not isinstance(routes, dict) or not routes:
        raise ValueError("Bridge has no routes")
    result = {}
    for service, backends in routes.items():
        if not isinstance(backends, list) or not backends:
            raise ValueError("empty backend list")
        for backend in backends:
            if not isinstance(backend, dict):
                raise ValueError("invalid backend")
            endpoint = backend.get("Address", backend.get("address"))
            if not isinstance(endpoint, str) or ":" not in endpoint:
                raise ValueError("invalid backend endpoint")
            try:
                raw_port = endpoint.rsplit(":", 1)[1]
                port = int(raw_port)
            except ValueError:
                raise ValueError("invalid backend port") from None
            if not 1 <= port <= 65535 or str(port) != raw_port:
                raise ValueError("invalid backend port")
            if port in result and result[port] != service:
                raise ValueError("ambiguous backend port")
            result[port] = service
    return result

def audit(xray, bridge):
    actual = xray_tcp_ports(xray)
    configured = bridge_ports(bridge)
    return {
        "xray": len(actual), "bridge": len(configured),
        "stale": [p for p in configured if p not in actual],
        "ambiguous": [p for p in configured if len(actual.get(p, [])) != 1],
        "not_exposed": [p for p in actual if p not in configured],
    }

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--xray-config", type=Path, required=True)
    parser.add_argument("--bridge-config", type=Path, required=True)
    args = parser.parse_args()
    try:
        result = audit(read(args.xray_config), read(args.bridge_config))
    except (OSError, ValueError, TypeError, json.JSONDecodeError) as exc:
        print("M6_XRAY_BRIDGE=BLOCKED_" + type(exc).__name__)
        return 2
    for field in ("xray", "bridge", "stale", "ambiguous", "not_exposed"):
        val = result[field]
        print("M6_" + field.upper() + "_COUNT=" + str(val if isinstance(val, int) else len(val)))
    if result["stale"] or result["ambiguous"]:
        print("M6_XRAY_BRIDGE=FAIL_ROUTE_DRIFT")
        return 1
    print("M6_XRAY_BRIDGE=CONFIG_MATCH_ONLY")
    print("TCP_CONNECTIVITY=NOT_TESTED")
    print("SERVER_CONFIG=UNCHANGED")
    return 0

if __name__ == "__main__":
    sys.exit(main())
