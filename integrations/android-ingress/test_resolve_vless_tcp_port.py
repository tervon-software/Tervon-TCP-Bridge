#!/usr/bin/env python3
"""Unit regression for M6 backend discovery; no live Xray or Docker access."""
import unittest
from resolve_vless_tcp_port import resolve


def inbound(port, network="tcp", security="tls", listen="0.0.0.0"):
    return {
        "protocol": "vless",
        "port": port,
        "listen": listen,
        "streamSettings": {"network": network, "security": security},
    }


class ResolverTests(unittest.TestCase):
    def test_uses_actual_port_not_any_constant(self):
        for port in (23456, 42117, 56123):
            self.assertEqual(resolve({"inbounds": [inbound(port)]}), port)
        self.assertEqual(resolve({"inbounds": [inbound(43819, network="raw")]}), 43819)

    def test_ignores_xhttp_and_local_only_listeners(self):
        self.assertEqual(resolve({"inbounds": [
            inbound(42311, network="xhttp"),
            inbound(58911, listen="127.0.0.1"),
            inbound(42779),
        ]}), 42779)

    def test_rejects_missing_ambiguous_or_non_tls(self):
        for x in (
            {},
            {"inbounds": []},
            {"inbounds": [inbound(41772), inbound(43811)]},
            {"inbounds": [inbound(40021, security="none")]},
            {"inbounds": [inbound(40021, network="xhttp")]},
            {"inbounds": [inbound(0)]},
            {"inbounds": [inbound(True)]},
        ):
            with self.subTest(config=x):
                with self.assertRaises(ValueError):
                    resolve(x)


if __name__ == "__main__":
    unittest.main()
