#!/usr/bin/env python3
"""Apply ONLY the approved full RuntimePolicy overlay to clean Linux v1.0.1 source.

Refuses a different source commit, dirty checkout, different anchors or
pre-existing target files. Never touches an installed binary or live service.
"""
import argparse
from pathlib import Path
import subprocess
import sys

RELEASE_COMMIT = "96c136702d10b6fb9a16aa2893f6b978e8b14e86"
HELPERS = ("runtime_policy.go", "runtime_policy_test.go", "runtime_policy_stream_test.go")


def git(root, *args):
    return subprocess.check_output(
        ["git", "-C", str(root), *args], text=True).strip()


def replace_one(text, before, after, label):
    if text.count(before) != 1:
        raise ValueError("release source mismatch: " + label)
    return text.replace(before, after)


def patch_bridge(text):
    text = replace_one(
        text,
        "\tAuthorizedServices map[[32]byte]map[string]bool\n",
        "\tAuthorizedServices map[[32]byte]map[string]bool\n"
        "\t// PolicyLookup returns both ACL admission and routes in one immutable snapshot.\n"
        "\tPolicyLookup func([32]byte, string) ([]Backend, bool)\n",
        "policy lookup server field",
    )
    text = replace_one(
        text,
        "func (s *Server) pick(ctx context.Context, service string) (net.Conn, error) {\n"
        "\ts.mu.Lock()\n"
        "\tlist := s.Routes[service]\n",
        "func (s *Server) pick(ctx context.Context, service string) (net.Conn, error) {\n"
        "\treturn s.pickWithRoutes(ctx, service, s.Routes[service])\n"
        "}\n\n"
        "// pickWithRoutes uses the routes selected from the SAME generation as authorization.\n"
        "func (s *Server) pickWithRoutes(ctx context.Context, service string, list []Backend) (net.Conn, error) {\n"
        "\ts.mu.Lock()\n",
        "route selection",
    )
    text = replace_one(
        text,
        "\tif !s.AuthorizedServices[fingerprint][service] {\n",
        "\tvar policyRoutes []Backend\n"
        "\tallowed := s.AuthorizedServices[fingerprint][service]\n"
        "\tif s.PolicyLookup != nil {\n"
        "\t\tpolicyRoutes, allowed = s.PolicyLookup(fingerprint, service)\n"
        "\t}\n"
        "\tif !allowed {\n",
        "stream admission",
    )
    text = replace_one(
        text,
        "\tc, err := s.pick(r.Context(), service)\n",
        "\tvar c net.Conn\n"
        "\tvar err error\n"
        "\tif s.PolicyLookup != nil {\n"
        "\t\tc, err = s.pickWithRoutes(r.Context(), service, policyRoutes)\n"
        "\t} else {\n"
        "\t\tc, err = s.pick(r.Context(), service)\n"
        "\t}\n",
        "same-generation backend connection",
    )
    return replace_one(
        text, "\tvar source io.Reader = resp.Body\n",
        "\tif cl.Metrics != nil {\n"
        "\t\tcl.Metrics.ObserveTLS(resp.TLS)\n"
        "\t}\n"
        "\tvar source io.Reader = resp.Body\n",
        "count accepted client Bridge streams",
    )


def patch_main(text):
    text = replace_one(
        text, "len(c.Routes) == 0 || len(c.Authorized) == 0",
        "c.Routes == nil || len(c.Authorized) == 0",
        "allow explicitly empty route policy",
    )
    text = replace_one(
        text, "len(services) == 0 {",
        "(len(services) == 0) != (len(c.Routes) == 0) {",
        "validate service grants against empty route policy",
    )
    text = replace_one(
        text, '\t"net/url"\n', '\t"net/url"\n\t"reflect"\n',
        "reflect import",
    )
    anchor = "\thandler := &bridge.Server{Routes: c.Routes, AuthorizedServices: authorized, MaxConcurrent: c.MaxStreams, Metrics: metrics}\n"
    replacement = (
        "\tpolicy, err := bridge.NewRuntimePolicy(c.Routes, c.Authorized)\n"
        "\tif err != nil { log.Fatal(\"invalid initial RuntimePolicy\") }\n"
        "\thandler := &bridge.Server{Routes: c.Routes, AuthorizedServices: authorized, MaxConcurrent: c.MaxStreams, Metrics: metrics, PolicyLookup: policy.Lookup}\n"
        "\t// SIGHUP updates routes and ACL as one generation for NEW streams.\n"
        "\t// An established TCP stream retains its already dialed backend.\n"
        "\thup := make(chan os.Signal, 1)\n"
        "\tsignal.Notify(hup, syscall.SIGHUP)\n"
        "\tdefer signal.Stop(hup)\n"
        "\tgo func() {\n"
        "\t\tfor {\n"
        "\t\t\tselect {\n"
        "\t\t\tcase <-ctx.Done():\n"
        "\t\t\t\treturn\n"
        "\t\t\tcase <-hup:\n"
        "\t\t\t\tfresh, e := load(*path)\n"
        "\t\t\t\tif e != nil {\n"
        "\t\t\t\t\tlog.Print(\"M6_POLICY_RELOAD=FAIL invalid config\")\n"
        "\t\t\t\t\tcontinue\n"
        "\t\t\t\t}\n"
        "\t\t\t\toldStatic, newStatic := c, fresh\n"
        "\t\t\t\toldStatic.Authorized, newStatic.Authorized = nil, nil\n"
        "\t\t\t\toldStatic.Routes, newStatic.Routes = nil, nil\n"
        "\t\t\t\tif !reflect.DeepEqual(oldStatic, newStatic) {\n"
        "\t\t\t\t\tlog.Print(\"M6_POLICY_RELOAD=FAIL non-policy configuration changed\")\n"
        "\t\t\t\t\tcontinue\n"
        "\t\t\t\t}\n"
        "\t\t\t\tdigest, e := policy.Replace(fresh.Routes, fresh.Authorized)\n"
        "\t\t\t\tif e != nil {\n"
        "\t\t\t\t\tlog.Print(\"M6_POLICY_RELOAD=FAIL invalid routes or ACL\")\n"
        "\t\t\t\t\tcontinue\n"
        "\t\t\t\t}\n"
        "\t\t\t\tlog.Printf(\"M6_POLICY_RELOAD=PASS digest=%s\", digest)\n"
        "\t\t\t}\n"
        "\t\t}\n"
        "\t}()\n"
    )
    text = replace_one(text, anchor, replacement, "core policy initialization")
    return replace_one(
        text, "\tsrv := newBridgeHTTPServer(c.Listen, handler, tlsCfg)\n",
        "\tmux := http.NewServeMux()\n"
        '\tmux.Handle("/v2/status", bridgeStatusHandler(policy.AllowsStatus, metrics.ServerStatus))\n'
        '\tmux.Handle("/", handler)\n'
        "\tsrv := newBridgeHTTPServer(c.Listen, mux, tlsCfg)\n",
        "authenticated status handler",
    )


def patch_metrics(text):
    return text.rstrip() + (
        "\n\n// ServerStatus reads status without resetting the telemetry rate window.\n"
        "func (r *Registry) ServerStatus() (bool, string, int64, uint64) {\n"
        "\treturn r.ready.Load(), r.BuildVersion, int64(time.Since(r.started) / time.Second), r.tls13h2mtls.Load()\n"
        "}\n"
    )


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--release-root", required=True, type=Path)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    root = args.release_root.resolve()
    if git(root, "rev-parse", "HEAD") != RELEASE_COMMIT:
        raise ValueError("unexpected v1.0.1 release source commit")
    if git(root, "status", "--porcelain"):
        raise ValueError("release source checkout must be clean")
    target = root / "prototype"
    if not (target / "go.mod").is_file():
        raise ValueError("release Go module missing")
    changes = {
        target / "bridge.go": patch_bridge((target / "bridge.go").read_text()),
        target / "cmd/bridge/main.go": patch_main((target / "cmd/bridge/main.go").read_text()),
    }
    metrics = target / "telemetry/metrics.go"
    changes[metrics] = patch_metrics(metrics.read_text())
    status_helpers = ("bridge_status.go", "bridge_status_test.go")
    for name in (*HELPERS, *status_helpers):
        path = target / ("cmd/bridge" if name in status_helpers else "") / name
        if path.exists() or path.is_symlink():
            raise ValueError("existing file must not be overwritten: " + name)
        source = Path(__file__).resolve().parent / name
        changes[path] = source.read_text(encoding="utf-8")
    print("M6_FULL_POLICY_SOURCE_PREFLIGHT=PASS")
    if args.check:
        print("SOURCE_MODIFIED=NO")
        return
    for path, data in changes.items():
        path.write_text(data, encoding="utf-8")
    print("M6_FULL_POLICY_SOURCE_STAGED=YES")
    print("RUNNING_SERVICES=UNCHANGED")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, subprocess.CalledProcessError) as exc:
        print("M6_FULL_POLICY_SOURCE=FAIL " + str(exc), file=sys.stderr)
        sys.exit(1)
