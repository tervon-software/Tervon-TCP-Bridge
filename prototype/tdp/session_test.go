package tdp

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestSessionReattachesSameBackend(t *testing.T) {
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	deadline := time.Now().Add(5 * time.Second)
	if err := listener.SetDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	backend, err := net.DialTimeout("tcp4", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	owner, other := [32]byte{1}, [32]byte{2}
	session := NewSession(owner, backend)
	defer session.Close()
	remote, err := listener.AcceptTCP()
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Close()
	for _, conn := range []net.Conn{backend, remote} {
		if err := conn.SetDeadline(deadline); err != nil {
			t.Fatal(err)
		}
	}

	first, err := session.Attach(owner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.Attach(owner); !errors.Is(err, ErrSessionBusy) {
		t.Fatalf("parallel attachment: %v", err)
	}
	send := func(path *Attachment, want string) {
		t.Helper()
		conn, err := path.Backend()
		if err != nil || conn != backend {
			t.Fatalf("backend identity: %v, %v", conn, err)
		}
		if _, err := io.WriteString(conn, want); err != nil {
			t.Fatal(err)
		}
		payload := make([]byte, len(want))
		if _, err := io.ReadFull(remote, payload); err != nil || string(payload) != want {
			t.Fatalf("received %q, err=%v", payload, err)
		}
	}
	send(first, "before")
	first.Detach()

	if _, err := session.Attach(other); !errors.Is(err, ErrSessionOwner) {
		t.Fatalf("wrong peer: %v", err)
	}
	second, err := session.Attach(owner)
	if err != nil {
		t.Fatal(err)
	}
	first.Detach()
	if _, err := first.Backend(); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("stale attachment: %v", err)
	}
	send(second, "after")
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := remote.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("backend still open: %v", err)
	}
	if _, err := session.Attach(owner); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("attach after close: %v", err)
	}
}
