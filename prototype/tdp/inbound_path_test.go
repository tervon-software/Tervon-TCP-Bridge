package tdp

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestInboundPathDeliversAndReleases(t *testing.T) {
	registry, err := NewRegistry(1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	backend, remote := net.Pipe()
	defer backend.Close()
	defer remote.Close()
	owner := [32]byte{1}
	id, first, err := registry.Open(owner, backend)
	if err != nil {
		t.Fatal(err)
	}
	registry.observeOpen(id, first, TransportTCP)

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	for _, conn := range []net.Conn{client, server, backend, remote} {
		if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatal(err)
		}
	}

	done := make(chan error, 1)
	go func() {
		done <- registry.ServeInboundPath(id, first, server, 1024)
	}()

	payload := []byte("through the backend")
	received := make([]byte, len(payload))
	backendRead := make(chan error, 1)
	go func() {
		_, err := io.ReadFull(remote, received)
		backendRead <- err
	}()
	if err := WriteFrame(client, Frame{Kind: DataFrame, Payload: payload}); err != nil {
		t.Fatal(err)
	}
	ack, err := ReadFrame(client)
	if err != nil || ack.Kind != AckFrame || ack.Offset != uint64(len(payload)) {
		t.Fatalf("ACK=%+v err=%v", ack, err)
	}
	if err := <-backendRead; err != nil || !bytes.Equal(received, payload) {
		t.Fatalf("backend data=%q err=%v", received, err)
	}
	if err := RequestRelease(client); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	resumed, err := registry.Resume(id, owner)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := resumed.Backend()
	if err != nil || conn != backend {
		t.Fatalf("same backend=%v err=%v", conn == backend, err)
	}
	if got := registry.PathStatsSnapshot().TCP.ClosedByReason[PathCloseHandoff]; got != 1 {
		t.Fatalf("handoff close count=%d", got)
	}
	if err := registry.End(id, resumed); err != nil {
		t.Fatal(err)
	}
	if _, err := remote.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("backend after End: %v", err)
	}
}
