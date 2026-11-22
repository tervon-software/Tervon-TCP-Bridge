#!/usr/bin/env python3
"""Resolve the existing, unambiguous VLESS TCP TLS inbound port from Xray JSON.

Read-only; do not infer the backend from a fixed port, SNI, or subscription.
"""
import argparse
import json
from pathlib import Path


def resolve(config: dict) -> int:
    if not isinstance(config, dict) or not isinstance(config.get("inbounds"), list):
        raise ValueError("missing Xray inbounds")
    found = []
    for inbound in config["inbounds"]:
        if not isinstance(inbound, dict) or inbound.get("protocol") != "vless":
            continue
        stream = inbound.get("streamSettings") or {}
        if not isinstance(stream, dict):
            raise ValueError("invalid VLESS streamSettings")
        network = str(stream.get("network") or "tcp").lower()
        if network not in ("tcp", "raw") or stream.get("security") != "tls":
            continue
        listen = str(inbound.get("listen") or "0.0.0.0")
        if listen in ("127.0.0.1", "::1", "localhost"):
            continue
        port = inbound.get("port")
        if type(port) is int and 1 <= port <= 65535:
            found.append(port)
    if len(found) != 1:
        raise ValueError("expected exactly one host-reachable VLESS TCP TLS inbound")
    return found[0]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--xray-config", type=Path, required=True)
    args = parser.parse_args()
    try:
        if not args.xray_config.is_absolute() or args.xray_config.is_symlink():
            raise ValueError("Xray config must be an absolute regular file")
        config = json.loads(args.xray_config.read_text())
        print(resolve(config))
        return 0
    except (OSError, ValueError, TypeError, json.JSONDecodeError):
        # stdout is reserved for the selected port, no secrets or config text.
        print("ERROR: unable to determine one VLESS TCP TLS backend", file=__import__("sys").stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
