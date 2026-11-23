#!/usr/bin/env python3
"""M6 controlled policy cutover. Live changes require explicit --apply."""

import argparse
import copy
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import tempfile
import time
import urllib.request

REG = "tervon-m6-auto-registrar.service"
SERVICE = "vpn-55535"
PORT = 55535


def command(*args):
    result = subprocess.run(
        args,
        check=True,
        capture_output=True,
        text=True,
    )

    if args[:2] == ("docker", "logs"):
        return "\n".join(
            part for part in (result.stdout, result.stderr)
            if part
        ).strip()

    return result.stdout.strip()


def generation_digest(config):
    routes = {
        key: [
            {"Name": b["Name"], "Address": b["Address"]}
            for b in config["routes"][key]
        ]
        for key in sorted(config["routes"])
    }

    authorized = {
        key: config["authorized"][key]
        for key in sorted(config["authorized"])
    }

    raw = json.dumps(
        {"routes": routes, "authorized": authorized},
        separators=(",", ":"),
        ensure_ascii=False,
    ).encode()

    return hashlib.sha256(raw).hexdigest()


def ack_in_logs(output, issued, expected):
    marker = "M6_POLICY_RELOAD=PASS digest=" + expected

    for line in output.splitlines():
        timestamp, separator, message = line.partition(" ")

        if not separator:
            continue

        try:
            observed = dt.datetime.fromisoformat(
                timestamp.replace("Z", "+00:00")
            )
        except ValueError:
            continue

        if observed.tzinfo is None or observed < issued:
            continue

        message = message.strip()

        if message == marker:
            return True

        match = re.fullmatch(
            r"\d{4}/\d{2}/\d{2} "
            r"\d{2}:\d{2}:\d{2} (.*)",
            message,
        )

        if match and match.group(1) == marker:
            return True

    return False


def reload_confirm(container_id, expected, timeout=12):
    issued = (
        dt.datetime.now(dt.timezone.utc)
        - dt.timedelta(milliseconds=100)
    )

    command(
        "docker", "kill",
        "--signal=HUP", container_id,
    )

    deadline = time.monotonic() + timeout
    seen = 0

    while True:
        output = command(
            "docker", "logs",
            "--timestamps",
            "--since", issued.isoformat(),
            container_id,
        )

        seen = len(output.splitlines())

        if ack_in_logs(output, issued, expected):
            return

        if time.monotonic() >= deadline:
            raise RuntimeError(
                "EXACT_POLICY_ACK_NOT_RECEIVED "
                f"timeout_seconds={timeout} "
                f"log_lines={seen}"
            )

        time.sleep(0.25)


def atomic_write(path, data, info):
    fd, temp = tempfile.mkstemp(
        prefix=".m6-policy-",
        dir=str(path.parent),
    )

    try:
        with os.fdopen(fd, "wb") as f:
            encoded = (
                json.dumps(
                    data,
                    indent=2,
                    ensure_ascii=False,
                ) + "\n"
            ).encode()

            f.write(encoded)
            f.flush()

            os.fchmod(
                f.fileno(),
                stat.S_IMODE(info.st_mode),
            )
            os.fchown(
                f.fileno(),
                info.st_uid,
                info.st_gid,
            )
            os.fsync(f.fileno())

        os.replace(temp, path)

        directory = os.open(
            path.parent,
            os.O_RDONLY,
        )

        try:
            os.fsync(directory)
        finally:
            os.close(directory)

    finally:
        if os.path.exists(temp):
            os.unlink(temp)


def stop_registrar_if_present():
    state = command(
        "systemctl", "show",
        "-p", "LoadState",
        "--value", REG,
    )

    if state != "not-found":
        command("systemctl", "stop", REG)


def start_registrar(binary, container_id, image, root):
    stop_registrar_if_present()

    subprocess.run(
        ["systemctl", "reset-failed", REG],
        check=False,
        capture_output=True,
    )

    for _ in range(60):
        state = command(
            "systemctl", "show",
            "-p", "LoadState",
            "--value", REG,
        )

        if state == "not-found":
            break

        time.sleep(0.2)
    else:
        raise RuntimeError(
            "REGISTRAR_UNIT_NOT_RELEASED"
        )

    command(
        "systemd-run",
        "--unit=tervon-m6-auto-registrar",
        "--collect",
        str(binary),
        "--root", str(root),
        "--listen", "127.0.0.1:18763",
        "--container-id", container_id,
        "--expected-image", image,
        "--expected-label-key",
        "com.docker.compose.service",
        "--expected-label-value",
        "android-bridge-decapsulator",
    )

    for _ in range(40):
        try:
            with urllib.request.urlopen(
                "http://127.0.0.1:18763/"
                ".well-known/tervon-bridge",
                timeout=2,
            ) as response:
                if response.status == 200:
                    return
        except (OSError, TimeoutError):
            pass

        time.sleep(0.25)

    raise RuntimeError("REGISTRAR_NOT_READY")


def preflight(path, binary, container_name, image):
    info = path.lstat()

    if not stat.S_ISREG(info.st_mode):
        raise RuntimeError("CONFIG_NOT_REGULAR")

    if not binary.is_file() or not os.access(
        binary, os.X_OK
    ):
        raise RuntimeError(
            "VERIFIED_REGISTRAR_BINARY_MISSING"
        )

    container = json.loads(
        command("docker", "inspect", container_name)
    )[0]

    if (
        container["Config"]["Image"] != image
        or not container["State"]["Running"]
        or container["Config"]["Labels"].get(
            "com.docker.compose.service"
        ) != "android-bridge-decapsulator"
    ):
        raise RuntimeError(
            "CONTAINER_IDENTITY_MISMATCH"
        )

    mounted = any(
        m.get("Type") == "bind"
        and m.get("Source") == str(path.parent)
        and m.get("Destination") == "/config"
        and not m.get("RW")
        for m in container.get("Mounts", [])
    )

    if not mounted:
        raise RuntimeError("CONFIG_MOUNT_MISMATCH")

    config = json.loads(path.read_text())

    if (
        config.get("mode") != "server"
        or set(config["routes"])
        != {"vpn", "vpn-xhttp"}
    ):
        raise RuntimeError("UNEXPECTED_ROUTE_SET")

    acl = config["authorized"]

    if len(acl) != 3 or not all(
        isinstance(fp, str)
        and re.fullmatch("[0-9a-f]{64}", fp)
        and isinstance(allowed, list)
        and len(allowed) == 2
        and set(allowed) == {"vpn", "vpn-xhttp"}
        for fp, allowed in acl.items()
    ):
        raise RuntimeError(
            "UNEXPECTED_ACL_STRUCTURE"
        )

    hosts = []

    for service, port in (
        ("vpn", 30000),
        ("vpn-xhttp", 55636),
    ):
        items = config["routes"][service]

        if (
            len(items) != 1
            or set(items[0]) != {"Name", "Address"}
        ):
            raise RuntimeError(
                "UNEXPECTED_BACKEND_STRUCTURE"
            )

        host, separator, number = (
            items[0]["Address"].rpartition(":")
        )

        if (
            not separator
            or not host
            or number != str(port)
        ):
            raise RuntimeError(
                "UNEXPECTED_BACKEND_PORT"
            )

        hosts.append(host)

    if hosts[0] != hosts[1]:
        raise RuntimeError(
            "BACKEND_HOST_AMBIGUOUS"
        )

    state = command(
        "systemctl", "show",
        "-p", "ActiveState",
        "--value", REG,
    )

    if state != "active":
        raise RuntimeError(
            "REGISTRAR_NOT_ACTIVE"
        )

    return info, container["Id"], config, hosts[0]


def cutover(path, binary, container_name, image, root):
    info, cid, old, host = preflight(
        path, binary, container_name, image
    )

    new = copy.deepcopy(old)

    new["routes"][SERVICE] = [{
        "Name": SERVICE,
        "Address": f"{host}:{PORT}",
    }]

    for allowed in new["authorized"].values():
        allowed.append(SERVICE)

    old_digest = generation_digest(old)
    new_digest = generation_digest(new)

    print(
        "EXISTING_ROUTES=2 "
        "TARGET_ROUTES=3 "
        "ACL_IDENTITIES=3",
        flush=True,
    )
    print(
        "PREVIOUS_POLICY_DIGEST=" + old_digest,
        flush=True,
    )
    print(
        "TARGET_POLICY_DIGEST=" + new_digest,
        flush=True,
    )

    changed = False
    stopped = False

    try:
        stop_registrar_if_present()
        stopped = True

        atomic_write(path, new, info)
        changed = True

        reload_confirm(cid, new_digest)

        start_registrar(
            binary, cid, image, root
        )

        current = json.loads(path.read_text())

        if generation_digest(current) != new_digest:
            raise RuntimeError(
                "POLICY_CHANGED_AFTER_ACK"
            )

        print(
            "M6_POLICY_3_ROUTES=PASS "
            "M6_POLICY_DIGEST_ACK=PASS "
            "M6_REGISTRAR=PASS"
        )
        print("XHTTP_ANDROID_E2E=NOT_TESTED")

        return 0

    except Exception as error:
        print(
            "UPDATE=FAIL REASON=" + str(error),
            flush=True,
        )

        rollback_ok = True

        if changed:
            try:
                atomic_write(path, old, info)
                reload_confirm(cid, old_digest)
                print("POLICY_ROLLBACK=PASS")
            except Exception as error:
                rollback_ok = False
                print(
                    "POLICY_ROLLBACK=FAIL "
                    "ROLLBACK_REASON=" + str(error)
                )

        if stopped:
            try:
                start_registrar(
                    binary, cid, image, root
                )
                print("REGISTRAR_RESTORE=PASS")
            except Exception as error:
                rollback_ok = False
                print(
                    "REGISTRAR_RESTORE=FAIL "
                    "RESTORE_REASON=" + str(error)
                )

        return 1 if rollback_ok else 2


def main():
    parser = argparse.ArgumentParser(
        description=__doc__
    )

    parser.add_argument(
        "--apply",
        action="store_true",
        help="explicit live policy change",
    )

    args = parser.parse_args()

    if not args.apply:
        parser.error(
            "read-only by default; "
            "--apply requires separate approval"
        )

    return cutover(
        Path(
            "/etc/tervon/bridge-android/"
            "config/server.json"
        ),
        Path(
            "/opt/tervon/Developed/"
            "m6-dynamic-cutover-8cd9a042/"
            "m6-auto-registrar"
        ),
        "android-ingress-android-bridge-decapsulator-1",
        "tervon-bridge-android-ingress:"
        "m6-dynamic-e1326094",
        Path("/etc/tervon/bridge-android"),
    )


if __name__ == "__main__":
    raise SystemExit(main())
