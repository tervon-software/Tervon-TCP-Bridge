package catalog

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestSignAndVerifyDomainSeparatedPayload(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"version":1,"serial":1,"issued_at":1700000000,"expires_at":1700086400,"servers":[]}`)
	signed, err := Sign(payload, "bridge_catalog_primary", priv)
	if err != nil {
		t.Fatal(err)
	}
	var envelope Envelope
	if err := json.Unmarshal(signed, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Format != Format || envelope.KeyID != "bridge_catalog_primary" {
		t.Fatal("unexpected signing envelope metadata")
	}
	body, err := base64.StdEncoding.DecodeString(envelope.PayloadBase64)
	if err != nil || !bytes.Equal(body, payload) {
		t.Fatal("signed payload was altered")
	}
	sig, err := base64.StdEncoding.DecodeString(envelope.SignatureBase64)
	if err != nil || len(sig) != ed25519.SignatureSize {
		t.Fatal("invalid signature encoding")
	}
	message := append([]byte(domain), body...)
	if !ed25519.Verify(pub, message, sig) {
		t.Fatal("correct signed catalog rejected")
	}
	modified := append([]byte(nil), message...)
	modified[len(modified)-2] ^= 1
	if ed25519.Verify(pub, modified, sig) {
		t.Fatal("tampered catalog accepted")
	}
	if ed25519.Verify(pub, body, sig) {
		t.Fatal("missing domain prefix accepted")
	}
	otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
	if ed25519.Verify(otherPub, message, sig) {
		t.Fatal("wrong signing key accepted")
	}
}

func TestRejectInvalidSignRequests(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	tests := []struct {
		name string
		payload []byte
		keyID string
		key ed25519.PrivateKey
	}{
		{"bad key id", []byte("{}"), "bad key!", priv},
		{"empty key id", []byte("{}"), "", priv},
		{"bad JSON", []byte("{invalid"), "primary", priv},
		{"empty payload", []byte(""), "primary", priv},
		{"oversized payload", []byte("[" + string(bytes.Repeat([]byte(" "), maxPayloadBytes)) + "]"), "primary", priv},
		{"missing signing key", []byte("{}"), "primary", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Sign(tc.payload, tc.keyID, tc.key); err == nil {
				t.Fatal("invalid signing request accepted")
			}
		})
	}
}
