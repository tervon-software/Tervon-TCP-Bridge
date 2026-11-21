#!/usr/bin/env python3
"""Guarded source-only Trojan integration for the existing TervonVPNDPI app.

No build, signing, APK publication, VPN process changes or server actions.
Existing VlessConfigParser is patched in-place rather than replaced.
"""
import argparse
import hashlib
import os
from pathlib import Path
import tempfile

KOTLIN = Path("android/app/src/main/kotlin/com/tervonvpndpi/app")
DART = Path("lib/vpn")
ORIGINAL = {
    KOTLIN / "VlessConfigParser.kt": ("4bb5f406ae0563a2ff84b635e4b86508f9d49085", "1977a9cdb61d5fdf7acc94c11183c94550ea985d"),
    DART / "sub_parser.dart": ("8d1d320868fab622effb193189a36a022d491ea0", "1c92557a5bd7db7373c80c107ed6b1ef4295c055"),
}
NEW_SOURCES = {
    KOTLIN / "TrojanOutboundParser.kt": "integrations/android-ingress/stage4/TrojanOutboundParser.kt",
    Path("test/trojan_subscription_test.dart"): "integrations/android-ingress/stage4/trojan_subscription_test.dart",
}
NEW_SOURCE_BLOBS = {
    KOTLIN / "TrojanOutboundParser.kt": "34642fe5363db16ad0198b6a0a75ed70f562a2fd",
    Path("test/trojan_subscription_test.dart"): "6ab113564fbca9f0580c1c51ec6f6c1e8a268e2f",
}

def blob(data):
    return hashlib.sha1(b"blob " + str(len(data)).encode() + b"\0" + data).hexdigest()

def single(source, old, new):
    if source.count(old) != 1:
        raise ValueError("expected exact one-time Android source insertion point")
    return source.replace(old, new, 1)

def transform_parser(source):
    source = single(source,
        '            "vless_tcp" -> find {',
        '''            "trojan", "trojan_tcp", "trojan_ws", "trojan_grpc", "trojan_xhttp" -> find {
                it.startsWith("trojan://", true) &&
                    (preferred == "trojan" ||
                     TrojanOutboundParser.transport(it) ==
                         preferred!!.removePrefix("trojan_"))
            }?.let { return it }
            "vless_tcp" -> find {''')
    source = single(source,
        '        find { it.startsWith("vless://", true) }?.let { return it }\n\n        return null',
        '''        find { it.startsWith("vless://", true) }?.let { return it }
        find { it.startsWith("trojan://", true) &&
            TrojanOutboundParser.transport(it) != null }?.let { return it }

        return null''')
    source = single(source,
        '                url.startsWith("vmess://", true) -> parseVmessOutbound(url)',
        '''                url.startsWith("vmess://", true) -> parseVmessOutbound(url)
                url.startsWith("trojan://", true) -> TrojanOutboundParser.create(url)''')
    return source

def transform_dart(source):
    source = single(source,
        "    'vless_tcp': 8,",
        """    'vless_tcp': 8,
    'trojan_tcp': 10,
    'trojan_ws': 11,
    'trojan_grpc': 12,
    'trojan_xhttp': 13,""")
    source = single(source,
        "    'vless_tcp': 'VLESS TCP',",
        """    'vless_tcp': 'VLESS TCP',
    'trojan_tcp': 'Trojan TCP/TLS',
    'trojan_ws': 'Trojan WebSocket/TLS',
    'trojan_grpc': 'Trojan gRPC/TLS',
    'trojan_xhttp': 'Trojan XHTTP/TLS',""")
    source = single(source,
        "    'vless_tcp': 'TCP · Базовый',",
        """    'vless_tcp': 'TCP · Базовый',
    'trojan_tcp': 'TLS · TCP',
    'trojan_ws': 'TLS · WebSocket',
    'trojan_grpc': 'TLS · gRPC',
    'trojan_xhttp': 'TLS · XHTTP поверх TCP',""")
    source = single(source,
        "    if (lower.startsWith('vmess://')) {",
        """    if (lower.startsWith('trojan://')) {
      final params = _parseQueryParams(line);
      final transport = (params['type'] ?? 'tcp').toLowerCase();
      final normalized = switch (transport) {
        'tcp' || 'raw' || '' => 'tcp',
        'ws' || 'websocket' => 'ws',
        'grpc' => 'grpc',
        'xhttp' || 'splithttp' => 'xhttp',
        _ => null,
      };
      if (normalized == null) return null;
      if ((params['security'] ?? 'tls').toLowerCase() != 'tls') return null;
      if ((params['alpn'] ?? '').toLowerCase().split(',').contains('h3')) return null;
      if (normalized == 'grpc' && (params['servicename'] ?? '').isEmpty) return null;
      final uri = Uri.tryParse(line);
      if (uri == null || uri.host.isEmpty || uri.userInfo.isEmpty ||
          !uri.hasPort || uri.port < 1 || uri.port > 65535) return null;
      return 'trojan_$normalized';
    }

    if (lower.startsWith('vmess://')) {""")
    return source

TRANSFORM = {
    KOTLIN / "VlessConfigParser.kt": transform_parser,
    DART / "sub_parser.dart": transform_dart,
}

def write_atomic(path, content):
    fd, temp = tempfile.mkstemp(prefix=".trojan-stage-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as file:
            file.write(content)
            file.flush()
            os.fsync(file.fileno())
        os.chmod(temp, path.stat().st_mode & 0o777 if path.exists() else 0o600)
        os.replace(temp, path)
    finally:
        if os.path.exists(temp):
            os.unlink(temp)

def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--repo-root", type=Path, required=True)
    p.add_argument("--app-root", type=Path, required=True)
    mode = p.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--apply", action="store_true")
    args = p.parse_args()
    repo, app = args.repo_root.resolve(), args.app_root.resolve()
    if not (app / "pubspec.yaml").is_file():
        print("M6_TROJAN=BLOCKED_INVALID_APP")
        return 1
    staged = {}
    for rel, transform in TRANSFORM.items():
        target = app / rel
        if not target.is_file() or target.is_symlink():
            print("M6_TROJAN=BLOCKED_MISSING_APP_SOURCE")
            return 1
        current = target.read_bytes()
        before, after = ORIGINAL[rel]
        try:
            # Accept only exact known current or expected post-apply Git blobs.
            if blob(current) == before:
                updated = transform(current.decode("utf-8")).encode("utf-8")
                if blob(updated) != after:
                    raise ValueError("transformed source differs from reviewed candidate")
            elif blob(current) == after:
                updated = current
            else:
                print("M6_TROJAN=BLOCKED_SOURCE_MISMATCH")
                return 1
        except (ValueError, UnicodeError):
            print("M6_TROJAN=BLOCKED_TRANSFORM")
            return 1
        if updated != current:
            staged[target] = updated
    for rel, path in NEW_SOURCES.items():
        source = repo / path
        dest = app / rel
        if not source.is_file() or source.is_symlink() or dest.is_symlink():
            print("M6_TROJAN=BLOCKED_MISSING_STAGED_SOURCE")
            return 1
        updated = source.read_bytes()
        if blob(updated) != NEW_SOURCE_BLOBS[rel]:
            print("M6_TROJAN=BLOCKED_STAGED_BLOB_MISMATCH")
            return 1
        if dest.exists() and dest.read_bytes() != updated:
            print("M6_TROJAN=BLOCKED_EXISTING_TROJAN_PARSER")
            return 1
        if not dest.exists():
            staged[dest] = updated
    print("M6_TROJAN_SOURCE_PREFLIGHT=PASS")
    print("M6_TROJAN_CHANGED_FILES=" + str(len(staged)))
    if args.check or not staged:
        print("M6_TROJAN_LIVE_APP_UNCHANGED=YES")
        return 0
    backups = {path: path.read_bytes() if path.exists() else None for path in staged}
    installed = []
    try:
        for path, updated in staged.items():
            write_atomic(path, updated)
            installed.append(path)
    except OSError:
        for path in reversed(installed):
            previous = backups[path]
            if previous is None:
                path.unlink(missing_ok=True)
            else:
                write_atomic(path, previous)
        print("M6_TROJAN=WRITE_ERROR_ROLLBACK_ATTEMPTED")
        return 1
    print("M6_TROJAN_APP_SOURCES=INSTALLED")
    print("M6_TROJAN_APK=NOT_BUILT")
    print("M6_LIVE_SERVICES=UNCHANGED")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
