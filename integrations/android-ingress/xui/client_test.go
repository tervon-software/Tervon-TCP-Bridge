package xui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func server(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	s := httptest.NewTLSServer(handler)
	t.Cleanup(s.Close)
	return s
}
func adapter(t *testing.T, s *httptest.Server) *Client {
	t.Helper()
	c, err := NewClient(s.URL+"/private-panel", "test-secret")
	if err != nil { t.Fatal(err) }
	c.httpClient = s.Client()
	return c
}

func TestListAndLookup(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer test-secret" || r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
			t.Errorf("invalid HTTP request"); w.WriteHeader(403); return
		}
		switch r.URL.EscapedPath() {
		case "/private-panel/panel/api/clients/list":
			_ = json.NewEncoder(w).Encode(map[string]any{"success":true,"obj":[]map[string]any{{"id":7,"email":"Diag","subId":"private","uuid":"private","enable":true,"expiryTime":0,"totalGB":0,"limitHwid":1}}})
		case "/private-panel/panel/api/clients/get/Diag":
			_ = json.NewEncoder(w).Encode(map[string]any{"success":true,"obj":map[string]any{"client":map[string]any{"id":7,"email":"Diag","subId":"private","uuid":"private","enable":true}}})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path); http.NotFound(w, r)
		}
	})
	c := adapter(t, s)
	list, err := c.List(context.Background())
	if err != nil || len(list) != 1 || list[0].ID != 7 { t.Fatalf("list: %v, %d", err, len(list)) }
	detail, err := c.GetByEmail(context.Background(), "Diag")
	if err != nil || detail.ID != 7 { t.Fatalf("detail: %v", err) }
}

func TestRejectInsecureAndInvalidToken(t *testing.T) {
	for _, tc := range []struct{url, token string}{
		{"http://localhost:5588", "token"},
		{"https://user:pass@example.org", "token"},
		{"https://example.org", "a\nb"},
		{"https://example.org", ""},
	} {
		if _, err := NewClient(tc.url, tc.token); err == nil { t.Error("accepted invalid input") }
	}
}
func TestHeaderFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "header")
	if err := os.WriteFile(path, []byte("Authorization: Bearer test-secret\n"), 0600); err != nil { t.Fatal(err) }
	if _, err := NewClientFromHeader("https://example.org/panelprefix", path); err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, []byte("Authorization: Bearer a\nInjected: something\n"), 0600); err != nil { t.Fatal(err) }
	if _, err := NewClientFromHeader("https://example.org", path); err == nil { t.Fatal("accepted multiple headers") }
}
func TestFailClosedOnPanelErrors(t *testing.T) {
	for _, tc := range []struct{status int; body string}{
		{404, "not found"},
		{200, `{"success":false,"obj":{}}`},
		{200, `{"success":true,"obj":null}`},
		{200, `{"success":true,"obj":{"other":1}}`},
	} {
		s := server(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); _, _ = w.Write([]byte(tc.body)) })
		c := adapter(t, s)
		_, err := c.GetByEmail(context.Background(), "Diag")
		if err == nil { t.Errorf("accepted invalid response: %d", tc.status) }
		if err != nil && (strings.Contains(err.Error(), "test-secret") || strings.Contains(err.Error(), "private-panel")) { t.Errorf("credential/route leak: %v", err) }
		s.Close()
	}
}
func TestLookupEmailMismatch(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"success":true,"obj":{"client":{"id":1,"email":"another"}}}`)) })
	c := adapter(t, s)
	if _, err := c.GetByEmail(context.Background(), "Diag"); err == nil { t.Fatal("accepted mismatched identity") }
}
func TestEmailEscaping(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/private-panel/panel/api/clients/get/a%25test@example.org" { t.Errorf("unexpected escaped path: %q", r.URL.EscapedPath()) }
		_, _ = w.Write([]byte(`{"success":true,"obj":{"client":{"id":1,"email":"a%test@example.org"}}}`))
	})
	c := adapter(t, s)
	if _, err := c.GetByEmail(context.Background(), "a%test@example.org"); err != nil { t.Fatal(err) }
}
