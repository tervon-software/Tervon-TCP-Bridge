// Package catalog signs the exact Bridge catalog payload for offline distribution.
//
// The signing key is supplied by the caller. It must never be placed in this
// repository, an APK, the running Bridge Decapsulator, or a public endpoint.
package catalog

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"regexp"
)

const (
	Format = "tervon.bridge.catalog.envelope.v1"
	domain = "Tervon-Bridge-Catalog-v1\x00"
	maxPayloadBytes = 65536
)

var keyIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

type Envelope struct {
	Format string `json:"format"`
	KeyID string `json:"key_id"`
	PayloadBase64 string `json:"payload_b64"`
	SignatureBase64 string `json:"signature_b64"`
}

// Sign signs exact UTF-8 JSON payload bytes, with a protocol-specific domain.
// Android validates the signed schema after checking the signature.
func Sign(payload []byte, keyID string, privateKey ed25519.PrivateKey) ([]byte, error) {
	if !keyIDPattern.MatchString(keyID) {
		return nil, errors.New("invalid signing key id")
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid signing key size")
	}
	if len(payload) == 0 || len(payload) > maxPayloadBytes || !json.Valid(payload) {
		return nil, errors.New("invalid catalog JSON payload")
	}
	message := make([]byte, 0, len(domain)+len(payload))
	message = append(message, domain...)
	message = append(message, payload...)
	signature := ed25519.Sign(privateKey, message)
	return json.Marshal(Envelope{
		Format: Format,
		KeyID: keyID,
		PayloadBase64: base64.StdEncoding.EncodeToString(payload),
		SignatureBase64: base64.StdEncoding.EncodeToString(signature),
	})
}
