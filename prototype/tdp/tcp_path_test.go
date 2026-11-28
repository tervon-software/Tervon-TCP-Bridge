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
	"errors"
	"io"
	"math/big"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/quic-go/quic-go"
)

func TestTCPPathResumesBackendOverMTLS(t *testing.T) {
	t.Run("TCP to TCP", func(t *testing.T) { testPathResumesBackend(t, false) })
	t.Run("QUIC to TCP", func(t *testing.T) { testPathResumesBackend(t, true) })
}

func testPathResumesBackend(t *testing.T, startQUIC bool) {
	t.Helper()
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
	events := make(chan PathEvent, 16)
	registry, err := NewObservedRegistry(1, time.Minute, func(event PathEvent) {
		events <- event
	})
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
		listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		if err := listener.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatal(err)
		}
		clientSide, err := (&net.Dialer{}).DialContext(ctx, "tcp4", listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = clientSide.Close() })
		serverSide, err := listener.AcceptTCP()
		if err != nil {
			t.Fatal(err)
		}
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
	var id SessionID
	var first *Attachment
	var clientConn, serverConn io.ReadWriter
	var closeFirst, dropUDP func()
	var prepareRecovered func(context.Context) (*quic.Conn, func() (*quic.Stream, *Attachment, error), error)
	if startQUIC {
		id, first, clientConn, serverConn, closeFirst, dropUDP, prepareRecovered = openLoopbackQUIC(t, PathServer(server), serverTLS, clientTLS)
	} else {
		id, first, clientConn, serverConn, closeFirst = open(nil)
	}
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

	// The sender has data in its window, but the first ACK cannot arrive.
	if startQUIC {
		dropUDP()
		if err := serverPath.SendACK(7); err != nil {
			t.Fatal(err)
		}
		stream := clientConn.(*quic.Stream)
		if err := stream.SetReadDeadline(time.Now().Add(150 * time.Millisecond)); err != nil {
			t.Fatal(err)
		}
	} else {
		closeFirst()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var next *Attachment
	var nextServer *FramedPath
	serverDone := make(chan error, 1)
	err = RunWithTCPFallback(ctx, id, clientTransfer, clientConn,
		func() {
			closeFirst()
			registry.Detach(id, first)
		},
		func(_ context.Context, resumeID SessionID) (io.ReadWriteCloser, error) {
			if resumeID != id {
				t.Fatalf("resume changed session ID: %x", resumeID)
			}
			_, attachment, nextClientConn, nextServerConn, _ := open(&resumeID)
			next = attachment
			nextTransfer, err := next.Transfer(1024)
			if err != nil {
				return nil, err
			}
			nextServer = NewFramedPath(nextServerConn, nextTransfer)
			go func() {
				ack, err := nextServer.Receive(&received)
				if err != nil {
					serverDone <- err
					return
				}
				if ack == nil || ack.Offset != 7 {
					serverDone <- ErrOffset
					return
				}
				serverDone <- nextServer.SendACK(ack.Offset)
			}()
			return nextClientConn, nil
		},
		func(ctx context.Context, path *FramedPath) error {
			response, err := path.Receive(io.Discard)
			if next == nil && startQUIC {
				var timeout net.Error
				if !errors.As(err, &timeout) || !timeout.Timeout() {
					t.Fatalf("expected ACK timeout during silent UDP loss, got %v", err)
				}
			}
			if response != nil {
				t.Fatalf("unexpected DATA while waiting for ACK: %v", response)
			}
			if err != nil || !startQUIC || next == nil {
				return err
			}

			recovered, acceptRecovered, err := prepareRecovered(ctx)
			if err != nil {
				return err
			}
			if _, err := registry.Resume(id, sha256.Sum256(der)); !errors.Is(err, ErrSessionBusy) {
				t.Fatalf("PROBE changed TCP attachment: %v", err)
			}

			receivedTCP := make(chan error, 1)
			go func() {
				ack, err := nextServer.Receive(&received)
				if err != nil {
					receivedTCP <- err
					return
				}
				if ack == nil || ack.Offset != 16 {
					receivedTCP <- ErrOffset
					return
				}
				receivedTCP <- nextServer.SendACK(ack.Offset)
			}()
			if err := path.Send([]byte("-recovery")); err != nil {
				return err
			}
			response, err = path.Receive(io.Discard)
			if err != nil {
				return err
			}
			if response != nil {
				return ErrFrame
			}
			if err := <-receivedTCP; err != nil {
				return err
			}

			released := make(chan error, 1)
			go func() {
				request, err := ReadFrame(nextServer.wire)
				if err == nil {
					err = registry.ConfirmRelease(id, next, nextServer.wire, request)
				}
				released <- err
			}()
			if err := RequestRelease(path.wire); err != nil {
				return err
			}
			if err := <-released; err != nil {
				return err
			}
			clientStream, resumedID, err := OpenQUICPath(ctx, recovered, &id)
			if err != nil {
				return err
			}
			if resumedID != id {
				return ErrPathProtocol
			}
			serverStream, attachment, err := acceptRecovered()
			if err != nil {
				return err
			}
			next = attachment
			for _, stream := range []*quic.Stream{clientStream, serverStream} {
				if deadline, ok := ctx.Deadline(); ok {
					if err := stream.SetDeadline(deadline); err != nil {
						return err
					}
				}
			}
			recoveredTransfer, err := next.Transfer(1024)
			if err != nil {
				return err
			}
			recoveredClient := NewFramedPath(clientStream, clientTransfer)
			recoveredServer := NewFramedPath(serverStream, recoveredTransfer)
			go func() {
				ack, err := recoveredServer.Receive(&received)
				if err != nil {
					receivedTCP <- err
					return
				}
				if ack == nil || ack.Offset != 21 {
					receivedTCP <- ErrOffset
					return
				}
				receivedTCP <- recoveredServer.SendACK(ack.Offset)
			}()
			if err := recoveredClient.Send([]byte("-quic")); err != nil {
				return err
			}
			if response, err := recoveredClient.Receive(io.Discard); err != nil {
				return err
			} else if response != nil {
				return ErrFrame
			}
			return <-receivedTCP
		},
	)
	if err != nil || next == nil {
		t.Fatalf("fallback: resumed=%v err=%v", next != nil, err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
	want := "payload"
	if startQUIC {
		want += "-recovery-quic"
	}
	if received.String() != want {
		t.Fatalf("duplicate or missing bytes after replay: %q", &received)
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

	wantEvents := []PathEvent{
		{Transport: TransportTCP, Opened: true},
		{Transport: TransportTCP},
		{Transport: TransportTCP, Opened: true},
		{Transport: TransportTCP},
	}
	if startQUIC {
		wantEvents = []PathEvent{
			{Transport: TransportQUIC, Opened: true},
			{Transport: TransportQUIC},
			{Transport: TransportTCP, PreviousTransport: TransportQUIC, Opened: true},
			{Transport: TransportTCP},
			{Transport: TransportQUIC, PreviousTransport: TransportTCP, Opened: true},
			{Transport: TransportQUIC},
		}
	}
	if len(events) != len(wantEvents) {
		t.Fatalf("path event count: got %d, want %d", len(events), len(wantEvents))
	}
	for _, want := range wantEvents {
		got := <-events
		if !got.Opened && got.Lifetime <= 0 {
			t.Fatalf("closed path has no lifetime: %+v", got)
		}
		got.Lifetime = 0
		if got != want {
			t.Fatalf("path event: got %+v, want %+v", got, want)
		}
	}
}

func openLoopbackQUIC(t *testing.T, server PathServer, serverTLS, clientTLS *tls.Config) (SessionID, *Attachment, io.ReadWriter, io.ReadWriter, func(), func(), func(context.Context) (*quic.Conn, func() (*quic.Stream, *Attachment, error), error)) {
	t.Helper()
	socket, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = socket.Close() })
	link := &testUDPLink{PacketConn: socket}
	listener, err := quic.Listen(link, serverTLS, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	type accepted struct {
		conn   *quic.Conn
		stream *quic.Stream
		id     SessionID
		path   *Attachment
		err    error
	}
	result := make(chan accepted, 1)
	go func() {
		conn, err := listener.Accept(ctx)
		if err != nil {
			result <- accepted{err: err}
			return
		}
		stream, id, path, err := server.AcceptQUICPath(ctx, conn)
		if err != nil {
			_ = conn.CloseWithError(1, "")
		}
		result <- accepted{conn, stream, id, path, err}
	}()
	client, err := quic.DialAddr(ctx, listener.Addr().String(), clientTLS, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.CloseWithError(0, "") })
	if err := ProbeQUICPath(ctx, client); err != nil {
		t.Fatalf("QUIC probe: %v", err)
	}
	clientStream, id, err := OpenQUICPath(ctx, client, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := <-result
	if got.err != nil || got.id != id {
		t.Fatalf("QUIC accept: id=%x err=%v", got.id, got.err)
	}
	t.Cleanup(func() { _ = got.conn.CloseWithError(0, "") })
	for _, stream := range []*quic.Stream{clientStream, got.stream} {
		if err := stream.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	closePath := func() {
		_ = client.CloseWithError(0, "")
		_ = got.conn.CloseWithError(0, "")
	}
	prepareRecovered := func(ctx context.Context) (*quic.Conn, func() (*quic.Stream, *Attachment, error), error) {
		link.drop.Store(false)
		result := make(chan accepted, 1)
		go func() {
			conn, err := listener.Accept(ctx)
			if err != nil {
				result <- accepted{err: err}
				return
			}
			t.Cleanup(func() { _ = conn.CloseWithError(0, "") })
			stream, id, attachment, err := server.AcceptQUICPath(ctx, conn)
			result <- accepted{conn, stream, id, attachment, err}
		}()
		conn, err := quic.DialAddr(ctx, listener.Addr().String(), clientTLS, nil)
		if err != nil {
			return nil, nil, err
		}
		t.Cleanup(func() { _ = conn.CloseWithError(0, "") })
		if err := ProbeQUICPath(ctx, conn); err != nil {
			return nil, nil, err
		}
		return conn, func() (*quic.Stream, *Attachment, error) {
			got := <-result
			return got.stream, got.path, got.err
		}, nil
	}
	return id, got.path, clientStream, got.stream, closePath, func() { link.drop.Store(true) }, prepareRecovered
}

// testUDPLink drops traffic in both directions without closing the UDP socket.
type testUDPLink struct {
	net.PacketConn
	drop atomic.Bool
}

func (l *testUDPLink) ReadFrom(p []byte) (int, net.Addr, error) {
	for {
		n, addr, err := l.PacketConn.ReadFrom(p)
		if err != nil || !l.drop.Load() {
			return n, addr, err
		}
	}
}

func (l *testUDPLink) WriteTo(p []byte, addr net.Addr) (int, error) {
	if l.drop.Load() {
		return len(p), nil
	}
	return l.PacketConn.WriteTo(p, addr)
}
