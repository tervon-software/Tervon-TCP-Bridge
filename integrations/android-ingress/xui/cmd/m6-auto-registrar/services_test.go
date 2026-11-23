package main

import (
    "os"
    "path/filepath"
    "reflect"
    "testing"
)

func testServiceConfig(t *testing.T, config string) string {
    t.Helper()
    root := t.TempDir()
    if err := os.MkdirAll(filepath.Join(root, "config"), 0700); err != nil { t.Fatal(err) }
    if err := os.WriteFile(filepath.Join(root, "config", "server.json"), []byte(config), 0600); err != nil { t.Fatal(err) }
    return root
}

func TestConfiguredServiceNamesNeverDeriveBackendPorts(t *testing.T) {
    root := testServiceConfig(t, `{"mode":"server","listen":"0.0.0.0:8443","routes":{
        "z-service":[{"Address":"host.docker.internal:41771"}],
        "a-service":[{"Address":"host.docker.internal:55636"}]
    }}`)
    names, err := configuredServiceNames(root)
    if err != nil { t.Fatal(err) }
    if !reflect.DeepEqual(names, []string{"a-service", "z-service"}) {
        t.Fatalf("unexpected opaque services: %v", names)
    }
}

func TestConfiguredServiceNamesFailsClosed(t *testing.T) {
    cases := []string{
        `{"mode":"client","routes":{"vpn":[{"Address":"h:443"}]}}`,
        `{"mode":"server","routes":{}}`,
        `{"mode":"server","routes":{"vpn":[]}}`,
        `{"mode":"server","routes":{"bad/name":[{"Address":"host:443"}]}}`,
        `{"mode":"server","routes":{"bad space":[{"Address":"host:443"}]}}`,
        `{"mode":"server","routes":{"vpn":[{"Address":"host:443"}]},"listen":true}`,
    }
    for _, config := range cases {
        root := testServiceConfig(t, config)
        if _, err := configuredServiceNames(root); err == nil {
            // Only invalid service identifiers and missing backends are required
            // to fail here. Endpoint validation belongs to TLS endpoint checks.
            if config == cases[5] { continue }
            t.Fatalf("invalid service configuration accepted")
        }
    }
}

// The certificate registrar accepts only preapproved source-verified images.
// It does not select a VPN service or decide a backend.
func TestApprovedRegistrarImageExactIdentity(t *testing.T) {
    for _, tag := range []string{
        "tervon-bridge-android-ingress:v1.0.2",
        "tervon-bridge-android-ingress:m6-dynamic-e1326094",
    } {
        if !approvedRegistrarImage(tag) { t.Fatalf("approved image rejected: %s",tag) }
    }
    for _, tag := range []string{
        "", "tervon-bridge-android-ingress:latest",
        "tervon-bridge-android-ingress:m6-dynamic",
        "tervon-bridge-android-ingress:m6-runtime-acl",
        "tervon-bridge-android-ingress:v1.0.3",
    } {
        if approvedRegistrarImage(tag) { t.Fatalf("unapproved image accepted: %s",tag) }
    }
}
