package xui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListInboundsSlim(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/private-panel/panel/api/inbounds/list/slim" {
			t.Errorf("unexpected request path or method")
		}
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Errorf("missing bearer token")
		}
		fmt.Fprint(w, `{"success":true,"obj":[{"id":2,"port":55636,"protocol":"vless","enable":true,"streamSettings":{"network":"xhttp","security":"tls"}},{"id":5,"port":55535,"protocol":"vless","enable":true,"streamSettings":"{\\"network\\":\\"tcp\\",\\"security\\":\\"tls\\"}"},{"id":7,"port":30000,"protocol":"trojan","enable":false,"streamSettings":{"network":"tcp","security":"tls"}}]}`)
	}))
	defer s.Close()
	c := adapter(t, s)
	v, err := c.ListInbounds(context.Background())
	if err != nil { t.Fatal(err) }
	if len(v) != 3 || v[0].Port != 55636 || v[1].Port != 55535 || v[1].Network != "tcp" || v[2].Enabled {
		t.Fatalf("unexpected redacted inbound snapshot: %+v", v)
	}
}

func TestListInboundsFailsClosed(t *testing.T) {
	for _, tc := range []struct{name, body string}{
		{"not-array", `{"success":true,"obj":{}}`},
		{"missing-port", `{"success":true,"obj":[{"id":5,"protocol":"vless","enable":true,"streamSettings":{}}]}`},
		{"invalid-port", `{"success":true,"obj":[{"id":5,"port":65536,"protocol":"vless","enable":true,"streamSettings":{}}]}`},
		{"duplicate-id", `{"success":true,"obj":[{"id":5,"port":1234,"protocol":"vless","enable":true,"streamSettings":{}},{"id":5,"port":1235,"protocol":"vless","enable":true,"streamSettings":{}}]}`},
		{"missing-enable", `{"success":true,"obj":[{"id":5,"port":1234,"protocol":"vless","streamSettings":{}}]}`},
		{"bad-settings", `{"success":true,"obj":[{"id":5,"port":1234,"protocol":"vless","enable":true,"streamSettings":"not-json"}]}`},
		{"api-failed", `{"success":false,"obj":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, tc.body)
			}))
			defer s.Close()
			c := adapter(t, s)
			if _, err := c.ListInbounds(context.Background()); err == nil {
				t.Fatal("accepted invalid inbound response")
			}
		})
	}
}

func TestListInboundsHTTPFailureDoesNotLeakSecrets(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer s.Close()
	c := adapter(t, s)
	_, err := c.ListInbounds(context.Background())
	if err == nil || strings.Contains(err.Error(), "test-secret") || strings.Contains(err.Error(), "private-panel") {
		t.Fatalf("unsafe error: %v", err)
	}
}
