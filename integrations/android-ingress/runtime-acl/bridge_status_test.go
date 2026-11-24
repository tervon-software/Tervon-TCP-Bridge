package main

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBridgeStatusAccess(t *testing.T) {
	registered := []byte("registered certificate fixture")
	handle := bridgeStatusHandler(
		func(id [32]byte) bool { return id == sha256.Sum256(registered) },
		func() (bool, string, int64, uint64) { return true, "1.0.2", 75, 12 },
	)
	for _, tc := range []struct {
		name, method string
		cert         []byte
		want         int
	}{
		{"registered", http.MethodGet, registered, 200},
		{"outside ACL", http.MethodGet, []byte("other"), 403},
		{"without verified certificate", http.MethodGet, nil, 403},
		{"wrong method", http.MethodPost, registered, 405},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "https://bridge.example/v2/status", nil)
			if tc.cert != nil {
				req.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{
					{{Raw: tc.cert}},
				}}
			}
			out := httptest.NewRecorder()
			handle.ServeHTTP(out, req)
			if out.Code != tc.want {
				t.Fatalf("status: got %d, want %d", out.Code, tc.want)
			}
			if tc.want == 200 && out.Body.String() !=
				"{\"ready\":true,\"version\":\"1.0.2\",\"uptime_seconds\":75,\"tls13_h2_mtls_requests\":12}\n" {
				t.Fatalf("unexpected response: %s", out.Body.String())
			}
		})
	}
}
