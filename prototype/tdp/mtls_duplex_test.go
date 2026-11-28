package tdp

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"testing"
	"time"

	"github.com/quic-go/quic-go"
)

func TestAuthenticatedDuplex(t *testing.T) {
	t.Run("TCP", func(t *testing.T) { testAuthenticatedDuplex(t, TransportTCP) })
	t.Run("QUIC", func(t *testing.T) { testAuthenticatedDuplex(t, TransportQUIC) })
}

func testAuthenticatedDuplex(t *testing.T, transport Transport) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	app, local := duplexTCPPair(t, deadline)
	backend, remote := duplexTCPPair(t, deadline)
	serverTLS, clientTLS, fingerprint := testPathTLS(t)
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
	var clientWire DuplexWire
	if transport == TransportQUIC {
		listener, err := quic.ListenAddr("127.0.0.1:0", serverTLS, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		go func() {
			conn, err := listener.Accept(ctx)
			if err == nil {
				t.Cleanup(func() { _ = conn.CloseWithError(0, "") })
				var stream *quic.Stream
				var id SessionID
				var path *Attachment
				stream, id, path, err = PathServer(server).AcceptQUICPath(ctx, conn)
				if err == nil {
					err = registry.ServeDuplexPath(ctx, id, path, stream, 257, time.Second)
				}
			}
			serverDone <- err
		}()
		conn, err := quic.DialAddr(ctx, listener.Addr().String(), clientTLS, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.CloseWithError(0, "")
		stream, _, err := OpenQUICPath(ctx, conn, nil)
		if err != nil {
			t.Fatal(err)
		}
		clientWire = stream
	} else {
		rawClient, rawServer := duplexTCPPair(t, deadline)
		clientTLSWire := tls.Client(rawClient, clientTLS)
		serverWire := tls.Server(rawServer, serverTLS)
		go func() {
			id, path, err := server.AcceptPath(ctx, serverWire)
			if err == nil {
				err = registry.ServeDuplexPath(ctx, id, path, serverWire, 257, time.Second)
			}
			serverDone <- err
		}()
		if _, err := OpenTCPPath(ctx, clientTLSWire, nil); err != nil {
			t.Fatal(err)
		}
		clientWire = clientTLSWire
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
	for _, done := range []<-chan error{appDone, clientDone} {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
	snapshot := registry.PathStatsSnapshot()
	stats := snapshot.TCP
	if transport == TransportQUIC {
		stats = snapshot.QUICUDP
	}
	if stats.Opened != 1 || stats.Closed != 1 || stats.ClosedByReason[PathCloseNormal] != 1 {
		t.Fatalf("unexpected path stats: %+v", stats)
	}
}
