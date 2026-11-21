package main

import (
    "os"
    "path/filepath"
    "testing"
)

func TestConfiguredTransportServicesOnlyAdvertisesExistingRoutes(t *testing.T) {
    root := t.TempDir()
    if err := os.MkdirAll(filepath.Join(root, "config"), 0700); err != nil { t.Fatal(err) }
    path := filepath.Join(root, "config", "server.json")
    write := func(value string) {
        t.Helper()
        if err := os.WriteFile(path, []byte(value), 0600); err != nil { t.Fatal(err) }
    }
    write(`{"mode":"server","listen":"0.0.0.0:8443","routes":{"vpn":[{"name":"tcp","address":"host.docker.internal:30000"}]}}`)
    routes, services, err := configuredTransportServices(root)
    if err != nil {t.Fatal(err)}
    if len(routes)!=1 || routes["tcp"]!="vpn" || len(services)!=1 || services[0]!="vpn" {
        t.Fatalf("legacy route incorrectly changed: %v %v", routes, services)
    }
    write(`{"mode":"server","listen":"0.0.0.0:8443","routes":{"vpn":[{"name":"tcp","address":"host.docker.internal:30000"}],"vpn-xhttp":[{"name":"xhttp","address":"host.docker.internal:30443"}]}}`)
    routes, services, err = configuredTransportServices(root)
    if err!=nil {t.Fatal(err)}
    if len(routes)!=2 || routes["xhttp"]!="vpn-xhttp" || len(services)!=2 || services[1]!="vpn-xhttp" {
        t.Fatalf("XHTTP capability missing: %v %v",routes,services)
    }
    write(`{"mode":"server","routes":{"vpn-xhttp":[{"address":"host.docker.internal:30443"}]}}`)
    if _,_,err := configuredTransportServices(root); err==nil {t.Fatal("accepted absent TCP route")}
    write(`{"mode":"server","routes":{"vpn":[{"address":"host.docker.internal:30000"}],"vpn-xhttp":[]}}`)
    routes, services, err = configuredTransportServices(root)
    if err != nil {t.Fatal(err)}
    if len(routes)!=1 || len(services)!=1 {t.Fatal("empty XHTTP route advertised")}
}

func TestConfiguredTransportServicesIncludesTrojanWithoutPortsInCode(t *testing.T) {
    root := t.TempDir()
    if err := os.MkdirAll(filepath.Join(root, "config"), 0700); err != nil {t.Fatal(err)}
    contents := `{"mode":"server","listen":"0.0.0.0:8443","routes":{
      "vpn":[{"address":"host.docker.internal:49312"}],
      "vpn-trojan":[{"address":"host.docker.internal:48111"}],
      "vpn-trojan-xhttp":[{"address":"host.docker.internal:47391"}]}}`
    if err := os.WriteFile(filepath.Join(root, "config/server.json"), []byte(contents), 0600); err != nil {t.Fatal(err)}
    routes, services, err := configuredTransportServices(root)
    if err != nil {t.Fatal(err)}
    if routes["trojan_tcp"] != "vpn-trojan" || routes["trojan_xhttp"] != "vpn-trojan-xhttp" {
        t.Fatalf("missing Trojan discovery: %v", routes)
    }
    if len(services) != 3 || services[1] != "vpn-trojan" || services[2] != "vpn-trojan-xhttp" {
        t.Fatalf("Trojan ACL mismatch: %v", services)
    }
}
