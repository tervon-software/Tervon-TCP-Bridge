package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestParseSigner(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { t.Fatal(err) }
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil { t.Fatal(err) }
	encoded := pem.EncodeToMemory(&pem.Block{Type:"PRIVATE KEY",Bytes:der})
	if _, err := parseSigner(encoded); err != nil { t.Fatal(err) }
	if _, err := parseSigner([]byte("invalid")); err == nil { t.Fatal("accepted invalid key") }
	if _, err := parseSigner(append(encoded,[]byte("EXTRA")...)); err == nil {
		t.Fatal("accepted trailing key material")
	}
}

func TestWriteNewNeverOverwrites(t *testing.T) {
	p:=filepath.Join(t.TempDir(),"certificate.crt")
	if err:=writeNew(p,[]byte("original"));err!=nil {t.Fatal(err)}
	if err:=writeNew(p,[]byte("replacement"));err==nil {t.Fatal("overwrote existing file")}
	b,err:=os.ReadFile(p)
	if err!=nil || string(b)!="original" {t.Fatal("existing file was changed")}
}
