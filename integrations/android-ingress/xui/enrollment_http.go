package xui

import (
    "crypto"
    "encoding/json"
    "errors"
    "io"
    "net/http"
    "net/url"
    "strings"
    "time"
)

// BridgeEnrollmentHTTP provides the two operations requested for M6:
// discover whether this server has Bridge, then sign an Android CSR.
// VPN user authorization remains in Xray; no subscription eligibility check.
// Mount on the existing subscription HTTPS server, not a second public service.
//
// Activation MUST atomically install the issued leaf fingerprint into the
// corresponding configured Bridge ACL AND make that ACL effective before
// returning nil. Current v1.0.1 reads its ACL at process startup.
// This handler never silently skips activation or changes live services.
type BridgeEnrollmentHTTP struct {
    RemoteURL string
    ServerName string
    Service string
    ClientCAPEM []byte
    ServerCAPEM []byte // existing public server trust root for Android Bridge TLS
    ClientCAKey crypto.Signer
    Activate func(fingerprint string) error
    Now func() time.Time
}

type bridgeDiscoveryResponse struct {
    Available bool `json:"available"`
    RemoteURL string `json:"remote_url,omitempty"`
    ServerName string `json:"server_name,omitempty"`
    Service string `json:"service,omitempty"`
    ClientCA string `json:"client_ca,omitempty"`
    ServerCA string `json:"server_ca,omitempty"`
}

type bridgeEnrollmentResponse struct {
    Certificate string `json:"certificate"`
}

// Handler intentionally exposes only GET discovery and POST CSR enrollment.
// It assumes its parent subscription web server provides HTTPS; when TLS is
// terminated by a trusted reverse proxy, the parent must enforce HTTPS there.
func (h *BridgeEnrollmentHTTP) Handler() (http.Handler, error) {
    if h == nil || h.ClientCAKey == nil || h.Activate == nil ||
        len(h.ClientCAPEM) == 0 || len(h.ServerCAPEM) == 0 || h.Service == "" || h.ServerName == "" {
        return nil, errors.New("Bridge registration is not configured")
    }
    u, err := url.Parse(h.RemoteURL)
    if err != nil || u.Scheme != "https" || u.Host == "" ||
        u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
        u.Path != "/v2/stream" || u.RawPath != "" {
        return nil, errors.New("invalid configured Bridge endpoint")
    }
    now := h.Now
    if now == nil { now = time.Now }
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Cache-Control", "no-store")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        switch r.Method {
        case http.MethodGet:
            // Discovery must not create a device key or issue a certificate.
            if r.URL.RawQuery != "" {
                http.Error(w, "unsupported query", http.StatusBadRequest)
                return
            }
            w.Header().Set("Content-Type", "application/json")
            _ = json.NewEncoder(w).Encode(bridgeDiscoveryResponse{
                Available:true, RemoteURL:h.RemoteURL,
                ServerName:h.ServerName,Service:h.Service,
                ClientCA:string(h.ClientCAPEM), ServerCA:string(h.ServerCAPEM),
            })
        case http.MethodPost:
            if r.URL.RawQuery != "" ||
                !strings.HasPrefix(r.Header.Get("Content-Type"), "application/pkcs10") {
                http.Error(w, "expected CSR", http.StatusUnsupportedMediaType)
                return
            }
            // Cap the input so anonymous callers cannot force unbounded reads.
            data, readErr := io.ReadAll(io.LimitReader(r.Body, 8193))
            if readErr != nil || len(data) == 0 || len(data) > 8192 {
                http.Error(w, "invalid CSR", http.StatusBadRequest)
                return
            }
            cert, err := IssueBridgeClientCertificate(
                r.Context(), data, h.ClientCAPEM, h.ClientCAKey, now(),
            )
            if err != nil {
                http.Error(w, "invalid certificate request", http.StatusBadRequest)
                return
            }
            // A certificate without an effective Bridge ACL is not success.
            if err := h.Activate(cert.Fingerprint); err != nil {
                http.Error(w, "Bridge registration unavailable", http.StatusServiceUnavailable)
                return
            }
            w.Header().Set("Content-Type", "application/json")
            _ = json.NewEncoder(w).Encode(bridgeEnrollmentResponse{
                Certificate:string(cert.PEM),
            })
        default:
            w.Header().Set("Allow","GET, POST")
            http.Error(w,"method not allowed",http.StatusMethodNotAllowed)
        }
    }),nil
}
