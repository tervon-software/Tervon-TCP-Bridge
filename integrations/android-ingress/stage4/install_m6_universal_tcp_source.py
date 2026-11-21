#!/usr/bin/env python3
"""Apply the approved M6 protocol-blind TCP route correction, source only.

The prior Trojan and XHTTP files are already installed. Verify exact Git
blobs of ALL inputs before any write. No APK build or live service changes.
"""
import argparse
import hashlib
import os
from pathlib import Path
import tempfile

SOURCE = Path("integrations/android-ingress/stage4")
KT = Path("android/app/src/main/kotlin/com/tervonvpndpi/app")
DART = Path("lib/features/home")

# Expected installed M6 XHTTP+Trojan source -> reviewed universal TCP source.
REVISIONS = {
    "BridgeVlessRoutePlan.kt": (
        "a73a5ef5797c7778e24635a8fbbc5ae75e637f33",
        "2decc9dba44d7e8295d26c69d9f623ee8e99de94",
    ),
    "BridgeVpnSession.kt": (
        "30b2a2f22b70dba3f363d4955940f19dfc669d59",
        "dc90c73170c7f7b579f52276db2042dd5444b92f",
    ),
    "BridgeXrayConfigRoute.kt": (
        "c97f107e361f2573cf11ca0e233b32ef16ed1b73",
        "1944e0f84de270e4d12d586cd60edcaa2794d608",
    ),
    "BridgeAutoEnrollmentHttp.kt": (
        "c136e8da473b2ae17c92a33dd0dd98a94d3f7f7c",
        "d93a3f78e35d4767e47fda1edeb3822fbb73ed7f",
    ),
    "bridge_settings_screen.dart": (
        "2b899818a8b2bc60e1fc1fa1e7b547fb26b04fa4",
        "5529269554fe005872e6194cce48875e45e82a81",
    ),
}

def blob(data: bytes) -> str:
    return hashlib.sha1(b"blob " + str(len(data)).encode() + b"\0" + data).hexdigest()

def atomic_write(path: Path, data: bytes, mode: int) -> None:
    fd, temp = tempfile.mkstemp(prefix=".m6-universal-tcp-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as f:
            f.write(data)
            f.flush()
            os.fsync(f.fileno())
        os.chmod(temp, mode)
        os.replace(temp, path)
    finally:
        if os.path.exists(temp):
            os.unlink(temp)

def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo-root", type=Path, required=True)
    parser.add_argument("--app-root", type=Path, required=True)
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--check", action="store_true")
    group.add_argument("--apply", action="store_true")
    a = parser.parse_args()
    repo, app = a.repo_root.resolve(), a.app_root.resolve()
    if not (app / "pubspec.yaml").is_file():
        print("M6_ANY_TCP=BLOCKED_NO_ANDROID_PROJECT")
        return 1
    pending = []
    for filename, (before, after) in REVISIONS.items():
        src = repo / SOURCE / filename
        dest = app / (DART if filename.endswith(".dart") else KT) / filename
        if src.is_symlink() or dest.is_symlink() or not src.is_file() or not dest.is_file():
            print("M6_ANY_TCP=BLOCKED_MISSING_OR_SYMLINK")
            return 1
        raw = src.read_bytes()
        if blob(raw) != after:
            print("M6_ANY_TCP=BLOCKED_SOURCE_SHA " + filename)
            return 1
        current = dest.read_bytes()
        if blob(current) not in (before, after):
            print("M6_ANY_TCP=BLOCKED_TARGET_SHA " + filename)
            return 1
        if current != raw:
            pending.append((dest, raw))
    print("M6_ANY_TCP_PREFLIGHT=PASS")
    print("M6_FILES_TO_UPDATE=" + str(len(pending)))
    if a.check:
        print("M6_APP_CHANGED=NO")
        return 0
    originals = {p: (p.read_bytes(), p.stat().st_mode & 0o777) for p, _ in pending}
    installed = []
    try:
        for dest, raw in pending:
            atomic_write(dest, raw, originals[dest][1])
            installed.append(dest)
    except OSError:
        for dest in reversed(installed):
            old, mode = originals[dest]
            try:
                atomic_write(dest, old, mode)
            except OSError:
                print("M6_ANY_TCP=ROLLBACK_INCOMPLETE")
                return 1
        print("M6_ANY_TCP=WRITE_FAILED_ROLLED_BACK")
        return 1
    print("M6_ANY_TCP_SOURCE_APPLIED=PASS")
    print("M6_ACTIVE_SERVICES=UNCHANGED")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
