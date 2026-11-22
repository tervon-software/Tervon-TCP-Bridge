#!/usr/bin/env python3
"""Isolated real-process RuntimePolicy ACK test of the verified Linux v1.0.2 candidate.
No production config, Docker, systemd, running service, or source change.
Only signals the child started by this exact script.
"""
import hashlib
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import tempfile
import time

BIN = Path("/opt/tervon/Developed/v1.0.2-runtime-policy-validation/tervon-bridge-linux-amd64-v1.0.2-candidate")
SHA256 = "aea59080e0a112e9d5c079d45667ed8cb9b06e126326a398ea33dd94fb0c1371"
STAGING = Path("/opt/tervon/Developed")
TIMEOUT = 8.0


class CheckFailed(Exception):
    pass


def ensure(condition, reason):
    if not condition:
        raise CheckFailed(reason)


def policy_digest(routes, authorized):
    # JSON layout matches Go struct field order: routes then authorized.
    # Backend field order: Name then Address. Map keys sorted as Go.
    normalized = {
        service: [{"Name": b["Name"], "Address": b["Address"]} for b in routes[service]]
        for service in sorted(routes)
    }
    auth = {fingerprint: authorized[fingerprint] for fingerprint in sorted(authorized)}
    data = json.dumps({"routes": normalized, "authorized": auth},
                      ensure_ascii=False, separators=(",", ":")).encode("utf-8")
    return hashlib.sha256(data).hexdigest()


def write_config(path, config):
    pending = path.with_name("server.json.next")
    pending.write_text(json.dumps(config, separators=(",", ":")), encoding="utf-8")
    os.chmod(pending, 0o600)
    os.replace(pending, path)


def lines(logfile):
    return logfile.read_text(encoding="utf-8", errors="replace")


def await_marker(logfile, marker, process, previous=0):
    until = time.monotonic() + TIMEOUT
    while time.monotonic() < until:
        current = lines(logfile)
        if current.count(marker) > previous:
            return
        if process.poll() is not None:
            raise CheckFailed("TEST_PROCESS_EXITED")
        time.sleep(0.04)
    raise CheckFailed("EXPECTED_ACK_OR_REJECTION_NOT_OBSERVED")


def await_listener(address, process):
    until = time.monotonic() + TIMEOUT
    while time.monotonic() < until:
        if process.poll() is not None:
            raise CheckFailed("TEST_PROCESS_EXITED_BEFORE_READY")
        try:
            with socket.create_connection(address, timeout=0.25):
                return
        except OSError:
            time.sleep(0.04)
    raise CheckFailed("TEST_LISTENER_NOT_READY")


def test():
    ensure(BIN.is_file(), "CANDIDATE_MISSING")
    h = hashlib.sha256()
    with BIN.open("rb") as stream:
        while block := stream.read(1 << 20):
            h.update(block)
    ensure(h.hexdigest() == SHA256, "CANDIDATE_SHA256_MISMATCH")
    ensure(STAGING.is_dir(), "STAGING_MISSING")
    print("CANDIDATE_SHA256=PASS", flush=True)

    with tempfile.TemporaryDirectory(prefix="v102-policy-runtime-", dir=STAGING) as tmp:
        root = Path(tmp)
        os.chmod(root, 0o700)
        cert, key = root / "server.crt", root / "server.key"
        # Ephemeral OpenSSL local-test identity; use original Go TLS implementation.
        subprocess.run([
            "openssl", "req", "-x509", "-newkey", "ec",
            "-pkeyopt", "ec_paramgen_curve:P-256", "-nodes",
            "-days", "1", "-subj", "/CN=localhost",
            "-addext", "subjectAltName=DNS:localhost",
            "-keyout", str(key), "-out", str(cert),
        ], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=15)
        os.chmod(key, 0o600)

        # Loopback only, temporary kernel-chosen port.
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as probe:
            probe.bind(("127.0.0.1", 0))
            port = probe.getsockname()[1]
        listen = f"127.0.0.1:{port}"

        routes_a = {"vpn": [{"Name": "initial", "Address": "127.0.0.1:11111"}]}
        routes_b = {"vpn-new": [{"Name": "replaced", "Address": "127.0.0.1:22222"}]}
        auth_a = {"a" * 64: ["vpn"]}
        auth_b = {"b" * 64: ["vpn-new"]}
        config = {
            "mode": "server", "listen": listen,
            "cert_file": str(cert), "key_file": str(key), "ca_file": str(cert),
            "max_streams": 0, "routes": routes_a, "authorized": auth_a,
        }
        cfgfile = root / "server.json"
        logpath = root / "candidate.log"
        write_config(cfgfile, config)
        with logpath.open("w+", encoding="utf-8") as log:
            process = subprocess.Popen([str(BIN), "-config", str(cfgfile)],
                                       stdin=subprocess.DEVNULL, stdout=log,
                                       stderr=subprocess.STDOUT, cwd=str(root))
            try:
                await_listener(("127.0.0.1", port), process)
                ensure(process.poll() is None, "TEST_PROCESS_NOT_RUNNING")
                print("ISOLATED_PROCESS=PASS", flush=True)

                config_b = dict(config, routes=routes_b, authorized=auth_b)
                write_config(cfgfile, config_b)
                expected_b = policy_digest(routes_b, auth_b)
                success_marker = "M6_POLICY_RELOAD=PASS digest="
                process.send_signal(signal.SIGHUP)
                await_marker(logpath, success_marker + expected_b, process)
                ensure(lines(logpath).count(success_marker) == 1, "UNEXPECTED_ACK_COUNT_AFTER_SWAP")
                print("ROUTES_AND_ACL_RELOAD=PASS", flush=True)
                print("EXACT_GENERATION_ACK=PASS", flush=True)

                # The strict load() checks missing service references before
                # RuntimePolicy.Replace(). Expect the correct first-stage rejection.
                wrong = dict(config_b, authorized=auth_a)
                write_config(cfgfile, wrong)
                process.send_signal(signal.SIGHUP)
                await_marker(logpath, "M6_POLICY_RELOAD=FAIL invalid config", process)
                ensure(lines(logpath).count(success_marker) == 1, "INCONSISTENT_POLICY_WAS_ACKNOWLEDGED")
                print("INCONSISTENT_POLICY_REJECTED_BY_LOADER=PASS", flush=True)

                # Both hex spellings decode to the same fingerprint. load()
                # accepts both keys individually, but RuntimePolicy rejects
                # the duplicate identity before publication.
                duplicate_acl = {
                    "b" * 64: ["vpn-new"],
                    "B" * 64: ["vpn-new"],
                }
                wrong_runtime = dict(config_b, authorized=duplicate_acl)
                write_config(cfgfile, wrong_runtime)
                process.send_signal(signal.SIGHUP)
                await_marker(logpath, "M6_POLICY_RELOAD=FAIL invalid routes or ACL", process)
                ensure(lines(logpath).count(success_marker) == 1, "INVALID_RUNTIME_POLICY_WAS_ACKNOWLEDGED")
                print("INVALID_POLICY_REJECTED_BY_CORE=PASS", flush=True)

                static_change = dict(config_b, max_streams=1)
                write_config(cfgfile, static_change)
                process.send_signal(signal.SIGHUP)
                await_marker(logpath, "M6_POLICY_RELOAD=FAIL non-policy configuration changed", process)
                ensure(lines(logpath).count(success_marker) == 1, "STATIC_CONFIG_WAS_ACKNOWLEDGED")
                print("NON_POLICY_CHANGE_REJECTED=PASS", flush=True)

                invalid_config_marker = "M6_POLICY_RELOAD=FAIL invalid config"
                previously_invalid = lines(logpath).count(invalid_config_marker)
                cfgfile.write_text("{", encoding="utf-8")
                process.send_signal(signal.SIGHUP)
                await_marker(logpath, invalid_config_marker, process,
                             previous=previously_invalid)
                ensure(lines(logpath).count(success_marker) == 1, "MALFORMED_CONFIG_WAS_ACKNOWLEDGED")
                print("MALFORMED_CONFIG_REJECTED=PASS", flush=True)

                write_config(cfgfile, config)
                expected_a = policy_digest(routes_a, auth_a)
                process.send_signal(signal.SIGHUP)
                await_marker(logpath, success_marker + expected_a, process)
                ensure(lines(logpath).count(success_marker) == 2, "RESTORE_ACK_COUNT_INVALID")
                ensure(process.poll() is None, "PROCESS_EXITED_AFTER_RELOAD")
                print("RESTORE_GENERATION_ACK=PASS", flush=True)
                print("PROCESS_ALIVE_AFTER_RELOAD=PASS", flush=True)
            finally:
                if process.poll() is None:
                    process.terminate()
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=5)
    print("ISOLATED_PROCESS_CLEANUP=PASS")
    print("LINUX_V1_0_2_RUNTIME_ACK=PASS_ISOLATED_PROCESS")
    print("PRODUCTION=UNCHANGED")


if __name__ == "__main__":
    try:
        test()
    except (CheckFailed, OSError, subprocess.CalledProcessError,
            subprocess.TimeoutExpired, ValueError) as exc:
        reason = str(exc)
        if not isinstance(exc, CheckFailed):
            reason = type(exc).__name__
        print("LINUX_V1_0_2_RUNTIME_ACK=FAIL", flush=True)
        print("REASON=" + reason, flush=True)
        sys.exit(1)
