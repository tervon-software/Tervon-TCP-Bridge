#!/usr/bin/env python3
"""Apply the focused M6 Trojan gRPC correction to installed Android sources."""
import argparse
import hashlib
import os
from pathlib import Path
import tempfile

DART = Path("lib/vpn/sub_parser.dart")
KOTLIN = Path("android/app/src/main/kotlin/com/tervonvpndpi/app/TrojanOutboundParser.kt")
STAGED_KOTLIN = Path("integrations/android-ingress/stage4/TrojanOutboundParser.kt")

DART_BEFORE = "1c92557a5bd7db7373c80c107ed6b1ef4295c055"
KOTLIN_BEFORE = "34642fe5363db16ad0198b6a0a75ed70f562a2fd"
KOTLIN_AFTER = "a6a61898483d6f2dd6f33999fb477c92b56080c6"

DART_GUARD = "      if (normalized == 'grpc' && (params['servicename'] ?? '').isEmpty) return null;\n"

def blob(content: bytes) -> str:
    return hashlib.sha1(b"blob " + str(len(content)).encode() + b"\0" + content).hexdigest()

def atomic(path: Path, data: bytes, mode: int) -> None:
    fd, name = tempfile.mkstemp(prefix=".m6-grpc-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as out:
            out.write(data)
            out.flush()
            os.fsync(out.fileno())
        os.chmod(name, mode)
        os.replace(name, path)
    finally:
        if os.path.exists(name):
            os.unlink(name)

def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo-root", type=Path, required=True)
    parser.add_argument("--app-root", type=Path, required=True)
    flags = parser.add_mutually_exclusive_group(required=True)
    flags.add_argument("--check", action="store_true")
    flags.add_argument("--apply", action="store_true")
    a = parser.parse_args()
    repo = a.repo_root.resolve()
    app = a.app_root.resolve()
    dart = app / DART
    kotlin = app / KOTLIN
    staged = repo / STAGED_KOTLIN
    if any(not p.is_file() or p.is_symlink() for p in (dart, kotlin, staged)):
        print("M6_GRPC_FIX=BLOCKED_MISSING_SOURCE")
        return 1
    new_kotlin = staged.read_bytes()
    if blob(new_kotlin) != KOTLIN_AFTER:
        print("M6_GRPC_FIX=BLOCKED_GITHUB_KOTLIN_SHA")
        return 1
    old_dart, old_kotlin = dart.read_bytes(), kotlin.read_bytes()
    if blob(old_kotlin) not in (KOTLIN_BEFORE, KOTLIN_AFTER):
        print("M6_GRPC_FIX=BLOCKED_APP_KOTLIN_SHA")
        return 1
    if blob(old_dart) == DART_BEFORE:
        text = old_dart.decode("utf-8")
        if text.count(DART_GUARD) != 1:
            print("M6_GRPC_FIX=BLOCKED_DART_ANCHOR")
            return 1
        new_dart = text.replace(DART_GUARD, "", 1).encode("utf-8")
    elif DART_GUARD.encode() not in old_dart and b"'trojan_grpc'" in old_dart:
        new_dart = old_dart
    else:
        print("M6_GRPC_FIX=BLOCKED_APP_DART_SOURCE")
        return 1
    changes = [(dart, new_dart), (kotlin, new_kotlin)]
    pending = [(p, data) for p, data in changes if p.read_bytes() != data]
    print("M6_GRPC_FIX_PREFLIGHT=PASS")
    print("M6_GRPC_FILES_TO_UPDATE=" + str(len(pending)))
    if a.check:
        return 0
    originals = {p: (p.read_bytes(), p.stat().st_mode & 0o777) for p, _ in pending}
    installed = []
    try:
        for path, data in pending:
            atomic(path, data, originals[path][1])
            installed.append(path)
    except OSError:
        for path in reversed(installed):
            try:
                original, mode = originals[path]
                atomic(path, original, mode)
            except OSError:
                print("M6_GRPC_FIX=ROLLBACK_INCOMPLETE")
                return 1
        print("M6_GRPC_FIX=WRITE_FAILED")
        return 1
    print("M6_GRPC_FIX=SOURCE_APPLIED")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
