package tdp

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/quic-go/quic-go"
)

func TestDuplexReturnsToQUICAfterUDPLoss(t *testing.T) {
	deadline := time.Now().Add(10 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	app, local := duplexTCPPair(t, deadline)
	backend, remote := duplexTCPPair(t, deadline)
	serverTLS, clientTLS, fingerprint := testPathTLS(t)
	registry, err := NewRegistry(1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	const window = 257
	transfer, err := NewTransfer(window)
	if err != nil {
		t.Fatal(err)
	}
	var dials atomic.Int32
	server := PathServer{
		Registry:  registry,
		Authorize: func(peer [32]byte) bool { return peer == fingerprint },
		DialBackend: func(context.Context) (net.Conn, error) {
			dials.Add(1)
			return backend, nil
		},
	}

	socket, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer socket.Close()
	link := &testUDPLink{PacketConn: socket}
	listener, err := quic.Listen(link, serverTLS, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	firstServer := make(chan error, 1)
	blackhole := make(chan struct{})
	go func() {
		conn, err := listener.Accept(ctx)
		if err == nil {
			defer conn.CloseWithError(0, "")
			var stream *quic.Stream
			var id SessionID
			var path *Attachment
			stream, id, path, err = server.AcceptQUICPath(ctx, conn)
			if err == nil {
				go func() {
					select {
					case <-blackhole:
						_ = stream.SetDeadline(time.Now().Add(200 * time.Millisecond))
					case <-ctx.Done():
					}
				}()
				err = registry.ServeDuplexPath(ctx, id, path, stream, window, time.Second)
			}
		}
		firstServer <- err
	}()
	quicConn, err := quic.DialAddr(ctx, listener.Addr().String(), clientTLS, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer quicConn.CloseWithError(0, "")
	stream, id, err := OpenQUICPath(ctx, quicConn, nil)
	if err != nil {
		t.Fatal(err)
	}
	firstClient := make(chan error, 1)
	go func() { firstClient <- RunDuplex(ctx, stream, local, transfer) }()

	exchange := func(prefix string) {
		t.Helper()
		for _, direction := range []struct {
			from, to net.Conn
			payload  []byte
		}{
			{app, remote, bytes.Repeat([]byte(prefix+" upload"), 250)},
			{remote, app, bytes.Repeat([]byte(prefix+" download"), 250)},
		} {
			if _, err := io.Copy(direction.from, bytes.NewReader(direction.payload)); err != nil {
				t.Fatal(err)
			}
			got := make([]byte, len(direction.payload))
			if _, err := io.ReadFull(direction.to, got); err != nil || !bytes.Equal(got, direction.payload) {
				t.Fatalf("%s: data mismatch, err=%v", prefix, err)
			}
		}
	}
	exchange("QUIC")
	link.drop.Store(true)
	close(blackhole)
	if err := stream.SetDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	for _, done := range []<-chan error{firstClient, firstServer} {
		if err := <-done; !errors.Is(err, ErrPathIO) {
			t.Fatalf("UDP loss: %v", err)
		}
	}

	rawClient, rawServer := duplexTCPPair(t, deadline)
	clientWire, serverWire := tls.Client(rawClient, clientTLS), tls.Server(rawServer, serverTLS)
	secondServer := make(chan error, 1)
	go func() {
		resumed, path, err := TCPServer(server).AcceptPath(ctx, serverWire)
		if err == nil {
			err = registry.ServeDuplexPath(ctx, resumed, path, serverWire, window, time.Second)
		}
		secondServer <- err
	}()
	if resumed, err := OpenTCPPath(ctx, clientWire, &id); err != nil || resumed != id {
		t.Fatalf("TCP resume: id=%x err=%v", resumed, err)
	}
	secondClient := make(chan error, 1)
	releaseTCP := make(chan struct{})
	go func() { secondClient <- RunDuplexWithRelease(ctx, clientWire, local, transfer, releaseTCP) }()
	exchange("TCP")
	link.drop.Store(false)
	finalServer := make(chan error, 1)
	go func() {
		conn, err := listener.Accept(ctx)
		if err == nil {
			t.Cleanup(func() { _ = conn.CloseWithError(0, "") })
			stream, resumed, path, acceptErr := server.AcceptQUICPath(ctx, conn)
			err = acceptErr
			if err == nil {
				err = registry.ServeDuplexPath(ctx, resumed, path, stream, window, time.Second)
			}
		}
		finalServer <- err
	}()
	recovered, err := quic.DialAddr(ctx, listener.Addr().String(), clientTLS, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.CloseWithError(0, "")
	if err := ProbeQUICPath(ctx, recovered); err != nil {
		t.Fatal(err)
	}
	exchange("TCP after QUIC probe")

	close(releaseTCP)
	if err := <-secondClient; !errors.Is(err, ErrPathReleased) {
		t.Fatalf("release TCP duplex: %v", err)
	}
	if err := <-secondServer; err != nil {
		t.Fatal(err)
	}
	finalWire, resumed, err := OpenQUICPath(ctx, recovered, &id)
	if err != nil || resumed != id {
		t.Fatalf("QUIC resume: id=%x err=%v", resumed, err)
	}
	finalClient := make(chan error, 1)
	go func() { finalClient <- RunDuplex(ctx, finalWire, local, transfer) }()
	exchange("QUIC after TCP")

	for _, conn := range []*net.TCPConn{app, remote} {
		if err := conn.CloseWrite(); err != nil {
			t.Fatal(err)
		}
	}
	for _, done := range []<-chan error{finalClient, finalServer} {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	for _, conn := range []*net.TCPConn{app, remote} {
		if _, err := conn.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
			t.Fatalf("expected EOF without trailing bytes: %v", err)
		}
	}
	if stats := registry.PathStatsSnapshot(); dials.Load() != 1 || stats.QUICToTCP != 1 || stats.TCPToQUIC != 1 ||
		stats.TCP.ClosedByReason[PathCloseHandoff] != 1 {
		t.Fatalf("dials=%d stats=%+v", dials.Load(), stats)
	}
}
