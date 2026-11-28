package tdp

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestRegistryResumeAndExpiry(t *testing.T) {
	grace := time.Second
	registry, err := NewRegistry(1, grace)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	registry.now = func() time.Time { return now }

	owner, other := [32]byte{1}, [32]byte{2}
	backend, remote := net.Pipe()
	defer backend.Close()
	defer remote.Close()
	if err := remote.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	id, first, err := registry.Open(owner, backend)
	if err != nil {
		t.Fatal(err)
	}
	if expired := registry.Reap(); expired != 0 {
		t.Fatalf("reaped attached session: %d", expired)
	}

	extra, extraRemote := net.Pipe()
	defer extra.Close()
	defer extraRemote.Close()
	if _, _, err := registry.Open(owner, extra); !errors.Is(err, ErrRegistryFull) {
		t.Fatalf("session limit: %v", err)
	}

	registry.Detach(id, first)
	now = now.Add(grace / 2)
	if _, err := registry.Resume(id, other); !errors.Is(err, ErrSessionMissing) {
		t.Fatalf("foreign resume: %v", err)
	}
	second, err := registry.Resume(id, owner)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.End(id, first); !errors.Is(err, ErrSessionMissing) {
		t.Fatalf("stale path ended resumed session: %v", err)
	}
	registry.Detach(id, first)
	now = now.Add(grace)
	if expired := registry.Reap(); expired != 0 {
		t.Fatalf("reaped resumed session: %d", expired)
	}
	if conn, err := second.Backend(); err != nil || conn != backend {
		t.Fatalf("backend after resume: %v, %v", conn, err)
	}

	registry.Detach(id, second)
	now = now.Add(grace)
	if expired := registry.Reap(); expired != 1 {
		t.Fatalf("expected one expired session, got %d", expired)
	}
	if _, err := remote.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("expired backend still open: %v", err)
	}
	if _, err := registry.Resume(id, owner); !errors.Is(err, ErrSessionMissing) {
		t.Fatalf("resume after expiry: %v", err)
	}

	nextID, next, err := registry.Open(owner, extra)
	if err != nil {
		t.Fatal(err)
	}
	registry.Detach(nextID, next)
	now = now.Add(grace)
	if _, err := registry.Resume(nextID, owner); !errors.Is(err, ErrSessionMissing) {
		t.Fatalf("resume at deadline without Reap: %v", err)
	}
}
