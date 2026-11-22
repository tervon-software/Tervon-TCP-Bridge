#!/usr/bin/env python3
"""Apply v1.0.2 atomic route+ACL candidate to a separately staged M6 ACL
release-v1.0.1 tree. Never touches a running container or existing services.
Run the original M6 runtime-acl overlay first, in an isolated release worktree.
"""
import argparse
from pathlib import Path
import subprocess
import sys

RELEASE_COMMIT = "96c136702d10b6fb9a16aa2893f6b978e8b14e86"
EXTRA = ("policy.go", "policy_test.go", "policy_stream_test.go")


def replace_once(text, original, replacement, name):
    if text.count(original) != 1:
        raise ValueError("unexpected M6 ACL staging: " + name)
    return text.replace(original, replacement)


def patch_bridge(raw):
    raw = replace_once(
        raw,
        "\tAuthorizationLookup func([32]byte, string) bool\n",
        "\tAuthorizationLookup func([32]byte, string) bool\n"
        "\t// Optional v1.0.2 atomic ACL+backend selector. A single call\n"
        "\t// grants admission and pins the exact backend list for this stream.\n"
        "\tRouteLookup func([32]byte, string) ([]Backend, bool)\n",
        "route callback field",
    )
    raw = replace_once(
        raw,
        '\tallowed := s.AuthorizedServices[fingerprint][service]\n'
        '\tif s.AuthorizationLookup != nil {\n'
        '\t\tallowed = s.AuthorizationLookup(fingerprint, service)\n'
        '\t}\n',
        '\tallowed := s.AuthorizedServices[fingerprint][service]\n'
        '\tvar selected []Backend\n'
        '\tif s.RouteLookup != nil {\n'
        '\t\tselected, allowed = s.RouteLookup(fingerprint, service)\n'
        '\t} else if s.AuthorizationLookup != nil {\n'
        '\t\tallowed = s.AuthorizationLookup(fingerprint, service)\n'
        '\t}\n',
        "atomic admission",
    )
    raw = replace_once(
        raw,
        '\tc, err := s.pick(r.Context(), service)\n',
        '\tvar c net.Conn\n'
        '\tvar err error\n'
        '\tif s.RouteLookup != nil {\n'
        '\t\tc, err = s.pickFrom(r.Context(), service, selected)\n'
        '\t} else {\n'
        '\t\tc, err = s.pick(r.Context(), service)\n'
        '\t}\n',
        "stream route selection",
    )
    marker = 'func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {\n'
    method = """// pickFrom uses the backend snapshot selected at stream admission.
// It never touches the mutable control-plane policy during a TCP dial.
func (s *Server) pickFrom(ctx context.Context, service string, list []Backend) (net.Conn, error) {
    if len(list) == 0 { return nil, fmt.Errorf("unknown service") }
    s.mu.Lock()
    if s.next == nil { s.next = make(map[string]int) }
    start := s.next[service] % len(list)
    s.next[service] = (start + 1) % len(list)
    s.mu.Unlock()
    timeout := s.DialTimeout
    if timeout <= 0 { timeout = 750 * time.Millisecond }
    var last error
    for i := range list {
        if err := ctx.Err(); err != nil { return nil, err }
        b := list[(start+i)%len(list)]
        c, err := (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp", b.Address)
        if err == nil { return c, nil }
        last = err
    }
    return nil, fmt.Errorf("backends unavailable: %w", last)
}

"""
    return replace_once(raw, marker, method+marker, "pickFrom insertion")


def patch_main(raw):
    raw = replace_once(
        raw,
        'acl, err := bridge.NewRuntimeACL(c.Authorized, c.Routes)',
        'policy, err := bridge.NewRuntimePolicy(c.Authorized, c.Routes)',
        "policy constructor",
    )
    raw = replace_once(raw, 'AuthorizationLookup: acl.Allowed}', 'RouteLookup: policy.Select}', "policy registration")
    raw = replace_once(
        raw,
        '\t\t\t\toldNonACL, newNonACL := c, fresh\n'
        '\t\t\t\toldNonACL.Authorized, newNonACL.Authorized = nil, nil\n'
        '\t\t\t\tif !reflect.DeepEqual(oldNonACL, newNonACL) {\n'
        '\t\t\t\t\tlog.Print("M6_ACL_RELOAD=FAIL non-ACL config changed")\n',
        '\t\t\t\toldFixed, newFixed := c, fresh\n'
        '\t\t\t\toldFixed.Authorized, newFixed.Authorized = nil, nil\n'
        '\t\t\t\toldFixed.Routes, newFixed.Routes = nil, nil\n'
        '\t\t\t\tif !reflect.DeepEqual(oldFixed, newFixed) {\n'
        '\t\t\t\t\tlog.Print("M6_POLICY_RELOAD=FAIL fixed config changed")\n',
        "reload immutable configuration",
    )
    raw = replace_once(
        raw,
        'if e := acl.Replace(fresh.Authorized, c.Routes); e != nil {',
        'if e := policy.Replace(fresh.Authorized, fresh.Routes); e != nil {',
        "atomic policy publication",
    )
    raw = replace_once(
        raw,
        '\t\t\t\taclJSON, _ := json.Marshal(fresh.Authorized)\n'
        '\t\t\t\taclDigest := sha256.Sum256(aclJSON)\n'
        '\t\t\t\tlog.Printf("M6_ACL_RELOAD=PASS digest=%x", aclDigest)\n',
        '\t\t\t\tpolicyJSON, _ := json.Marshal(struct {\n'
        '\t\t\t\t\tRoutes map[string][]bridge.Backend `json:"routes"`\n'
        '\t\t\t\t\tAuthorized map[string][]string `json:"authorized"`\n'
        '\t\t\t\t}{fresh.Routes, fresh.Authorized})\n'
        '\t\t\t\tpolicyDigest := sha256.Sum256(policyJSON)\n'
        '\t\t\t\tlog.Printf("M6_POLICY_RELOAD=PASS digest=%x", policyDigest)\n',
        "policy success acknowledgment",
    )
    return raw


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--release-root", type=Path, required=True)
    mode = p.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--apply", action="store_true")
    a = p.parse_args()
    root = a.release_root.resolve(strict=True)
    commit = subprocess.check_output(["git", "-C", str(root), "rev-parse", "HEAD"], text=True).strip()
    if commit != RELEASE_COMMIT:
        raise ValueError("unexpected v1.0.1 release HEAD")
    go = root / "prototype"
    original = go / "bridge.go"
    cli = go / "cmd/bridge/main.go"
    updates = {
        original: patch_bridge(original.read_text()),
        cli: patch_main(cli.read_text()),
    }
    for name in EXTRA:
        dst = go / name
        if dst.exists() or dst.is_symlink():
            raise ValueError("refusing to replace staged helper: "+name)
        src = Path(__file__).resolve().parent / name
        if not src.is_file():
            raise ValueError("missing route policy helper: "+name)
        updates[dst] = src.read_text()
    # Source-only integration test of the real isolated subprocess reload.
    # Never install this file in a live Docker image.
    test_src = Path(__file__).resolve().parent / "policy_ack_integration_test.go"
    test_dst = go / "cmd/bridge/policy_ack_integration_test.go"
    if test_dst.exists() or test_dst.is_symlink() or not test_src.is_file():
        raise ValueError("refusing to overwrite or omit isolated route ACK test")
    updates[test_dst] = test_src.read_text()
    print("M6_V102_ROUTE_SOURCE_CHECK=PASS")
    if a.check:
        return
    for path, content in updates.items():
        path.write_text(content)
    print("M6_V102_ROUTE_SOURCE=STAGED")
    print("RUNNING_SERVICES=UNCHANGED")


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print("M6_V102_ROUTE_SOURCE=FAIL: "+str(exc), file=sys.stderr)
        sys.exit(1)
