package tdp

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"testing"
	"time"
)

func TestAuthenticatedTCPDuplex(t *testing.T) {
	deadline := time.Now().Add(5 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	app, local := duplexTCPPair(t, deadline)
	backend, remote := duplexTCPPair(t, deadline)
	rawClient, rawServer := duplexTCPPair(t, deadline)
	serverTLS, clientTLS, fingerprint := testPathTLS(t)
	clientWire := tls.Client(rawClient, clientTLS)
	serverWire := tls.Server(rawServer, serverTLS)
	registry, err := NewRegistry(1, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	server := TCPServer{
		Registry:    registry,
		Authorize:   func(peer [32]byte) bool { return peer == fingerprint },
		DialBackend: func(context.Context) (net.Conn, error) { return backend, nil },
	}
	serverDone := make(chan error, 1)
	go func() {
		id, path, err := server.AcceptPath(ctx, serverWire)
		if err == nil {
			err = registry.ServeDuplexPath(ctx, id, path, serverWire, 257, time.Second)
		}
		serverDone <- err
	}()
	if _, err := OpenTCPPath(ctx, clientWire, nil); err != nil {
		t.Fatal(err)
	}
	transfer, err := NewTransfer(257)
	if err != nil {
		t.Fatal(err)
	}
	clientDone := make(chan error, 1)
	go func() { clientDone <- RunDuplex(ctx, clientWire, local, transfer) }()

	request := bytes.Repeat([]byte("request"), 4096)
	reply := bytes.Repeat([]byte("reply"), 4096)
	exchange := func(conn *net.TCPConn, send, want []byte) error {
		if _, err := io.Copy(conn, bytes.NewReader(send)); err != nil {
			return err
		}
		if err := conn.CloseWrite(); err != nil {
			return err
		}
		got, err := io.ReadAll(conn)
		if err == nil && !bytes.Equal(got, want) {
			return io.ErrUnexpectedEOF
		}
		return err
	}
	appDone := make(chan error, 1)
	go func() { appDone <- exchange(app, request, reply) }()
	if err := exchange(remote, reply, request); err != nil {
		t.Fatalf("backend exchange: %v", err)
	}
	for _, done := range []<-chan error{appDone, clientDone, serverDone} {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	stats := registry.PathStatsSnapshot().TCP
	if stats.Opened != 1 || stats.Closed != 1 || stats.ClosedByReason[PathCloseNormal] != 1 {
		t.Fatalf("unexpected path stats: %+v", stats)
	}
}
