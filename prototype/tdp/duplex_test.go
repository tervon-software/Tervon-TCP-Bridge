package tdp

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"
)

func TestDuplexMovesBothDirectionsBeyondReplayWindow(t *testing.T) {
	deadline := time.Now().Add(5 * time.Second)
	client, left := duplexTCPPair(t, deadline)
	right, backend := duplexTCPPair(t, deadline)
	leftWire, rightWire := duplexTCPPair(t, deadline)

	outbound := bytes.Repeat([]byte("client to server"), 2048)
	inbound := bytes.Repeat([]byte("server to client"), 2048)
	leftTransfer, err := NewTransfer(257)
	if err != nil {
		t.Fatal(err)
	}
	rightTransfer, err := NewTransfer(257)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	results := make(chan error, 4)
	go func() { results <- RunDuplex(ctx, leftWire, left, leftTransfer) }()
	go func() { results <- RunDuplex(ctx, rightWire, right, rightTransfer) }()
	send := func(conn *net.TCPConn, data []byte) {
		_, err := io.Copy(conn, bytes.NewReader(data))
		if err == nil {
			err = conn.CloseWrite()
		}
		results <- err
	}
	go send(client, outbound)
	go send(backend, inbound)
	type readResult struct {
		data []byte
		err  error
	}
	replies := make(chan readResult, 1)
	go func() {
		data, err := io.ReadAll(client)
		replies <- readResult{data, err}
	}()
	request, err := io.ReadAll(backend)
	if err != nil || !bytes.Equal(request, outbound) {
		t.Fatalf("backend bytes=%d err=%v", len(request), err)
	}
	reply := <-replies
	if reply.err != nil || !bytes.Equal(reply.data, inbound) {
		t.Fatalf("client bytes=%d err=%v", len(reply.data), reply.err)
	}
	for range 4 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
}

func duplexTCPPair(t *testing.T, deadline time.Time) (*net.TCPConn, *net.TCPConn) {
	t.Helper()
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := listener.SetDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	client, err := net.DialTCP("tcp4", nil, listener.Addr().(*net.TCPAddr))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	server, err := listener.AcceptTCP()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	for _, conn := range []*net.TCPConn{client, server} {
		if err := conn.SetDeadline(deadline); err != nil {
			t.Fatal(err)
		}
	}
	return client, server
}
