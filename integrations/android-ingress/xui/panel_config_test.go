package xui

import (
	"context"
	"encoding/json"
	"os"
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
		client, err := configuredPanelClient(settingsPath)
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
	if _, err := source(context.Background()); err == nil || err.Error() != "Remnawave inbound adapter unavailable" {
		t.Fatal("Remnawave must wait for its inbound adapter")
	}
}
