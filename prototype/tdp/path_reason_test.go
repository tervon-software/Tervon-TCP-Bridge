package tdp

import (
	"errors"
	"io"
	"testing"
)

func TestFrameFailureReasons(t *testing.T) {
	transfer, err := NewTransfer(1024)
	if err != nil {
		t.Fatal(err)
	}
	wire := &fallbackWire{failWrites: true}
	path := NewFramedPath(wire, transfer)
	if err := path.Send([]byte("data")); !errors.Is(err, ErrPathIO) || PathFailureReason(err) != PathCloseTransportWrite {
		t.Fatalf("transport write: %v", err)
	}
	if _, err := path.Receive(io.Discard); !errors.Is(err, io.EOF) || PathFailureReason(err) != PathCloseTransportRead {
		t.Fatalf("transport read: %v", err)
	}
	if err := WriteFrame(&wire.Buffer, Frame{Kind: DataFrame, Payload: []byte("data")}); err != nil {
		t.Fatal(err)
	}
	if _, err := path.Receive(&fallbackWire{failWrites: true}); !errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, ErrPathIO) || PathFailureReason(err) != PathCloseBackendWrite {
		t.Fatalf("backend write: %v", err)
	}
}
