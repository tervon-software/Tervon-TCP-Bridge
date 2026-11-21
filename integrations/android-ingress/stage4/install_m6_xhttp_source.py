#!/usr/bin/env python3
"""Guarded M6 Android XHTTP source update; no build, signing or service action."""
import argparse
import hashlib
import os
from pathlib import Path
import tempfile

SOURCE_DIR = Path("integrations/android-ingress/stage4")
TARGET_KT = Path("android/app/src/main/kotlin/com/tervonvpndpi/app")
TARGET_DART = Path("lib/features/home")
# Git blob SHA-1 of the immediately preceding reviewed Android sources and
# of the committed new versions. No wildcard or implicit rewrite is allowed.
REVISIONS = {
    "BridgeVpnSession.kt": (
        "16eab42b49173f55e2d997f7c04d6edb2b58e527",
        "338fa4864c0e898127a1f2e0bc93e1266990ab98",
    ),
    "BridgeAutoEnrollmentHttp.kt": (
        "c2a3d6d0d29046ee5033dac23cbba580627448e1",
        "c136e8da473b2ae17c92a33dd0dd98a94d3f7f7c",
    ),
    "BridgeVlessRoutePlan.kt": (
        "cc2162eac04f456e75eb9a64b2b9466bb1512e97",
        "a73a5ef5797c7778e24635a8fbbc5ae75e637f33",
    ),
    "BridgeXrayConfigRoute.kt": (
        "a0f4e404028b42a2efbbaa309bbbe9c18c57fd2e",
        "c97f107e361f2573cf11ca0e233b32ef16ed1b73",
    ),
    "bridge_settings_screen.dart": (
        "9ce2ab30b225513a95d9b60ccc3726a201e1f58b",
        "7f99e3c75c94c818b34c97762de2be7fa27fa06f",
    ),
}


def blob(data: bytes) -> str:
    return hashlib.sha1(b"blob " + str(len(data)).encode() + b"\0" + data).hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo-root", type=Path, required=True)
    parser.add_argument("--app-root", type=Path, required=True)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    repo, app = args.repo_root.resolve(), args.app_root.resolve()
    if not (app / "pubspec.yaml").is_file():
        print("M6_XHTTP=BLOCKED_NO_ANDROID_PROJECT")
        return 1
    pending = []
    for name, (before, after) in REVISIONS.items():
        src = repo / SOURCE_DIR / name
        dst = app / (TARGET_DART if name.endswith(".dart") else TARGET_KT) / name
        if (not src.is_file() or not dst.is_file() or src.is_symlink() or dst.is_symlink()
                or blob(src.read_bytes()) != after):
            print("M6_XHTTP=BLOCKED_SOURCE " + name)
            return 1
        current = blob(dst.read_bytes())
        if current not in (before, after):
            print("M6_XHTTP=BLOCKED_UNEXPECTED_TARGET " + name)
            return 1
        if current != after:
            pending.append((src, dst))
    print("M6_XHTTP_SOURCE_PREFLIGHT=PASS")
    print("M6_CHANGED_FILES=" + str(len(pending)))
    if not pending:
        print("M6_XHTTP=ALREADY_APPLIED")
        return 0
    if args.check:
        print("M6_ANDROID_SOURCES_UNCHANGED=YES")
        return 0
    originals = {}
    installed = []
    try:
        for src, dst in pending:
            originals[dst] = dst.read_bytes()
            fd, temporary = tempfile.mkstemp(prefix=".m6-xhttp-", dir=dst.parent)
            try:
                with os.fdopen(fd, "wb") as output:
                    output.write(src.read_bytes())
                    output.flush()
                    os.fsync(output.fileno())
                os.chmod(temporary, dst.stat().st_mode & 0o777)
                os.replace(temporary, dst)
                installed.append(dst)
            finally:
                if os.path.exists(temporary):
                    os.unlink(temporary)
    except OSError:
        for dst in reversed(installed):
            dst.write_bytes(originals[dst])
        print("M6_XHTTP=WRITE_FAILED_ROLLBACK_ATTEMPTED")
        return 1
    print("M6_XHTTP=ANDROID_SOURCE_APPLIED")
    print("M6_BUILD=NOT_RUN")
    print("LIVE_SERVICES=UNCHANGED")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
