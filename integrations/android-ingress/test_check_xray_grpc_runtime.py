#!/usr/bin/env python3
"""Fixtures matching Xray-core common/reflect/marshal.go custom JSON."""
import unittest
from check_xray_grpc_runtime import extract


def handler(port=55535, protocol="vless", network="splithttp", tag="test-1"):
    return {
        "tag": tag,
        "receiverSettings": {
            "_TypedMessage_": "xray.app.proxyman.ReceiverConfig",
            "portList": port,
            "streamSettings": {"protocolName": network},
            "listen": "0.0.0.0",
        },
        "proxySettings": {
            "_TypedMessage_": "xray.proxy." + protocol + ".inbound.Config",
            "clients": [{"id": "secret-that-must-not-be-printed"}],
        },
    }


class TestRuntimeGRPCProjection(unittest.TestCase):
    def test_extract_custom_cli_json(self):
        self.assertEqual(
            extract([handler()]),
            [("test-1", 55535, "vless", "splithttp")],
        )

    def test_internal_api_does_not_require_proxy_credentials(self):
        self.assertEqual(
            extract([{"tag": "api"}, handler()]),
            [("api", 0, "internal", "internal"),
             ("test-1", 55535, "vless", "splithttp")],
        )

    def test_reject_range(self):
        for value in ("1000-2000", "1000,2000", 0, 65536, True):
            bad = handler(port=value)
            with self.subTest(value=value), self.assertRaises(ValueError):
                extract([bad])

    def test_reject_duplicate_tag(self):
        with self.assertRaises(ValueError):
            extract([handler(), handler(port=30000)])

    def test_reject_invalid_schema(self):
        for bad in (
            {"tag": "test-1"},
            dict(handler(), receiverSettings={
                "_TypedMessage_": "unexpected.Type", "portList": 55535,
            }),
            dict(handler(), proxySettings={"type": "not-CLI-format"}),
        ):
            with self.assertRaises(ValueError):
                extract([bad])

    def test_trojan_and_socks_server_config(self):
        for protocol in ("trojan", "socks", "http"):
            with self.subTest(protocol=protocol):
                inbound=handler(port=30000,protocol=protocol)
                inbound["proxySettings"]["_TypedMessage_"] = (
                    "xray.proxy."+protocol+".ServerConfig")
                self.assertEqual(extract([inbound])[0][2], protocol)

    def test_dokodemo_config(self):
        inbound=handler(port=10811)
        inbound["proxySettings"]["_TypedMessage_"]="xray.proxy.dokodemo.Config"
        self.assertEqual(extract([inbound])[0][2], "dokodemo-door")

    def test_default_tcp_network(self):
        bad = handler()
        bad["receiverSettings"].pop("streamSettings")
        self.assertEqual(extract([bad])[0][3], "tcp")


if __name__ == "__main__":
    unittest.main()
