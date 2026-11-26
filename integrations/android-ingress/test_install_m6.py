import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("install_m6", Path(__file__).with_name("install-m6.py"))
installer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(installer)


class InstallM6Test(unittest.TestCase):
    def test_nginx_selects_exact_https_vhost_and_preserves_redirect(self):
        text = """
server { listen 443 ssl; server_name sub.example.net; location / { return 204; } }
server { listen 9443 ssl; server_name sub.example.net; return 302 "https://web.example.net/#page"; }
"""
        result = installer.stage_nginx(text, "sub.example.net", 9443, 18763)
        self.assertIn("location / { return 302 \"https://web.example.net/#page\"; }", result)
        self.assertEqual(result.split("server {", 2)[1], text.split("server {", 2)[1])
        self.assertEqual(result.count(installer.DISCOVERY), 1)
        with self.assertRaises(ValueError):
            installer.stage_nginx(text + text, "sub.example.net", 9443, 18763)

    def test_install_modes_generate_matching_units(self):
        for mode in ("native", "docker"):
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as temp:
                self.check_mode(mode, Path(temp))

    def check_mode(self, mode, temp):
        root, app, units, source = (temp / item for item in ("config", "app", "units", "source"))
        units.mkdir()
        source.mkdir()
        for name in installer.PROBE_FILES:
            (source / name).touch()
        vhost = temp / "nginx.conf"
        vhost.write_text("server { listen 9443 ssl; server_name sub.example.net; }")
        binary = b"fixture"
        sha = hashlib.sha256(binary).hexdigest()
        calls = []
        answers = [
            mode, "sub.example.net", "9443", "18763", "bridge.example.net", "18443",
            "3x-ui", "https://panel.example.net:5588/prefix", "yes", "x-ui.service",
            "xray-linux-amd64",
        ]
        if mode == "native":
            answers.append("127.0.0.1")
        answers.extend((str(vhost), "9443", "yes"))

        def invoke(args, **kwargs):
            args = list(map(str, args))
            calls.append(args)
            cmd = args[0]
            if args[:2] == ["systemctl", "show"]:
                return subprocess.CompletedProcess(args, 1, "not-found\n")
            if args[:3] == ["docker", "image", "inspect"]:
                return subprocess.CompletedProcess(args, 1)
            if cmd == "curl" and "--output" in args:
                Path(args[args.index("--output") + 1]).write_bytes(binary)
            elif cmd == "curl":
                self.assertEqual(args[-1], "https://sub.example.net:9443/.well-known/tervon-bridge")
                return subprocess.CompletedProcess(args, 0, json.dumps({
                    "available": True, "remote_url": "https://bridge.example.net:18443/v2/stream",
                    "server_ca": "fixture-ca",
                }))
            elif cmd == "go":
                Path(args[args.index("-o") + 1]).write_bytes(binary)
            elif cmd == "bash":
                self.assertEqual(kwargs["env"]["M6_EMPTY_POLICY"], "1")
                (root / "config").mkdir(parents=True)
                (root / "private").mkdir()
                (root / "private/server-ca.crt").write_text("fixture-ca")
                (root / "config/server.json").write_text(json.dumps({
                    "listen": "0.0.0.0:8443", "cert_file": "/config/pki/server.crt",
                    "key_file": "/config/pki/server.key", "ca_file": "/config/pki/client-ca.crt",
                }))
            return subprocess.CompletedProcess(args, 0, "")

        with (
            patch.multiple(installer, ROOT=root, APP=app, UNITS=units, SOURCE=source, SHA256=sha),
            patch.object(sys, "argv", ["install.sh"]),
            patch.object(installer.os, "geteuid", return_value=0),
            patch.object(installer.os, "chown"),
            patch.object(installer.os, "umask"),
            patch.object(installer.shutil, "which", side_effect=lambda cmd: "/usr/bin/" + cmd),
            patch.object(installer.getpass, "getpass", return_value="fixture-secret"),
            patch.object(installer, "free_listener"),
            patch.object(installer.subprocess, "run", side_effect=invoke),
            patch("builtins.input", side_effect=answers),
        ):
            self.assertEqual(installer.main(), 0)
        registrar = (units / installer.REGISTRAR).read_text()
        self.assertIn(f"--deployment={mode}", registrar)
        self.assertNotIn("fixture-secret", registrar)
        self.assertEqual((root / "private/panel.header").read_text(), "Authorization: Bearer fixture-secret\n")
        self.assertIn("location = /.well-known/tervon-bridge", vhost.read_text())
        if mode == "native":
            self.assertFalse(any(row[0] == "docker" for row in calls))
            self.assertIn("--bridge-public-port=18443", registrar)
            conf = json.loads((root / "config/server.json").read_text())
            self.assertEqual(conf["listen"], "0.0.0.0:18443")
            self.assertEqual(conf["cert_file"], str(root / "config/pki/server.crt"))
        else:
            self.assertIn(f"--container-name={installer.CONTAINER}", registrar)
            service = json.loads((app / "compose.json").read_text())["services"]["decapsulator"]
            self.assertEqual(service["ports"], ["0.0.0.0:18443:8443/tcp"])


if __name__ == "__main__":
    unittest.main()
