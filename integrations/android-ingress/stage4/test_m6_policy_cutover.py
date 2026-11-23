import datetime as dt
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location(
    "m6",
    Path(__file__).with_name(
        "m6_policy_cutover.py"
    ),
)

m6 = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m6)


class Tests(unittest.TestCase):

    def setUp(self):
        self.digest = "a" * 64
        self.issued = dt.datetime(
            2026, 11, 23, 9, 42,
            tzinfo=dt.timezone.utc,
        )
        self.stamp = (
            "2026-11-23T09:42:04.999470907Z"
        )
        self.ack = (
            "M6_POLICY_RELOAD=PASS digest="
            + self.digest
        )

    def test_ack_go(self):
        output = (
            self.stamp
            + " 2026/11/23 09:42:04 "
            + self.ack
        )
        self.assertTrue(
            m6.ack_in_logs(
                output, self.issued, self.digest
            )
        )

    def test_ack_direct(self):
        self.assertTrue(
            m6.ack_in_logs(
                self.stamp + " " + self.ack,
                self.issued,
                self.digest,
            )
        )

    def test_stale(self):
        self.assertFalse(
            m6.ack_in_logs(
                "2026-11-23T09:41:59Z "
                + self.ack,
                self.issued,
                self.digest,
            )
        )

    def test_wrong_digest(self):
        self.assertFalse(
            m6.ack_in_logs(
                self.stamp + " " + self.ack,
                self.issued,
                "b" * 64,
            )
        )

    def test_suffix(self):
        self.assertFalse(
            m6.ack_in_logs(
                self.stamp + " " + self.ack
                + " suffix",
                self.issued,
                self.digest,
            )
        )

    def test_acl(self):
        self.assertFalse(
            m6.ack_in_logs(
                self.stamp
                + " M6_ACL_RELOAD=PASS digest="
                + self.digest,
                self.issued,
                self.digest,
            )
        )

    def test_stderr(self):
        result = subprocess.CompletedProcess(
            [],
            0,
            "",
            self.stamp + " " + self.ack,
        )

        with patch.object(
            m6.subprocess,
            "run",
            return_value=result,
        ):
            output = m6.command(
                "docker", "logs", "--timestamps"
            )

        self.assertTrue(
            m6.ack_in_logs(
                output, self.issued, self.digest
            )
        )

    def test_registrar_missing(self):
        with patch.object(
            m6,
            "command",
            return_value="not-found",
        ) as mock:
            m6.stop_registrar_if_present()
            self.assertEqual(mock.call_count, 1)

    def test_registrar_present(self):
        with patch.object(
            m6,
            "command",
            side_effect=["loaded", ""],
        ) as mock:
            m6.stop_registrar_if_present()
            self.assertEqual(mock.call_count, 2)

    def test_digest(self):
        config = {
            "routes": {
                "vpn": [{
                    "Name": "vpn",
                    "Address": "127.0.0.1:30000",
                }]
            },
            "authorized": {
                "a" * 64: ["vpn"]
            },
        }

        raw = (
            b'{"routes":{"vpn":[{"Name":"vpn",'
            b'"Address":"127.0.0.1:30000"}]},'
            b'"authorized":{"'
            + b"a" * 64
            + b'":["vpn"]}}'
        )

        self.assertEqual(
            m6.generation_digest(config),
            hashlib.sha256(raw).hexdigest(),
        )

    def old_config(self):
        return {
            "mode": "server",
            "routes": {
                "vpn": [{
                    "Name": "vpn",
                    "Address": "127.0.0.1:30000",
                }],
                "vpn-xhttp": [{
                    "Name": "vpn-xhttp",
                    "Address": "127.0.0.1:55636",
                }],
            },
            "authorized": {
                "a" * 64: [
                    "vpn",
                    "vpn-xhttp",
                ]
            },
        }

    def test_success_cutover(self):
        with tempfile.TemporaryDirectory() as d:
            path = Path(d) / "server.json"
            old = self.old_config()
            path.write_text(json.dumps(old))

            with (
                patch.object(
                    m6, "preflight",
                    return_value=(
                        path.stat(),
                        "cid",
                        old,
                        "127.0.0.1",
                    ),
                ),
                patch.object(
                    m6,
                    "stop_registrar_if_present",
                ),
                patch.object(
                    m6,
                    "reload_confirm",
                ) as ack,
                patch.object(
                    m6,
                    "start_registrar",
                ),
            ):
                result = m6.cutover(
                    path,
                    Path("/bin/true"),
                    "c",
                    "image",
                    Path(d),
                )

                self.assertEqual(result, 0)
                self.assertEqual(
                    len(json.loads(
                        path.read_text()
                    )["routes"]),
                    3,
                )
                ack.assert_called_once()

    def test_failed_ack_rolls_back(self):
        with tempfile.TemporaryDirectory() as d:
            path = Path(d) / "server.json"
            old = self.old_config()
            path.write_text(json.dumps(old))

            with (
                patch.object(
                    m6, "preflight",
                    return_value=(
                        path.stat(),
                        "cid",
                        old,
                        "127.0.0.1",
                    ),
                ),
                patch.object(
                    m6,
                    "stop_registrar_if_present",
                ),
                patch.object(
                    m6,
                    "reload_confirm",
                    side_effect=[
                        RuntimeError("timeout"),
                        None,
                    ],
                ) as ack,
                patch.object(
                    m6,
                    "start_registrar",
                ) as registrar,
            ):
                result = m6.cutover(
                    path,
                    Path("/bin/true"),
                    "c",
                    "image",
                    Path(d),
                )

                self.assertEqual(result, 1)
                self.assertEqual(
                    json.loads(path.read_text()),
                    old,
                )
                self.assertEqual(
                    ack.call_count, 2
                )
                registrar.assert_called_once()

    def test_rollback_failure_reported(self):
        with tempfile.TemporaryDirectory() as d:
            path = Path(d) / "server.json"
            old = self.old_config()
            path.write_text(json.dumps(old))

            with (
                patch.object(
                    m6, "preflight",
                    return_value=(
                        path.stat(),
                        "cid",
                        old,
                        "127.0.0.1",
                    ),
                ),
                patch.object(
                    m6,
                    "stop_registrar_if_present",
                ),
                patch.object(
                    m6,
                    "reload_confirm",
                    side_effect=RuntimeError(
                        "timeout"
                    ),
                ),
                patch.object(
                    m6,
                    "start_registrar",
                ),
            ):
                result = m6.cutover(
                    path,
                    Path("/bin/true"),
                    "c",
                    "image",
                    Path(d),
                )

                self.assertEqual(result, 2)
                self.assertEqual(
                    json.loads(path.read_text()),
                    old,
                )


if __name__ == "__main__":
    unittest.main(verbosity=2)
