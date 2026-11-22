package main

import (
    "crypto/ed25519"
    "crypto/sha256"
    "crypto/x509"
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "encoding/pem"
    "os"
    "path/filepath"
    "testing"
)

func TestProvisionAndRefuseOverwrite(t *testing.T) {
    dir := t.TempDir()
    privateDir := filepath.Join(dir, "keys")
    if err := os.Mkdir(privateDir, 0700); err != nil { t.Fatal(err) }
    keyPath, assetPath := filepath.Join(privateDir, "catalog.pem"), filepath.Join(dir, "trust.json")
    fingerprint, err := provision(keyPath, assetPath, "primary_2026")
    if err != nil { t.Fatal(err) }
    keyBytes, err := os.ReadFile(keyPath)
    if err != nil { t.Fatal(err) }
    block, _ := pem.Decode(keyBytes)
    if block == nil { t.Fatal("missing PKCS8 PEM") }
    loaded, err := x509.ParsePKCS8PrivateKey(block.Bytes)
    if err != nil { t.Fatal(err) }
    priv, ok := loaded.(ed25519.PrivateKey)
    if !ok { t.Fatal("not Ed25519") }
    assetBytes, err := os.ReadFile(assetPath)
    if err != nil { t.Fatal(err) }
    var asset struct {
        Format string `json:"format"`
        Keys map[string]string `json:"keys"`
    }
    if err := json.Unmarshal(assetBytes, &asset); err != nil { t.Fatal(err) }
    if asset.Format != "tervon.bridge.catalog.trust.v1" { t.Fatal("bad trust format") }
    decoded, err := base64.StdEncoding.DecodeString(asset.Keys["primary_2026"])
    if err != nil { t.Fatal(err) }
    if string(decoded) != string(priv.Public().(ed25519.PublicKey)) {
        t.Fatal("public/private mismatch")
    }
    digest := sha256.Sum256(decoded)
    if hex.EncodeToString(digest[:]) != fingerprint { t.Fatal("wrong public fingerprint") }
    if _, err := provision(keyPath, assetPath, "primary_2026"); err == nil {
        t.Fatal("private key overwrite accepted")
    }
    after, err := os.ReadFile(keyPath)
    if err != nil || string(after) != string(keyBytes) { t.Fatal("private key modified") }
}

func TestPrivateDirectoryAndIDValidation(t *testing.T) {
    dir := t.TempDir()
    insecure := filepath.Join(dir, "insecure")
    if err := os.Mkdir(insecure, 0755); err != nil { t.Fatal(err) }
    if _, err := provision(filepath.Join(insecure, "key"), filepath.Join(dir, "trust"), "primary"); err == nil {
        t.Fatal("insecure private directory accepted")
    }
    if _, err := provision(filepath.Join(dir, "key"), filepath.Join(dir, "trust"), "bad id!"); err == nil {
        t.Fatal("invalid id accepted")
    }
}

func TestSecondOutputCollisionRollsBackNewKey(t *testing.T) {
    dir := t.TempDir()
    keys := filepath.Join(dir, "keys")
    if err := os.Mkdir(keys, 0700); err != nil { t.Fatal(err) }
    existing := filepath.Join(dir, "existing")
    if err := os.WriteFile(existing, []byte("original"), 0600); err != nil { t.Fatal(err) }
    key := filepath.Join(keys, "key")
    if _, err := provision(key, existing, "primary"); err == nil {
        t.Fatal("public file overwrite accepted")
    }
    if _, err := os.Stat(key); !os.IsNotExist(err) { t.Fatal("partial private key remains") }
    b, err := os.ReadFile(existing)
    if err != nil || string(b) != "original" { t.Fatal("public target modified") }
}
