#!/usr/bin/env python3
"""Offline fixtures for the M6 read-only 3x-ui/Xray/Bridge audit."""
import json
import tempfile
import unittest
from pathlib import Path

from check_panel_xray_bridge_routes import (
    AuditBlocked, approved_routes, compare, normalize, parse_panel,
)


def record(port=49211, protocol="vless", network="xhttp", enable=True, ident=10):
    return {"id": ident, "port": port, "protocol": protocol, "enable": enable,
            "streamSettings": json.dumps({"network": network, "security": "tls"})}


class PanelRuntimeBridgeAuditTests(unittest.TestCase):
    def test_enabled_route_alias_matches(self):
        panel = parse_panel({"success": True, "obj": [record()]})
        result = compare({49211: "opaque"}, panel, {49211: ("vless", "xhttp")})
        self.assertEqual(result[0][3], "MATCH")
        self.assertEqual(normalize("SplitHTTP"), "xhttp")

    def test_disabled_is_not_authorized(self):
        panel = parse_panel({"success": True, "obj": [record(enable=False)]})
        result = compare({49211: "opaque"}, panel, {49211: ("vless", "xhttp")})
        self.assertEqual(result[0][3], "DISABLED_IN_PANEL")

    def test_runtime_protocol_drift(self):
        panel = parse_panel({"success": True, "obj": [record()]})
        result = compare({49211: "opaque"}, panel, {49211: ("trojan", "xhttp")})
        self.assertEqual(result[0][3], "PANEL_XRAY_MISMATCH")

    def test_missing_panel_or_runtime(self):
        self.assertEqual(compare({49211: "svc"}, {}, {49211: ("vless", "xhttp")})[0][3],
                         "NOT_IN_PANEL")
        panel = parse_panel({"success": True, "obj": [record()]})
        self.assertEqual(compare({49211: "svc"}, panel, {})[0][3],
                         "NOT_IN_XRAY_RUNTIME")

    def test_unapproved_panel_inbound_never_added(self):
        panel = parse_panel({"success": True, "obj": [
            record(), record(49212, "socks", "tcp", True, 11)]})
        results = compare({49211: "only-approved"}, panel,
                          {49211: ("vless", "xhttp"), 49212: ("socks", "tcp")})
        self.assertEqual(len(results), 1)
        self.assertEqual(results[0][0], 49211)

    def test_duplicate_port_and_bad_stream_rejected(self):
        duplicate = {"success": True, "obj": [
            record(), record(49211, "trojan", "tcp", True, 11)]}
        with self.assertRaises(AuditBlocked):
            parse_panel(duplicate)
        invalid = {"success": True, "obj": [dict(record(), streamSettings="not-json")]}
        with self.assertRaises(AuditBlocked):
            parse_panel(invalid)

    def test_bridge_existing_routes_and_ambiguity(self):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / "server.json"
            path.write_text(json.dumps({"mode":"server", "routes":{
                "approved":[{"Address":"host.docker.internal:49211"}]}}))
            self.assertEqual(approved_routes(path), {49211: "approved"})
            path.write_text(json.dumps({"mode":"server", "routes":{
                "a":[{"Address":"host.docker.internal:49211"}],
                "b":[{"Address":"host.docker.internal:49211"}]}}))
            with self.assertRaises(AuditBlocked):
                approved_routes(path)


if __name__ == "__main__":
    unittest.main()
