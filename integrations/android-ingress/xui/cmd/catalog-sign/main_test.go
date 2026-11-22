package main

import (
    "crypto/ed25519"
    "crypto/rand"
    "crypto/x509"
    "encoding/base64"
    "encoding/json"
    "encoding/pem"
    "os"
    "path/filepath"
    "testing"
)

func writeTestKey(t *testing.T, dir string) (string, ed25519.PublicKey) {
    t.Helper()
    pub, key, err := ed25519.GenerateKey(rand.Reader)
    if err != nil { t.Fatal(err) }
    encoded, err := x509.MarshalPKCS8PrivateKey(key)
    if err != nil { t.Fatal(err) }
    path := filepath.Join(dir, "test-private.pem")
    if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded}), 0600); err != nil { t.Fatal(err) }
    return path, pub
}

func TestSignOfflineValidAndRefusesOverwrite(t *testing.T) {
    dir := t.TempDir()
    key, pub := writeTestKey(t, dir)
    payload := filepath.Join(dir, "payload.json")
    out := filepath.Join(dir, "catalog.json")
    body := []byte(`{"version":1,"serial":5,"issued_at":1700000000,"expires_at":1700086400,"servers":[]}`)
    if err := os.WriteFile(payload, body, 0600); err != nil { t.Fatal(err) }
    if err := signOffline(key, payload, out, "primary_2026"); err != nil { t.Fatal(err) }
    raw, err := os.ReadFile(out)
    if err != nil { t.Fatal(err) }
    var result struct {
        PayloadBase64 string `json:"payload_b64"`
        SignatureBase64 string `json:"signature_b64"`
    }
    if err := json.Unmarshal(raw, &result); err != nil { t.Fatal(err) }
    sig, err := base64.StdEncoding.DecodeString(result.SignatureBase64)
    if err != nil { t.Fatal(err) }
    prefix := []byte("Tervon-Bridge-Catalog-v1\x00")
    if !ed25519.Verify(pub, append(prefix, body...), sig) { t.Fatal("signature mismatch") }
    if err := signOffline(key, payload, out, "primary_2026"); err == nil { t.Fatal("overwrite accepted") }
    saved, err := os.ReadFile(out)
    if err != nil || string(saved) != string(raw) { t.Fatal("output changed on refused overwrite") }
}

func TestKeyPermissionsAndSymlinkAreRefused(t *testing.T) {
    dir := t.TempDir()
    key, _ := writeTestKey(t, dir)
    if err := os.Chmod(key, 0644); err != nil { t.Fatal(err) }
    if _, err := loadSigningKey(key); err == nil { t.Fatal("insecure file accepted") }
    if err := os.Chmod(key, 0600); err != nil { t.Fatal(err) }
    link := filepath.Join(dir, "link.pem")
    if err := os.Symlink(key, link); err != nil { t.Fatal(err) }
    if _, err := loadSigningKey(link); err == nil { t.Fatal("symlink accepted") }
}

func TestInvalidInputAndNoOutput(t *testing.T) {
    dir := t.TempDir()
    key, _ := writeTestKey(t, dir)
    payload := filepath.Join(dir, "payload")
    out := filepath.Join(dir, "signed")
    if err := os.WriteFile(payload, []byte("not-json"), 0600); err != nil { t.Fatal(err) }
    if err := signOffline(key, payload, out, "primary"); err == nil { t.Fatal("invalid JSON accepted") }
    if _, err := os.Stat(out); !os.IsNotExist(err) { t.Fatal("output created after invalid data") }
    if err := signOffline(key, payload, key, "primary"); err == nil { t.Fatal("output equals key accepted") }
}

func TestFHSDefaultsAreAbsoluteAndDistinct(t *testing.T) {
    paths := []string{defaultSigningKeyPath, defaultPayloadPath, defaultEnvelopePath}
    if !filepath.IsAbs(paths[0]) || !filepath.IsAbs(paths[1]) || !filepath.IsAbs(paths[2]) {
        t.Fatal("catalog publisher defaults must be absolute paths")
    }
    if paths[0] != "/etc/tervon/bridge/keys/catalog-ed25519.pk8.pem" {
        t.Fatal("private signing key not under FHS configuration path")
    }
    if paths[1] != "/var/lib/tervon/bridge/catalog/catalog.json" ||
       paths[2] != "/var/lib/tervon/bridge/catalog/catalog.signed.json" {
        t.Fatal("catalog state paths not under FHS persistent state")
    }
    if paths[0] == paths[1] || paths[0] == paths[2] || paths[1] == paths[2] {
        t.Fatal("private key, payload, signed envelope must be distinct files")
    }
}
