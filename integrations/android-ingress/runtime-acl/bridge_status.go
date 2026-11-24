package main

import (
	"crypto/sha256"
	"encoding/json"
	"net/http"
)

func bridgeStatusHandler(allowed func([32]byte) bool, status func() (bool, string, int64, uint64)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 ||
			len(r.TLS.VerifiedChains[0]) == 0 ||
			!allowed(sha256.Sum256(r.TLS.VerifiedChains[0][0].Raw)) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		ready, version, uptime, requests := status()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Ready         bool   `json:"ready"`
			Version       string `json:"version"`
			UptimeSeconds int64  `json:"uptime_seconds"`
			TLSRequests   uint64 `json:"tls13_h2_mtls_requests"`
		}{ready, version, uptime, requests})
	})
}
