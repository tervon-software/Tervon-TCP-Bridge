package main

import (
    "os"
    "path/filepath"
    "testing"
)

func testTCPRouteConfig(t *testing.T, config string) string {
    t.Helper()
    root := t.TempDir()
    if err := os.MkdirAll(filepath.Join(root, "config"), 0700); err != nil {t.Fatal(err)}
    if err := os.WriteFile(filepath.Join(root, "config", "server.json"), []byte(config), 0600); err != nil {t.Fatal(err)}
    return root
}

func TestDynamicTCPRoutesDoNotInspectVPNProtocols(t *testing.T) {
    root := testTCPRouteConfig(t, `{"mode":"server","listen":"0.0.0.0:8443","routes":{
        "ordinary":[{"Name":"tcp","Address":"host.docker.internal:41771"}],
        "random-name":[{"Name":"xhttp","Address":"host.docker.internal:55636"}],
        "yet-another-service":[{"Name":"arbitrary","Address":"host.docker.internal:49123"}]
    }}`)
    routes, services, err := configuredTCPRoutes(root)
    if err != nil {t.Fatal(err)}
    if len(routes) != 3 || len(services) != 3 {
        t.Fatalf("wrong dynamic route count: %v / %v", routes, services)
    }
    if routes["41771"] != "ordinary" || routes["55636"] != "random-name" ||
        routes["49123"] != "yet-another-service" {
        t.Fatalf("ports not derived from config: %v", routes)
    }
}

func TestDynamicTCPRoutesFailClosed(t *testing.T) {
    cases := []string{
        `{"mode":"client","routes":{"vpn":[{"Address":"h:443"}]}}`,
        `{"mode":"server","routes":{}}`,
        `{"mode":"server","routes":{"vpn":[]}}`,
        `{"mode":"server","routes":{"vpn":[{"Address":"not-hostport"}]}}`,
        `{"mode":"server","routes":{"vpn":[{"Address":"host:0"}]}}`,
        `{"mode":"server","routes":{"vpn":[{"Address":"host:443"}],"another":[{"Address":"host:443"}]}}`,
        `{"mode":"server","routes":{"vpn":[{"Address":"host:443"}],"bad/name":[{"Address":"host:444"}]}}`,
    }
    for _,config := range cases {
        root := testTCPRouteConfig(t, config)
        if _,_,err := configuredTCPRoutes(root); err == nil {
            t.Fatalf("unexpected acceptance of malformed or ambiguous backend: %s",config)
        }
    }
}
