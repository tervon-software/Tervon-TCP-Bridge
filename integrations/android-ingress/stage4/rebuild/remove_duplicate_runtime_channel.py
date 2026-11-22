#!/usr/bin/env python3
"""Guarded source-only elimination of a duplicate Android Bridge process control channel."""
import argparse
import hashlib
from pathlib import Path
import sys

EXPECTED = {
    "BridgeRuntimeControl.kt": "a6f9695d4cdc6a3d120ac0f685ddd6f2d9ed4a122469e13135e542dbfacdc596",
    "BridgeControlChannel.kt": "0e503518ef49e860e0a48be2b25f68dea3705d39d9602e936604c487af750eef",
}

RUNTIME = """package com.tervonvpndpi.app

import android.content.Context

/** One internal Go process owner. Only BridgeVpnSession may initiate lifecycle. */
internal object BridgeRuntimeControl {
    private var manager: BridgeCoreManager? = null

    @Synchronized
    private fun core(context: Context): BridgeCoreManager =
        manager ?: BridgeCoreManager(context.applicationContext).also { manager = it }

    @Synchronized
    internal fun coreState(): BridgeCoreManager.State =
        manager?.state() ?: BridgeCoreManager.State.STOPPED

    @Synchronized
    internal fun launchClient(context: Context): Boolean {
        val env = BridgeConfigManager(context)
        return core(context).launch(listOf("-config", env.configFile.absolutePath))
    }

    @Synchronized
    internal fun stopClient(context: Context) = core(context).stop()

    @Synchronized
    internal fun onUnexpectedExit(context: Context, callback: (() -> Unit)?) {
        core(context).onUnexpectedExit = callback
    }
}
"""

def sha(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()

def build(root: Path) -> dict[Path, bytes]:
    src = {name: (root / name).read_bytes() for name in EXPECTED}
    for name, expected in EXPECTED.items():
        if sha(src[name]) != expected:
            raise ValueError(f"{name}: source SHA256 mismatch; no changes made")
    channel = src["BridgeControlChannel.kt"].decode("utf-8")
    line = "        BridgeRuntimeControl.register(messenger, app)\n"
    if channel.count(line) != 1:
        raise ValueError("BridgeControlChannel.register anchor mismatch; no changes made")
    channel = channel.replace(line, "", 1)
    if "BridgeRuntimeControl.register(" in channel:
        raise ValueError("duplicate runtime register remains")
    return {
        root / "BridgeRuntimeControl.kt": RUNTIME.encode("utf-8"),
        root / "BridgeControlChannel.kt": channel.encode("utf-8"),
    }

def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--app-kotlin-dir", type=Path, required=True)
    p.add_argument("--check", action="store_true")
    p.add_argument("--apply", action="store_true")
    args = p.parse_args()
    if args.check == args.apply:
        p.error("choose exactly one of --check / --apply")
    root = args.app_kotlin_dir.resolve(strict=True)
    if not root.is_dir() or root.name != "app":
        raise ValueError("expected Kotlin package directory named app")
    new = build(root)
    for path, data in new.items():
        print(f"VERIFIED {path.name} {sha(data)}")
    if args.apply:
        for path, data in new.items():
            path.write_bytes(data)
        print("M6_DUPLICATE_RUNTIME_CHANNEL=REMOVED_SOURCE_ONLY")
    else:
        print("M6_DUPLICATE_RUNTIME_CHANNEL=CHECK_PASS_NO_WRITE")
    return 0

if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError) as exc:
        print(f"BLOCKED: {exc}", file=sys.stderr)
        sys.exit(1)
