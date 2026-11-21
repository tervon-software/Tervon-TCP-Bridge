#!/usr/bin/env python3
"""M6 source-only diagnostic overlay: released Go v1.0.1 and Android runtime.
The Go checkout must be the exact immutable release, not the M6 prototype.
No binaries, APKs or services are changed here.
"""
import argparse
import hashlib
import os
from pathlib import Path
import tempfile

ANDROID = Path("android/app/src/main/kotlin/com/tervonvpndpi/app/BridgeCoreManager.kt")
GO = Path("prototype/bridge.go")
RELEASE_SHA = "45860c4ed5eff7e52321a98e2b4bc6d82e350d98"

def blob(raw):
    return hashlib.sha1(b"blob " + str(len(raw)).encode() + b"\0" + raw).hexdigest()

def replace(s, old, new, label):
    if s.count(old) != 1:
        raise ValueError("source anchor mismatch: " + label)
    return s.replace(old, new, 1)

def android(s):
    s = replace(s,
        '            // No process output is persisted: stdout may contain sensitive paths.\n            .redirectOutput(ProcessBuilder.Redirect.to(File("/dev/null")))',
        '            // Drain merged streams; only fixed error codes reach logcat.\n            .redirectOutput(ProcessBuilder.Redirect.PIPE)',
        "Android redirect")
    anchor = '            process = started\n            state = if (started.isAlive) State.RUNNING else State.FAILED'
    reader = '''
            // The single-threaded executor drains the pipe before waitFor().
            executor.execute {
                try {
                    started.inputStream.bufferedReader().useLines { lines ->
                        lines.forEach { line ->
                            val marker = "M6_BRIDGE_SESSION_ERROR="
                            val index = line.indexOf(marker)
                            if (index >= 0) {
                                val code = line.substring(index + marker.length).trim()
                                if (code in setOf(
                                        "CONNECT_REFUSED", "NETWORK_UNREACHABLE", "TIMEOUT",
                                        "TLS_CERTIFICATE", "TLS_HANDSHAKE", "HTTP_REJECTED",
                                        "REMOTE_IO", "CANCELLED", "UNKNOWN")) {
                                    android.util.Log.w("TervonBridge", marker + code)
                                }
                            }
                        }
                    }
                } catch (_: Exception) {
                    // Diagnostic stream failure must never terminate the VPN.
                }
            }'''
    return replace(s, anchor, anchor + reader, "Android reader")

def go(s):
    s = replace(s, '"context"',
        '"context"\n"crypto/x509"\n"errors"\n"log"\n"strings"\n"sync/atomic"\n"syscall"',
        "Go imports")
    s = replace(s, '\t\t\t_ = cl.HandleContext(sessionCtx, c)',
        '''            if err := cl.HandleContext(sessionCtx, c); err != nil {
                // Fixed category only: no URLs, IPs, PEM paths or payloads.
                now := time.Now().UnixNano()
                last := lastBridgeSessionDiagnostic.Load()
                if now-last >= int64(time.Second) &&
                    lastBridgeSessionDiagnostic.CompareAndSwap(last, now) {
                    log.Printf("M6_BRIDGE_SESSION_ERROR=%s", bridgeSessionFailureCode(err))
                }
            }''', "Go session error")
    fn = '''
// M6 source-only observability: fixed codes, at most once per second.
var lastBridgeSessionDiagnostic atomic.Int64

func bridgeSessionFailureCode(err error) string {
    switch {
    case errors.Is(err, context.Canceled):
        return "CANCELLED"
    case errors.Is(err, syscall.ECONNREFUSED):
        return "CONNECT_REFUSED"
    case errors.Is(err, syscall.ENETUNREACH), errors.Is(err, syscall.EHOSTUNREACH):
        return "NETWORK_UNREACHABLE"
    }
    var authority x509.UnknownAuthorityError
    if errors.As(err, &authority) { return "TLS_CERTIFICATE" }
    var hostname x509.HostnameError
    if errors.As(err, &hostname) { return "TLS_CERTIFICATE" }
    var record tls.RecordHeaderError
    if errors.As(err, &record) { return "TLS_HANDSHAKE" }
    var ne net.Error
    if errors.As(err, &ne) && ne.Timeout() { return "TIMEOUT" }
    if strings.HasPrefix(err.Error(), "remote status ") { return "HTTP_REJECTED" }
    if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) { return "REMOTE_IO" }
    return "UNKNOWN"
}

'''
    return replace(s, 'func (cl *Client) ListenAndServe(ctx context.Context, addr string) error {',
        fn + 'func (cl *Client) ListenAndServe(ctx context.Context, addr string) error {',
        "Go diagnostic helper")

def atomic(path, data):
    st = path.stat()
    fd, tmp = tempfile.mkstemp(prefix=".m6-diagnostic-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as output:
            output.write(data)
            output.flush()
            os.fsync(output.fileno())
        os.chmod(tmp, st.st_mode & 0o777)
        os.replace(tmp, path)
    finally:
        if os.path.exists(tmp):
            os.unlink(tmp)

def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--app-root", type=Path, required=True)
    p.add_argument("--release-root", type=Path, required=True)
    m = p.add_mutually_exclusive_group(required=True)
    m.add_argument("--check", action="store_true")
    m.add_argument("--apply", action="store_true")
    a = p.parse_args()
    targets = (a.app_root / ANDROID, a.release_root / GO)
    if any(not path.is_file() or path.is_symlink() for path in targets):
        print("M6_DIAGNOSTICS=BLOCKED_MISSING_SOURCE")
        return 1
    originals = {path: path.read_bytes() for path in targets}
    if blob(originals[targets[1]]) != RELEASE_SHA:
        print("M6_DIAGNOSTICS=BLOCKED_RELEASE_SHA")
        return 1
    try:
        changed = {
            targets[0]: android(originals[targets[0]].decode()).encode(),
            targets[1]: go(originals[targets[1]].decode()).encode(),
        }
    except (ValueError, UnicodeError) as e:
        print("M6_DIAGNOSTICS=BLOCKED_SOURCE_MISMATCH", e)
        return 1
    print("M6_DIAGNOSTICS_PREFLIGHT=PASS")
    if a.check:
        print("LIVE_SERVICES=UNCHANGED")
        return 0
    done = []
    try:
        for path in targets:
            atomic(path, changed[path])
            done.append(path)
    except OSError:
        for path in reversed(done):
            try:
                atomic(path, originals[path])
            except OSError:
                print("M6_DIAGNOSTICS=ROLLBACK_INCOMPLETE")
                return 1
        print("M6_DIAGNOSTICS=WRITE_FAILED_ROLLED_BACK")
        return 1
    print("M6_DIAGNOSTICS_SOURCE_APPLIED=PASS")
    print("LIVE_SERVICES=UNCHANGED")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
