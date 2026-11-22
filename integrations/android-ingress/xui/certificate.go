package xui

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"math/big"
	"time"
)

// IssuedCertificate contains only the public device certificate and its
// SHA-256 DER fingerprint, NOT the device private key or a CA private key.
type IssuedCertificate struct {
	PEM         []byte
	Fingerprint string
}

// SignDeviceCSR is a cryptographic building block for the M6 registrar.
// CRITICAL: The caller MUST first authenticate subscription ownership and
// authorize the individual device. Never expose this function as a public
// endpoint or sign a CSR based solely on email, subId, IP, or UUID lookup.
// now is explicit for deterministic tests and validity enforcement.
func SignDeviceCSR(csrPEM, caPEM []byte, caKey crypto.Signer, now time.Time, lifetime time.Duration) (IssuedCertificate, error) {
	var zero IssuedCertificate
	if caKey == nil || lifetime <= 0 || lifetime > 24*time.Hour {
		return zero, errors.New("invalid CA signer or certificate lifetime")
	}
	csrBlock, rest := pem.Decode(csrPEM)
	if csrBlock == nil || csrBlock.Type != "CERTIFICATE REQUEST" || len(rest) != 0 {
		return zero, errors.New("invalid CSR PEM")
	}
	csr, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	if err != nil || csr.CheckSignature() != nil {
		return zero, errors.New("invalid CSR signature")
	}
	// Android Bridge uses an individual ECDSA P-256 key, not a shared key.
	key, ok := csr.PublicKey.(*ecdsa.PublicKey)
	if !ok || key.Curve != elliptic.P256() {
		return zero, errors.New("unsupported device public key")
	}
	caBlock, caRest := pem.Decode(caPEM)
	if caBlock == nil || caBlock.Type != "CERTIFICATE" || len(caRest) != 0 {
		return zero, errors.New("invalid client CA PEM")
	}
	ca, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil || !ca.IsCA || ca.KeyUsage&x509.KeyUsageCertSign == 0 {
		return zero, errors.New("invalid client CA")
	}
	caPublic, equalSupported := ca.PublicKey.(interface{ Equal(crypto.PublicKey) bool })
	if !equalSupported || !caPublic.Equal(caKey.Public()) {
		return zero, errors.New("client CA key mismatch")
	}
	if now.Before(ca.NotBefore) || now.After(ca.NotAfter) || now.Add(lifetime).After(ca.NotAfter) {
		return zero, errors.New("client CA not valid for requested lifetime")
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return zero, errors.New("cannot generate certificate serial")
	}
	if serial.Sign() == 0 {
		serial.SetInt64(1)
	}
	leaf := &x509.Certificate{
		SerialNumber: serial,
		Subject: csr.Subject,
		NotBefore: now.Add(-time.Minute),
		NotAfter: now.Add(lifetime),
		KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA: false,
	}
	der, err := x509.CreateCertificate(rand.Reader, leaf, ca, csr.PublicKey, caKey)
	if err != nil {
		return zero, errors.New("failed to sign device CSR")
	}
	sum := sha256.Sum256(der)
	return IssuedCertificate{
		PEM: pem.EncodeToMemory(&pem.Block{Type:"CERTIFICATE", Bytes:der}),
		Fingerprint: hex.EncodeToString(sum[:]),
	}, nil
}
