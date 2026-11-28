package tdp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net"
	"testing"
	"time"
)

func TestTCPPathResumesBackendOverMTLS(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     []string{"tdp.test"},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(leaf)
	certificate := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{certificate}, ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs: roots, MinVersion: tls.VersionTLS13, NextProtos: []string{TCPProtocol},
	}
	clientTLS := &tls.Config{
		Certificates: []tls.Certificate{certificate}, RootCAs: roots, ServerName: "tdp.test",
		MinVersion: tls.VersionTLS13, NextProtos: []string{TCPProtocol},
	}
	registry, err := NewRegistry(1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	backend, remote := net.Pipe()
	defer backend.Close()
	defer remote.Close()
	dials := 0
	server := TCPServer{
		Registry:  registry,
		Authorize: func(peer [32]byte) bool { return peer == sha256.Sum256(der) },
		DialBackend: func(context.Context) (net.Conn, error) {
			dials++
			return backend, nil
		},
	}

	open := func(resume *SessionID) (SessionID, *Attachment) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		clientSide, serverSide := net.Pipe()
		defer clientSide.Close()
		defer serverSide.Close()
		type accepted struct {
			id   SessionID
			path *Attachment
			err  error
		}
		result := make(chan accepted, 1)
		go func() {
			id, path, err := server.AcceptPath(ctx, tls.Server(serverSide, serverTLS))
			result <- accepted{id, path, err}
		}()
		id, err := OpenTCPPath(ctx, tls.Client(clientSide, clientTLS), resume)
		if err != nil {
			t.Fatal(err)
		}
		got := <-result
		if got.err != nil || got.id != id {
			t.Fatalf("accept: id=%x err=%v", got.id, got.err)
		}
		return id, got.path
	}
	id, first := open(nil)
	registry.Detach(id, first)
	_, next := open(&id)
	conn, err := next.Backend()
	if err != nil || conn != backend || dials != 1 {
		t.Fatalf("resumed backend=%v dials=%d err=%v", conn == backend, dials, err)
	}
	if err := registry.End(id, next); err != nil {
		t.Fatal(err)
	}
}
