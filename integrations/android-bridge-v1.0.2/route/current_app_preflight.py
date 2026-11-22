#!/usr/bin/env python3
"""Read-only, redacted audit of current TervonVPN Android Bridge integration.

Reports only relative paths, line numbers, SHA256, and Git state. Never prints
source lines, config values, credential material, URLs or device identifiers.
"""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import sys

TARGETS = (
    "BridgeVpnSession.kt",
    "BridgeXrayConfigRoute.kt",
    "BridgeVlessRoutePlan.kt",
    "BridgeRuntimeControl.kt",
    "BridgeControlChannel.kt",
    "BridgeCoreManager.kt",
    "BridgeConfigManager.kt",
)
# Only check call-sites, never dump source or secrets.
SYMBOLS = (
    "BridgeVpnSession", "BridgeXrayConfigRoute", "BridgeVlessRoutePlan",
    "BridgeRuntimeControl", "BridgeControlChannel", "BridgeCoreManager",
    "BridgeConfigManager", "BridgeProcessDiagnostics", "libtervonbridge.so",
)
EXPECTED_ELF_SHA256 = "a7f693981f111436cd0023ced47417b9f7f222df33a60c319c1a4b073acefccd"
MAX_FILE_SIZE = 2_000_000
MAX_FILES = 12000


def file_sha256(path):
    h = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(65536), b""):
            h.update(block)
    return h.hexdigest()


def under(root, path):
    return path.relative_to(root).as_posix()


def scan(root):
    android = root / "android"
    flutter = root / "lib"
    if not android.is_dir():
        raise ValueError("NOT_ANDROID_PROJECT_ROOT")
    ktroot = android / "app" / "src" / "main" / "kotlin"
    if not ktroot.is_dir():
        raise ValueError("KOTLIN_SOURCE_DIRECTORY_MISSING")
    all_kt = sorted(ktroot.rglob("*.kt"))
    if len(all_kt) > MAX_FILES:
        raise ValueError("KOTLIN_FILE_LIMIT_EXCEEDED")
    by_name = {name: [] for name in TARGETS}
    refs = {name: [] for name in SYMBOLS}
    candidates = all_kt
    if flutter.is_dir():
        candidates = all_kt + sorted(flutter.rglob("*.dart"))
    if len(candidates) > MAX_FILES:
        raise ValueError("SOURCE_FILE_LIMIT_EXCEEDED")
    for path in candidates:
        if path.is_symlink() or not path.is_file():
            raise ValueError("UNSAFE_SOURCE_ENTRY")
        rel = under(root, path)
        if path.name in by_name:
            by_name[path.name].append({"path": rel, "sha256": file_sha256(path)})
        # Read only bounded UTF-8 files; no actual source content leaves machine.
        if path.stat().st_size > MAX_FILE_SIZE:
            continue
        for number, line in enumerate(path.read_text(encoding="utf-8", errors="replace").splitlines(), 1):
            for name in SYMBOLS:
                if name in line:
                    refs[name].append({"path": rel, "line": number})
    elf = android / "app" / "src" / "main" / "jniLibs" / "arm64-v8a" / "libtervonbridge.so"
    if elf.is_symlink():
        raise ValueError("UNSAFE_CORE_SYMLINK")
    elf_sha = file_sha256(elf) if elf.is_file() else None
    git = {"branch": None, "head": None, "tracked_dirty": None}
    for key, cmd in {
        "branch": ["git", "-C", str(root), "branch", "--show-current"],
        "head": ["git", "-C", str(root), "rev-parse", "HEAD"],
        "tracked_dirty": ["git", "-C", str(root), "diff", "--quiet"],
    }.items():
        try:
            proc = subprocess.run(cmd, capture_output=True, text=True, timeout=5, check=False)
            if key == "tracked_dirty":
                git[key] = False if proc.returncode == 0 else (True if proc.returncode == 1 else None)
            elif proc.returncode == 0:
                git[key] = proc.stdout.strip() or None
        except (OSError, subprocess.TimeoutExpired):
            pass
    return {
        "schema": "tervon.android-bridge.current-app-preflight.v1",
        "mode": "READ_ONLY",
        "app_git": git,
        "seven_targets": by_name,
        "callsite_locations_only": refs,
        "arm64_core": {"present": elf_sha is not None, "sha256": elf_sha,
                         "matches_confirmed_diagnostic_core": elf_sha == EXPECTED_ELF_SHA256},
        "note": "No source text or credentials emitted; no files modified",
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--app-root", type=Path, required=True)
    args = parser.parse_args()
    root = args.app_root.resolve(strict=True)
    try:
        result = scan(root)
    except (OSError, ValueError) as exc:
        # Do not print raw exceptions which may contain sensitive file paths.
        print("CURRENT_APP_PREFLIGHT=BLOCKED", file=sys.stderr)
        return 2
    print(json.dumps(result, ensure_ascii=False, indent=2, sort_keys=True))
    print("CURRENT_APP_PREFLIGHT=PASS_READ_ONLY")
    return 0


if __name__ == "__main__":
    sys.exit(main())
