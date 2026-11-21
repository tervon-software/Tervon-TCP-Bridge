// m6-auto-registrar serves Bridge discovery and CSR issuance behind an existing
// HTTPS reverse proxy. This is NOT a 3x-ui or VPN-subscription authorization API.
package main

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"path/filepath"
	"time"

	"github.com/tervon-software/Tervon-TCP-Bridge/integrations/android-ingress/xui"
)

func clientCASigner(data []byte) (crypto.Signer,error) {
	block,tail:=pem.Decode(data)
	if block==nil || len(tail)!=0{return nil,errors.New("invalid existing client CA key PEM")}
	var key any
	var err error
	switch block.Type {
	case "PRIVATE KEY": key,err=x509.ParsePKCS8PrivateKey(block.Bytes)
	case "EC PRIVATE KEY": key,err=x509.ParseECPrivateKey(block.Bytes)
	default: return nil,errors.New("unsupported existing client CA private key type")
	}
	if err!=nil{return nil,errors.New("cannot parse client CA key")}
	s,ok:=key.(*ecdsa.PrivateKey)
	if !ok{return nil,errors.New("existing client CA is not ECDSA")}
	return s,nil
}


type bridgeRuntimeConfig struct {
    Mode string `json:"mode"`
    Listen string `json:"listen"`
    Routes map[string][]json.RawMessage `json:"routes"`
}

type publishedBinding struct {
    HostIp string `json:"HostIp"`
    HostPort string `json:"HostPort"`
}

// Resolve the public mTLS endpoint from the VERIFIED running Bridge container,
// not from a second, manually maintained hostname or port list.
func discoverBridgeEndpoint(root, containerID, service string, serverCAPEM []byte) (string, string, error) {
    data, err := os.ReadFile(filepath.Join(root, "config/server.json"))
    if err != nil { return "", "", err }
    var cfg bridgeRuntimeConfig
    if err = json.Unmarshal(data, &cfg); err != nil { return "", "", err }
    if cfg.Mode != "server" || len(cfg.Routes[service]) == 0 {
        return "", "", errors.New("existing Bridge VPN route unavailable")
    }
    _, internalPort, err := net.SplitHostPort(cfg.Listen)
    if err != nil { return "", "", errors.New("invalid Bridge listen address") }

    serverCertPEM, err := os.ReadFile(filepath.Join(root, "config/pki/server.crt"))
    if err != nil { return "", "", err }
    leafBlock, _ := pem.Decode(serverCertPEM)
    rootBlock, _ := pem.Decode(serverCAPEM)
    if leafBlock == nil || rootBlock == nil { return "", "", errors.New("Bridge server trust is incomplete") }
    leaf, err := x509.ParseCertificate(leafBlock.Bytes)
    if err != nil { return "", "", err }
    ca, err := x509.ParseCertificate(rootBlock.Bytes)
    if err != nil { return "", "", err }
    if len(leaf.DNSNames) != 1 || leaf.DNSNames[0] == "" {
        return "", "", errors.New("Bridge needs exactly one configured server TLS DNS identity")
    }
    name := strings.ToLower(leaf.DNSNames[0])
    pool := x509.NewCertPool()
    pool.AddCert(ca)
    if _, err = leaf.Verify(x509.VerifyOptions{
        DNSName:name, Roots:pool, KeyUsages:[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
    }); err != nil { return "", "", errors.New("Bridge server certificate verification failed") }

    // Docker's published port is the only permitted source of the external
    // host port. There must be exactly one non-loopback IPv4 binding.
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    output, err := exec.CommandContext(ctx, "docker", "inspect", "--format",
        "{{json .NetworkSettings.Ports}}", containerID).Output()
    if err != nil { return "", "", errors.New("cannot inspect running Bridge port") }
    var published map[string][]publishedBinding
    if err = json.Unmarshal(output, &published); err != nil { return "", "", err }
    bindings := published[internalPort + "/tcp"]
    if len(bindings) != 1 || bindings[0].HostIp != "0.0.0.0" {
        return "", "", errors.New("Bridge has no unambiguous public TCP listener")
    }
    port, err := strconv.Atoi(bindings[0].HostPort)
    if err != nil || port < 1 || port > 65535 {
        return "", "", errors.New("invalid published Bridge port")
    }
    return "https://" + net.JoinHostPort(name, strconv.Itoa(port)) + "/v2/stream", name, nil
}

// Transport service names are advertised only when the existing Bridge
// configuration has a server-side route. This function never adds a route.
func configuredTransportServices(root string) (map[string]string, []string, error) {
    data, err := os.ReadFile(filepath.Join(root, "config/server.json"))
    if err != nil { return nil, nil, err }
    var cfg bridgeRuntimeConfig
    if err := json.Unmarshal(data, &cfg); err != nil { return nil, nil, err }
    if cfg.Mode != "server" || len(cfg.Routes["vpn"]) == 0 {
        return nil, nil, errors.New("TCP Bridge backend unavailable")
    }
    transports := map[string]string{"tcp": "vpn"}
    services := []string{"vpn"}
    if len(cfg.Routes["vpn-xhttp"]) > 0 {
        transports["xhttp"] = "vpn-xhttp"
        services = append(services, "vpn-xhttp")
    }
    if len(cfg.Routes["vpn-trojan"]) > 0 {
        transports["trojan_tcp"] = "vpn-trojan"
        services = append(services, "vpn-trojan")
    }
    if len(cfg.Routes["vpn-trojan-xhttp"]) > 0 {
        transports["trojan_xhttp"] = "vpn-trojan-xhttp"
        services = append(services, "vpn-trojan-xhttp")
    }
    return transports, services, nil
}

func main() {
    var root, listen, containerID string
    flag.StringVar(&root, "root", "/etc/tervon/bridge-android", "existing Android Bridge root")
    flag.StringVar(&listen, "listen", "", "loopback address for existing HTTPS proxy (required)")
    flag.StringVar(&containerID, "container-id", "", "running Android ingress Docker container ID")
    flag.Parse()
    if filepath.Clean(root) != root || !filepath.IsAbs(root) || root == "/" {
        log.Fatal("invalid existing Android Bridge root")
    }
    host, port, err := net.SplitHostPort(listen)
    if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || port == "" {
        log.Fatal("registrar must bind a literal loopback IP:port")
    }
    if containerID == "" { log.Fatal("running Android Bridge container ID required") }
    clientCAPEM, err := os.ReadFile(filepath.Join(root, "config/pki/client-ca.crt"))
    if err != nil { log.Fatal("existing client CA unavailable") }
    serverCAPEM, err := os.ReadFile(filepath.Join(root, "private/server-ca.crt"))
    if err != nil { log.Fatal("existing public server CA unavailable") }
    privatePEM, err := os.ReadFile(filepath.Join(root, "private/client-ca.key"))
    if err != nil { log.Fatal("existing client CA signer unavailable") }
    signer, err := clientCASigner(privatePEM)
    if err != nil { log.Fatal(err) }
    const service = "vpn" // existing M6 service identifier, not a destination
    remote, serverName, err := discoverBridgeEndpoint(root, containerID, service, serverCAPEM)
    if err != nil { log.Fatal(err) }
    transports, services, err := configuredTransportServices(root)
    if err != nil { log.Fatal(err) }
    activator := &xui.HotACLActivator{
        Path:filepath.Join(root,"config/server.json"), Service:service, Services:services,
        Confirm:xui.DockerACLConfirmer{
            ContainerID:containerID,
            HostConfigDirectory:filepath.Join(root,"config"),
        }.Confirm,
    }
    enrollment := &xui.BridgeEnrollmentHTTP{
        RemoteURL:remote, ServerName:serverName, Service:service, TransportServices:transports,
        ClientCAPEM:clientCAPEM, ServerCAPEM:serverCAPEM, ClientCAKey:signer,
        Activate:func(fingerprint string) error {
            ctx,cancel := context.WithTimeout(context.Background(),15*time.Second)
            defer cancel()
            return activator.Activate(ctx,fingerprint)
        },
    }
    handler,err := enrollment.Handler()
    if err != nil { log.Fatal(err) }
    mux := http.NewServeMux()
    mux.Handle("/.well-known/tervon-bridge",handler)
    ln,err := net.Listen("tcp",listen)
    if err != nil { log.Fatal(fmt.Errorf("cannot open registrar loopback listener: %w",err)) }
    log.Print("M6_BRIDGE_REGISTRAR=LOOPBACK_LISTENING")
    srv := &http.Server{
        Handler:mux, ReadHeaderTimeout:5*time.Second,
        IdleTimeout:15*time.Second, MaxHeaderBytes:16<<10,
    }
    log.Fatal(srv.Serve(ln))
}
