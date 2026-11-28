package tdp

import (
	"bytes"
	"errors"
	"net"
	"testing"
)

func TestReattachedPathReplaysUnacknowledgedFrame(t *testing.T) {
	backend, remote := net.Pipe()
	defer remote.Close()
	owner := [32]byte{1}
	session := NewSession(owner, backend)
	defer session.Close()
	path, err := session.Attach(owner)
	if err != nil {
		t.Fatal(err)
	}
	first, err := path.Transfer(1024)
	if err != nil {
		t.Fatal(err)
	}
	peer, err := NewTransfer(1024)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := first.Queue([]byte("payload"))
	if err != nil {
		t.Fatal(err)
	}
	var dst bytes.Buffer
	if _, err := peer.Handle(&dst, frame); err != nil {
		t.Fatal(err)
	}

	// The path is lost before the ACK reaches the sender.
	path.Detach()
	if _, err := path.Transfer(1024); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("stale attachment: %v", err)
	}
	resumed, err := session.Attach(owner)
	if err != nil {
		t.Fatal(err)
	}
	second, err := resumed.Transfer(1024)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := second.Pending(frame.Offset)
	if err != nil {
		t.Fatal(err)
	}
	ack, err := peer.Handle(&dst, retry)
	if err != nil || dst.String() != "payload" || ack.Offset != 7 {
		t.Fatalf("replay: data=%q ack=%v err=%v", &dst, ack, err)
	}
	if _, err := second.Handle(nil, *ack); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Pending(frame.Offset); !errors.Is(err, ErrOffset) {
		t.Fatalf("acknowledged frame retained: %v", err)
	}
}
