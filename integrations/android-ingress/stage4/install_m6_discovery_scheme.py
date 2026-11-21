#!/usr/bin/env python3
"""M6 source-only: transfer reviewed Android HTTPS discovery, no APK or services.
Port is determined by platform HTTPS scheme, never by an explicit numeric port.
"""
import argparse
import hashlib
import os
import tempfile
from pathlib import Path

SOURCE = Path("integrations/android-ingress/stage4/BridgeAutoEnrollmentHttp.kt")
TARGET = Path("android/app/src/main/kotlin/com/tervonvpndpi/app/BridgeAutoEnrollmentHttp.kt")
BEFORE = "d93a3f78e35d4767e47fda1edeb3822fbb73ed7f"
AFTER = "61a0f45fc7fd8070d197ebce268730c051490311"

def blob(raw):
    return hashlib.sha1(b"blob " + str(len(raw)).encode() + b"\0" + raw).hexdigest()

def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--repo-root", type=Path, required=True)
    p.add_argument("--app-root", type=Path, required=True)
    group = p.add_mutually_exclusive_group(required=True)
    group.add_argument("--check", action="store_true")
    group.add_argument("--apply", action="store_true")
    args = p.parse_args()
    src = args.repo_root.resolve() / SOURCE
    dst = args.app_root.resolve() / TARGET
    if any(not f.is_file() or f.is_symlink() for f in (src, dst)):
        print("M6_DISCOVERY_SOURCE=BLOCKED_MISSING_FILE")
        return 1
    new, old = src.read_bytes(), dst.read_bytes()
    if blob(new) != AFTER or blob(old) not in (BEFORE, AFTER):
        print("M6_DISCOVERY_SOURCE=BLOCKED_SHA_MISMATCH")
        return 1
    print("M6_DISCOVERY_SOURCE_PREFLIGHT=PASS")
    if args.check or old == new:
        print("M6_DISCOVERY_CHANGED=NO")
        return 0
    fd, temp = tempfile.mkstemp(prefix=".m6-discovery-", dir=dst.parent)
    try:
        with os.fdopen(fd, "wb") as f:
            f.write(new)
            f.flush()
            os.fsync(f.fileno())
        os.chmod(temp, dst.stat().st_mode & 0o777)
        os.replace(temp, dst)
    finally:
        if os.path.exists(temp):
            os.unlink(temp)
    print("M6_DISCOVERY_SOURCE_APPLIED=PASS")
    print("LIVE_SERVICES=UNCHANGED")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
