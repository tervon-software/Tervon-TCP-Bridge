import unittest
from check_live_xray_bridge_routes import audit

class DynamicRouteChecks(unittest.TestCase):
    def test_runtime_values_not_hardcoded(self):
        for port in (28413, 47821):
            xray = {"inbounds":[{"protocol":"vless","port":port,
                    "streamSettings":{"network":"tcp","security":"tls"}}]}
            bridge = {"mode":"server","routes":{"opaque":[{"Address":"host.docker.internal:"+str(port)}]}}
            result = audit(xray, bridge)
            self.assertEqual(result["stale"], [])
            self.assertEqual(result["ambiguous"], [])
    def test_stale_route(self):
        xray = {"inbounds":[{"protocol":"vless","port":28413}]}
        bridge = {"mode":"server","routes":{"opaque":[{"Address":"host.docker.internal:47821"}]}}
        self.assertEqual(audit(xray, bridge)["stale"], [47821])
    def test_conflicting_services_fail(self):
        bridge = {"mode":"server","routes":{"one":[{"Address":"host.docker.internal:47821"}],
                                            "two":[{"Address":"host.docker.internal:47821"}]}}
        with self.assertRaises(ValueError):
            audit({"inbounds":[]}, bridge)
    def test_local_only_not_routable(self):
        xray = {"inbounds":[{"protocol":"vless","listen":"127.0.0.1","port":47821}]}
        bridge = {"mode":"server","routes":{"opaque":[{"Address":"host.docker.internal:47821"}]}}
        self.assertEqual(audit(xray, bridge)["stale"], [47821])

if __name__ == "__main__":
    unittest.main()
