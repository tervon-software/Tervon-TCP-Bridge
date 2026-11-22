// Command catalog-sign produces a signed, offline Tervon Bridge catalog envelope.
// It accepts an existing Ed25519 PKCS#8 PEM key. It never generates keys or
// connects to the network, and it refuses to overwrite any output file.
package main

import (
    "crypto/ed25519"
    "crypto/x509"
    "encoding/pem"
    "errors"
    "flag"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "syscall"

    "github.com/tervon-software/Tervon-TCP-Bridge/integrations/android-ingress/xui/catalog"
)

func loadSigningKey(path string) (ed25519.PrivateKey, error) {
    if path == "" { return nil, errors.New("missing signing key path") }
    info, err := os.Lstat(path)
    if err != nil { return nil, fmt.Errorf("inspect signing key: %w", err) }
    if !info.Mode().IsRegular() || info.Mode().Perm() & 0077 != 0 {
        return nil, errors.New("signing key must be a regular private file with no group/other permissions")
    }
    // O_NOFOLLOW plus inode check protects the path against symlink races.
    f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
    if err != nil { return nil, errors.New("cannot open private key safely") }
    defer f.Close()
    actual, err := f.Stat()
    if err != nil || !actual.Mode().IsRegular() || actual.Mode().Perm() & 0077 != 0 || !os.SameFile(info, actual) {
        return nil, errors.New("signing key replaced or insecure")
    }
    if actual.Size() < 50 || actual.Size() > 16384 { return nil, errors.New("invalid signing key size") }
    data := make([]byte, actual.Size())
    n, err := io.ReadFull(f, data)
    if err != nil || n != len(data) { return nil, errors.New("cannot read complete signing key") }
    block, remainder := pem.Decode(data)
    if block == nil || block.Type != "PRIVATE KEY" || len(remainder) != 0 {
        return nil, errors.New("expected single unencrypted PKCS#8 private key PEM")
    }
    parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
    if err != nil { return nil, errors.New("invalid PKCS#8 key") }
    key, ok := parsed.(ed25519.PrivateKey)
    if !ok { return nil, errors.New("catalog signing key is not Ed25519") }
    return key, nil
}

func signOffline(keyPath, payloadPath, outputPath, keyID string) error {
    if keyPath == "" || payloadPath == "" || outputPath == "" || keyID == "" {
        return errors.New("all four flags are required")
    }
    key, err := loadSigningKey(keyPath)
    if err != nil { return err }
    info, err := os.Lstat(payloadPath)
    if err != nil || !info.Mode().IsRegular() || info.Size() > 65536 {
        return errors.New("payload must be an existing regular file of at most 65536 bytes")
    }
    payload, err := os.ReadFile(payloadPath)
    if err != nil { return errors.New("cannot read payload") }
    encoded, err := catalog.Sign(payload, keyID, key)
    if err != nil { return err }
    if filepath.Clean(outputPath) == filepath.Clean(keyPath) || filepath.Clean(outputPath) == filepath.Clean(payloadPath) {
        return errors.New("output must not equal an input path")
    }
    output, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
    if err != nil { return errors.New("output already exists or cannot be created") }
    defer output.Close()
    if _, err := output.Write(append(encoded, '\n')); err != nil {
        output.Close()
        _ = os.Remove(outputPath)
        return errors.New("cannot write signed envelope")
    }
    if err := output.Sync(); err != nil {
        output.Close()
        _ = os.Remove(outputPath)
        return errors.New("cannot sync signed envelope")
    }
    return nil
}

// FHS-based defaults apply only to the catalog publisher, NOT to every
// Bridge installation. All paths are overridable for unprivileged Linux setups.
const (
    defaultSigningKeyPath = "/etc/tervon/bridge/keys/catalog-ed25519.pk8.pem"
    defaultPayloadPath    = "/var/lib/tervon/bridge/catalog/catalog.json"
    defaultEnvelopePath   = "/var/lib/tervon/bridge/catalog/catalog.signed.json"
)

func main() {
    key := flag.String("key", defaultSigningKeyPath, "path to existing private Ed25519 PKCS#8 PEM (0600)")
    payload := flag.String("payload", defaultPayloadPath, "path to catalog JSON payload")
    out := flag.String("out", defaultEnvelopePath, "new output envelope path (must not exist)")
    id := flag.String("key-id", "", "trusted catalog signing key ID (required)")
    flag.Parse()
    if err := signOffline(*key, *payload, *out, *id); err != nil {
        fmt.Fprintln(os.Stderr, "catalog-sign: refused:", err)
        os.Exit(1)
    }
    fmt.Println("M6_SIGNED_CATALOG=CREATED")
}
