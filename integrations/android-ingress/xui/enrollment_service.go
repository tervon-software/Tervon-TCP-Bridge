package xui

import (
    "context"
    "crypto"
    "time"
)

// IssueBridgeClientCertificate issues the already-defined short-lived
// Bridge mTLS client certificate from an existing Android P-256 CSR.
//
// This is transport enrollment, NOT Xray/VLESS subscriber authorization.
// A valid VPN subscription is NOT required to receive a Bridge certificate:
// Xray remains responsible for VPN identity and access.
//
// NOT an HTTP server and does NOT install/reload the Bridge ACL.
// The deployed registrar must ensure this request is for a configured Bridge
// and apply the returned fingerprint to that Bridge before announcing success.
// Reuse the existing client CA and SignDeviceCSR; no new crypto or credentials.
func IssueBridgeClientCertificate(
    ctx context.Context,
    csrPEM, clientCAPEM []byte,
    clientCAKey crypto.Signer,
    now time.Time,
) (IssuedCertificate, error) {
    if err := ctx.Err(); err != nil {
        return IssuedCertificate{}, err
    }
    return SignDeviceCSR(csrPEM, clientCAPEM, clientCAKey, now, 24*time.Hour)
}
