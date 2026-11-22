// Command catalog-keygen provisions a new Ed25519 publisher key and public APK trust asset.
// Offline and opt-in: it neither changes Bridge services nor installs an APK.
package main

import (
    "crypto/ed25519"
    "crypto/rand"
    "crypto/sha256"
    "crypto/x509"
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "encoding/pem"
    "errors"
    "flag"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
)

const (
    defaultPrivatePath = "/etc/tervon/bridge/keys/catalog-ed25519.pk8.pem"
    defaultPublicPath  = "/var/lib/tervon/bridge/catalog/bridge_catalog_trust_v1.json"
)

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func parentDirectory(path string, private bool) error {
    dir := filepath.Dir(path)
    st, err := os.Lstat(dir)
    if err != nil || !st.IsDir() {
        return errors.New("destination parent must be an existing real directory")
    }
    if private && st.Mode().Perm() & 0077 != 0 {
        return errors.New("private key directory must deny group/other access (0700 recommended)")
    }
    return nil
}

func provision(keyPath, publicPath, keyID string) (string, error) {
    if !idPattern.MatchString(keyID) {
        return "", errors.New("invalid catalog signing key ID")
    }
    if keyPath == "" || publicPath == "" {
        return "", errors.New("both output paths required")
    }
    keyAbs, err := filepath.Abs(keyPath)
    if err != nil {
        return "", errors.New("invalid private key path")
    }
    publicAbs, err := filepath.Abs(publicPath)
    if err != nil {
        return "", errors.New("invalid public trust asset path")
    }
    if keyAbs == publicAbs {
        return "", errors.New("private and public output must differ")
    }
    if err := parentDirectory(keyAbs, true); err != nil {
        return "", err
    }
    if err := parentDirectory(publicAbs, false); err != nil {
        return "", err
    }

    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return "", errors.New("secure random key generation failed")
    }
    der, err := x509.MarshalPKCS8PrivateKey(priv)
    if err != nil {
        return "", errors.New("Ed25519 PKCS#8 encoding failed")
    }
    privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
    asset, err := json.MarshalIndent(struct {
        Format string            `json:"format"`
        Keys   map[string]string `json:"keys"`
    }{
        Format: "tervon.bridge.catalog.trust.v1",
        Keys: map[string]string{keyID: base64.StdEncoding.EncodeToString(pub)},
    }, "", "  ")
    if err != nil {
        return "", errors.New("cannot encode public trust asset")
    }

    // O_EXCL refuses overwrite and destination symlinks. If the public file
    // cannot be written, remove only the key newly created by this invocation.
    f, err := os.OpenFile(keyAbs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
    if err != nil {
        return "", errors.New("private key target already exists or cannot be created")
    }
    privateCreated := true
    defer func() {
        if privateCreated {
            _ = os.Remove(keyAbs)
        }
    }()
    if _, err = f.Write(privatePEM); err != nil {
        _ = f.Close()
        return "", errors.New("cannot write private key")
    }
    if err = f.Sync(); err != nil {
        _ = f.Close()
        return "", errors.New("cannot sync private key")
    }
    if err = f.Close(); err != nil {
        return "", errors.New("cannot close private key")
    }

    publicFile, err := os.OpenFile(publicAbs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
    if err != nil {
        return "", errors.New("public trust asset target already exists or cannot be created")
    }
    if _, err = publicFile.Write(append(asset, '\n')); err != nil {
        _ = publicFile.Close()
        _ = os.Remove(publicAbs)
        return "", errors.New("cannot write public trust asset")
    }
    if err = publicFile.Sync(); err != nil {
        _ = publicFile.Close()
        _ = os.Remove(publicAbs)
        return "", errors.New("cannot sync public trust asset")
    }
    if err = publicFile.Close(); err != nil {
        _ = os.Remove(publicAbs)
        return "", errors.New("cannot close public trust asset")
    }
    privateCreated = false
    fingerprint := sha256.Sum256(pub)
    return hex.EncodeToString(fingerprint[:]), nil
}

func main() {
    key := flag.String("key", defaultPrivatePath, "new private PKCS#8 path (must not exist)")
    public := flag.String("trust-out", defaultPublicPath, "new public APK trust asset path (must not exist)")
    id := flag.String("key-id", "", "publisher signing key ID (required)")
    flag.Parse()
    fingerprint, err := provision(*key, *public, *id)
    if err != nil {
        fmt.Fprintln(os.Stderr, "catalog-keygen: refused:", err)
        os.Exit(1)
    }
    fmt.Println("M6_CATALOG_PUBLISHER_KEY=CREATED")
    fmt.Println("M6_PUBLIC_KEY_SHA256=" + fingerprint)
    fmt.Println("M6_PRIVATE_KEY_NOT_PRINTED=YES")
}
