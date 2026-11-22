// m6-first-device prepares a single operator-approved test device.
// It never touches the live Bridge configuration or running services.
package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tervon-software/Tervon-TCP-Bridge/integrations/android-ingress/xui"
)

func main() {
	var csrPath, outDir, expectedCN string
	flag.StringVar(&csrPath, "csr", "", "Android-generated PEM CSR (no private key)")
	flag.StringVar(&outDir, "output", "", "new private staging directory (created once)")
	flag.StringVar(&expectedCN, "device-cn", "", "expected exact CSR common name")
	flag.Parse()
	if err := run(csrPath, outDir, expectedCN); err != nil {
		fmt.Fprintln(os.Stderr, "M6_FIRST_DEVICE=FAIL:", err)
		os.Exit(1)
	}
	fmt.Println("M6_FIRST_DEVICE=STAGED_ONLY")
	fmt.Println("M6_LIVE_ACL=UNCHANGED")
	fmt.Println("M6_DOCKER=UNCHANGED")
}

func run(csrPath, outDir, expectedCN string) error {
	const root = "/etc/tervon/bridge-android"
	if os.Geteuid() != 0 {
		return errors.New("root operator required")
	}
	if csrPath == "" || outDir == "" || expectedCN == "" || len(expectedCN) > 80 {
		return errors.New("required: --csr, --output, --device-cn")
	}
	if !filepath.IsAbs(outDir) || filepath.Clean(outDir) != outDir ||
		filepath.Dir(outDir) != root+"/private" {
		return errors.New("output must be a new direct child of the private Bridge directory")
	}
	if strings.ContainsAny(expectedCN, "/\\\x00\r\n") {
		return errors.New("invalid expected device name")
	}
	csrBytes, err := os.ReadFile(csrPath)
	if err != nil { return errors.New("cannot read device CSR") }
	csrBlock, rest := pem.Decode(csrBytes)
	if csrBlock == nil || csrBlock.Type != "CERTIFICATE REQUEST" || len(rest) != 0 {
		return errors.New("invalid device CSR")
	}
	req, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	if err != nil || req.CheckSignature() != nil || req.Subject.CommonName != expectedCN {
		return errors.New("CSR signature or device identity mismatch")
	}
	caPEM, err := os.ReadFile(root+"/config/pki/client-ca.crt")
	if err != nil { return errors.New("cannot read client CA certificate") }
	keyPEM, err := os.ReadFile(root+"/private/client-ca.key")
	if err != nil { return errors.New("cannot read client CA key") }
	caKey, err := parseSigner(keyPEM)
	if err != nil { return err }
	issued, err := xui.SignDeviceCSR(csrBytes, caPEM, caKey, time.Now().UTC(), 24*time.Hour)
	if err != nil { return err }
	serverJSON, err := os.ReadFile(root+"/config/server.json")
	if err != nil { return errors.New("cannot read Android ingress server configuration") }
	stagedJSON, err := xui.AddDeviceToACL(serverJSON, issued.Fingerprint, "vpn")
	if err != nil { return err }

	// Never overwrite existing enrollment material or live config.
	if err := os.Mkdir(outDir, 0700); err != nil {
		return errors.New("private output directory must not already exist")
	}
	// A failed staging leaves only a private, incomplete directory; no live changes.
	if err := writeNew(filepath.Join(outDir,"android-client.crt"), issued.PEM); err != nil { return err }
	if err := writeNew(filepath.Join(outDir,"server.staged.json"), stagedJSON); err != nil { return err }
	if err := writeNew(filepath.Join(outDir,"leaf.sha256"), []byte(issued.Fingerprint+"\n")); err != nil { return err }
	return nil
}

func parseSigner(p []byte) (crypto.Signer, error) {
	b, rest := pem.Decode(p)
	if b == nil || len(rest) != 0 {
		return nil, errors.New("invalid client CA private key PEM")
	}
	var key any
	var err error
	switch b.Type {
	case "PRIVATE KEY":
		key, err = x509.ParsePKCS8PrivateKey(b.Bytes)
	case "EC PRIVATE KEY":
		key, err = x509.ParseECPrivateKey(b.Bytes)
	default:
		return nil, errors.New("unsupported client CA private key encoding")
	}
	if err != nil { return nil, errors.New("invalid client CA private key") }
	s, ok := key.(*ecdsa.PrivateKey)
	if !ok { return nil, errors.New("client CA private key must be ECDSA") }
	return s, nil
}

func writeNew(path string, content []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil { return errors.New("cannot create new private output file") }
	if _, err := f.Write(content); err != nil { _ = f.Close(); return errors.New("cannot write staged output") }
	if err := f.Sync(); err != nil { _ = f.Close(); return errors.New("cannot sync staged output") }
	if err := f.Close(); err != nil { return errors.New("cannot close staged output") }
	return nil
}
