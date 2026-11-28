package tdp

import (
	"errors"
	"net"
	"testing"
	"time"
)

func TestReleaseAllowsSameBackendToResume(t *testing.T) {
	registry, err := NewRegistry(1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	backend, remote := net.Pipe()
	defer backend.Close()
	defer remote.Close()
	owner := [32]byte{42}
	id, first, err := registry.Open(owner, backend)
	if err != nil {
		t.Fatal(err)
	}
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	for _, conn := range []net.Conn{client, server} {
		if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatal(err)
		}
	}

	done := make(chan error, 1)
	go func() {
		request, err := ReadFrame(server)
		if err == nil {
			err = registry.ConfirmRelease(id, first, server, request)
		}
		done <- err
	}()
	if err := RequestRelease(client); err != nil {
		t.Fatal(err)
	}
	next, err := registry.Resume(id, owner)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := registry.ConfirmRelease(id, first, server, Frame{Kind: ReleaseFrame}); !errors.Is(err, ErrSessionMissing) {
		t.Fatalf("stale release: %v", err)
	}
	if conn, err := next.Backend(); err != nil || conn != backend {
		t.Fatalf("resumed backend identity: %v, %v", conn == backend, err)
	}
	if err := registry.End(id, next); err != nil {
		t.Fatal(err)
	}
}
