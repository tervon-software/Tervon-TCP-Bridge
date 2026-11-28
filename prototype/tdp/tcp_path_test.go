package tdp

import (
	"bytes"
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

	open := func(resume *SessionID) (SessionID, *Attachment, *tls.Conn, *tls.Conn, func()) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		clientSide, serverSide := net.Pipe()
		closePath := func() {
			_ = clientSide.Close()
			_ = serverSide.Close()
		}
		t.Cleanup(closePath)
		clientConn := tls.Client(clientSide, clientTLS)
		serverConn := tls.Server(serverSide, serverTLS)
		type accepted struct {
			id   SessionID
			path *Attachment
			err  error
		}
		result := make(chan accepted, 1)
		go func() {
			id, path, err := server.AcceptPath(ctx, serverConn)
			result <- accepted{id, path, err}
		}()
		id, err := OpenTCPPath(ctx, clientConn, resume)
		if err != nil {
			t.Fatal(err)
		}
		got := <-result
		if got.err != nil || got.id != id {
			t.Fatalf("accept: id=%x err=%v", got.id, got.err)
		}
		for _, conn := range []*tls.Conn{clientConn, serverConn} {
			if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
				t.Fatal(err)
			}
		}
		return id, got.path, clientConn, serverConn, closePath
	}
	id, first, clientConn, serverConn, closeFirst := open(nil)
	clientTransfer, err := NewTransfer(1024)
	if err != nil {
		t.Fatal(err)
	}
	serverTransfer, err := first.Transfer(1024)
	if err != nil {
		t.Fatal(err)
	}
	clientPath := NewFramedPath(clientConn, clientTransfer)
	serverPath := NewFramedPath(serverConn, serverTransfer)
	var received bytes.Buffer
	sent := make(chan error, 1)
	go func() { sent <- clientPath.Send([]byte("payload")) }()
	if ack, err := serverPath.Receive(&received); err != nil || ack == nil || ack.Offset != 7 {
		t.Fatalf("first receive: ACK=%v err=%v", ack, err)
	}
	if err := <-sent; err != nil {
		t.Fatal(err)
	}

	// Drop the first transport before the receiver sends the ACK.
	closeFirst()
	registry.Detach(id, first)
	_, next, nextClientConn, nextServerConn, _ := open(&id)
	nextTransfer, err := next.Transfer(1024)
	if err != nil {
		t.Fatal(err)
	}
	nextClient := NewFramedPath(nextClientConn, clientTransfer)
	nextServer := NewFramedPath(nextServerConn, nextTransfer)
	go func() { sent <- nextClient.Replay() }()
	ack, err := nextServer.Receive(&received)
	if err != nil || ack == nil || ack.Offset != 7 || received.String() != "payload" {
		t.Fatalf("replay: ACK=%v data=%q err=%v", ack, &received, err)
	}
	if err := <-sent; err != nil {
		t.Fatal(err)
	}
	go func() { sent <- nextServer.SendACK(ack.Offset) }()
	if response, err := nextClient.Receive(nil); err != nil || response != nil {
		t.Fatalf("ACK handling: response=%v err=%v", response, err)
	}
	if err := <-sent; err != nil {
		t.Fatal(err)
	}
	if len(clientTransfer.send.pending) != 0 {
		t.Fatal("ACK did not release the send window")
	}
	conn, err := next.Backend()
	if err != nil || conn != backend || dials != 1 {
		t.Fatalf("resumed backend=%v dials=%d err=%v", conn == backend, dials, err)
	}
	if err := registry.End(id, next); err != nil {
		t.Fatal(err)
	}
}
