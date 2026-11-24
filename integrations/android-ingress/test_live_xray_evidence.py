import unittest
from unittest.mock import patch

import live_xray_evidence as live


class LiveXrayEvidenceTest(unittest.TestCase):
    @patch.object(live, "command")
    def test_only_xray_owns_api_socket(self, command):
        command.return_value = (
            'LISTEN 0 4096 127.0.0.1:1234 0.0.0.0:* users:(("xray",pid=21,fd=5))\n'
        )
        live.verify_listener(21, "127.0.0.1", 1234)
        with self.assertRaises(ValueError):
            live.verify_listener(22, "127.0.0.1", 1234)

    @patch.object(live, "verify_listener")
    @patch.object(live, "reachable_ports", return_value={55535})
    @patch.object(live, "backend_target", return_value="192.0.2.1")
    @patch.object(live, "get_xray_live", return_value={55535: ("vless", "xhttp"), 10811: ("socks", "tcp")})
    @patch.object(live, "process_identity", return_value="70")
    @patch.object(live, "find_xray", return_value=(20, 21, "50"))
    def test_publish_only_reachable_vpn_projection(self, process, _, grpc, alias, connect, listener):
        self.assertEqual(live.observe("panel.service", "xray", 30, "backend"), {
            "Address": "192.0.2.1",
            "Inbounds": [{"Port": 55535, "Protocol": "vless", "Network": "xhttp"}],
        })
        grpc.assert_called_once_with(21, "xray", live.verify_listener)
        listener.assert_called_once_with(21, "192.0.2.1", 55535, allow_wildcard=True)
        connect.assert_called_once_with(30, "192.0.2.1", [55535])
        connect.return_value = set()
        with self.assertRaisesRegex(ValueError, "reachability"):
            live.observe("panel.service", "xray", 30, "backend")
        connect.return_value = {55535}
        process.side_effect = [(20, 21, "50"), (20, 21, "51")]
        with self.assertRaisesRegex(ValueError, "identity changed"):
            live.observe("panel.service", "xray", 30, "backend")
