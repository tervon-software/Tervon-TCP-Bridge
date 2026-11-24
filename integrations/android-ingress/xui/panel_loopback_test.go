package xui

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestConfiguredPanelLoopbackKeepsTLSIdentity(t *testing.T) {
	var expectedHost string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != expectedHost || r.URL.Path != "/prefix/panel/api/inbounds/list/slim" ||
			r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("panel request lost the configured HTTP host, path or credential")
		}
		_, _ = w.Write([]byte(`{"success":true,"obj":[]}`))
	}))
	defer server.Close()
	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	name := server.Certificate().DNSNames[0]
	expectedHost = net.JoinHostPort(name, port)
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())

	dir := t.TempDir()
	credential := filepath.Join(dir, "api-header")
	config := filepath.Join(dir, "panel.json")
	if err := os.WriteFile(credential, []byte("Authorization: Bearer test-token"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, host := range []string{name, "wrong-name.invalid"} {
		body, err := json.Marshal(panelSettings{
			Type: "3x-ui", BaseURL: "https://" + net.JoinHostPort(host, port) + "/prefix",
			APICredential: credential, ConnectLoopback: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(config, body, 0600); err != nil {
			t.Fatal(err)
		}
		client, err := configuredPanelClient(config)
		if err != nil {
			t.Fatal(err)
		}
		client.httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: roots}
		_, err = client.SelectedBridgeInbounds(context.Background())
		if (err == nil) != (host == name) {
			t.Fatalf("unexpected certificate verification result for %q: %v", host, err)
		}
	}
}
