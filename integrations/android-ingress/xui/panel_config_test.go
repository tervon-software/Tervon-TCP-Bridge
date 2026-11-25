package xui

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfiguredPanelSettingsReloadAndPermissions(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "panel.json")
	credentialPath := filepath.Join(dir, "api-header")
	write := func(path, value string) {
		t.Helper()
		staged := path + ".next"
		if err := os.WriteFile(staged, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(staged, path); err != nil {
			t.Fatal(err)
		}
	}
	set := func(panelType, baseURL string) {
		t.Helper()
		config, err := json.Marshal(panelSettings{Type: panelType, BaseURL: baseURL, APICredential: credentialPath})
		if err != nil {
			t.Fatal(err)
		}
		write(settingsPath, string(config))
	}

	if ConfiguredPanelInbounds("") != nil {
		t.Fatal("legacy enrollment should have no configured panel")
	}
	source := ConfiguredPanelInbounds(settingsPath)
	for _, update := range []struct{ url, token string }{
		{"https://panel.example:5588/private", "test-first"},
		{"https://panel.example:5577/other", "test-second"},
	} {
		set("3x-ui", update.url)
		write(credentialPath, "Authorization: Bearer "+update.token+"\n")
		settings, header, err := loadPanelSettings(settingsPath)
		if err != nil {
			t.Fatal(err)
		}
		client, err := configuredPanelClient(settings, header)
		if err != nil {
			t.Fatal(err)
		}
		if client.base.String() != update.url || client.token != update.token {
			t.Fatal("panel settings were not reloaded")
		}
	}
	for _, path := range []string{settingsPath, credentialPath} {
		if err := os.Chmod(path, 0640); err != nil {
			t.Fatal(err)
		}
		if _, err := source(context.Background()); err == nil || strings.Contains(err.Error(), dir) {
			t.Fatal("expected a redacted error for a file readable by the group")
		}
		if err := os.Chmod(path, 0600); err != nil {
			t.Fatal(err)
		}
	}
	set("remnawave", "https://panel.example")
	if _, err := source(context.Background()); err == nil || err.Error() != "invalid Remnawave node UUID" {
		t.Fatal("Remnawave must require an explicit node UUID")
	}
}

func TestConfiguredRemnawaveNodeInbounds(t *testing.T) {
	if config := os.Getenv("TERVON_TEST_REMNA_CONFIG"); config != "" {
		inbounds, err := ConfiguredPanelInbounds(config)(context.Background())
		if err != nil || len(inbounds) != 1 || inbounds[0].Port != 7771 {
			t.Fatalf("unexpected node projection: %+v, %v", inbounds, err)
		}
		return
	}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("unexpected panel request")
		}
		switch r.URL.Path {
		case "/api/nodes/" + nodeID:
			fmt.Fprintf(w, `{"response":{"uuid":%q,"isDisabled":false,"configProfile":{"activeConfigProfileUuid":%q,"activeInbounds":[{"uuid":%q}]}}}`, nodeID, profileID, activeID)
		case "/api/config-profiles/" + profileID + "/inbounds":
			fmt.Fprintf(w, `{"response":[{"uuid":%q,"profileUuid":%q,"port":7771,"type":"vless","network":"tcp"},{"uuid":%q,"profileUuid":%q,"port":7772,"type":"trojan","network":"tcp"}]}`, activeID, profileID, inactiveID, profileID)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	credential := filepath.Join(dir, "api-header")
	config := filepath.Join(dir, "panel.json")
	ca := filepath.Join(dir, "ca.pem")
	body, err := json.Marshal(panelSettings{
		Type: "remnawave", BaseURL: server.URL, APICredential: credential, NodeUUID: nodeID,
	})
	if err != nil {
		t.Fatal(err)
	}
	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	for path, data := range map[string][]byte{
		config: body, credential: []byte("Authorization: Bearer test-token"), ca: cert,
	} {
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}

	// Start with an empty system-root cache so the TLS fixture can be trusted
	// without changing the client's production TLS configuration.
	cmd := exec.Command(os.Args[0], "-test.run=^TestConfiguredRemnawaveNodeInbounds$")
	cmd.Env = append(os.Environ(), "TERVON_TEST_REMNA_CONFIG="+config, "SSL_CERT_FILE="+ca)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("configured source: %v\n%s", err, output)
	}
}
