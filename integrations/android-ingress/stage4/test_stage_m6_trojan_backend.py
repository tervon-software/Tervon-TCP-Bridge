#!/usr/bin/env python3
"""Unit-only tests for Trojan inbound-derived server route creation."""
import unittest
from stage_m6_trojan_backend import candidate

def fixture():
    xray = {"inbounds":[
        {"protocol":"trojan","port":48321,"streamSettings":{"network":"tcp","security":"tls"}},
        {"protocol":"trojan","port":47319,"streamSettings":{"network":"xhttp","security":"tls"}},
        {"protocol":"vless","port":45111,"streamSettings":{"network":"tcp","security":"tls"}},
    ]}
    bridge = {
        "mode":"server",
        "routes":{"vpn":[{"Name":"old","Address":"host.docker.internal:45111"}]},
        "authorized":{"a"*64:["vpn"],"b"*64:["other"]},
        "padding_enabled": True
    }
    return xray, bridge

class TrojanBackendTests(unittest.TestCase):
    def test_routes_derive_ports_and_preserve_old(self):
        xray,bridge=fixture()
        out=candidate(xray,bridge)
        self.assertEqual(out["routes"]["vpn-trojan"][0]["Address"],"host.docker.internal:48321")
        self.assertEqual(out["routes"]["vpn-trojan-xhttp"][0]["Address"],"host.docker.internal:47319")
        self.assertEqual(out["routes"]["vpn"],bridge["routes"]["vpn"])
        self.assertEqual(out["authorized"]["a"*64],["vpn","vpn-trojan","vpn-trojan-xhttp"])
        self.assertEqual(out["authorized"]["b"*64],["other"])
        self.assertEqual(bridge["authorized"]["a"*64],["vpn"])
        self.assertNotIn("vpn-trojan",bridge["routes"])
        self.assertTrue(out["padding_enabled"])

    def test_missing_or_duplicate_refused(self):
        xray,bridge=fixture()
        xray["inbounds"] = [x for x in xray["inbounds"] if x["protocol"]!="trojan"]
        with self.assertRaises(ValueError):
            candidate(xray,bridge)
        xray,bridge=fixture()
        xray["inbounds"].append(xray["inbounds"][0].copy())
        with self.assertRaises(ValueError):
            candidate(xray,bridge)
        xray,bridge=fixture()
        bridge["routes"]["vpn-trojan"]=[]
        with self.assertRaises(ValueError):
            candidate(xray,bridge)

    def test_no_loopback_or_udp_route(self):
        xray,bridge=fixture()
        xray["inbounds"][0]["listen"]="127.0.0.1"
        with self.assertRaises(ValueError):
            candidate(xray,bridge)
        xray,bridge=fixture()
        xray["inbounds"][0]["streamSettings"]["network"]="quic"
        out=candidate(xray,bridge)
        self.assertNotIn("vpn-trojan",out["routes"])

if __name__=="__main__":
    unittest.main()
