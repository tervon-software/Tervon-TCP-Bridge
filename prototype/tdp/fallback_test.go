package tdp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

type fallbackWire struct {
	bytes.Buffer
	closed     bool
	failWrites bool
}

func (w *fallbackWire) Close() error {
	w.closed = true
	return nil
}

func (w *fallbackWire) Write(data []byte) (int, error) {
	if w.failWrites {
		return 0, io.ErrClosedPipe
	}
	return w.Buffer.Write(data)
}

func TestTCPFallbackReplaysPendingBytes(t *testing.T) {
	transfer, err := NewTransfer(1024)
	if err != nil {
		t.Fatal(err)
	}
	id := SessionID{42}
	quic := &fallbackWire{failWrites: true}
	tcp := &fallbackWire{}
	attempts := 0
	err = RunWithTCPFallback(context.Background(), id, transfer, quic,
		func() { _ = quic.Close() },
		func(_ context.Context, resumed SessionID) (io.ReadWriteCloser, error) {
			if resumed != id || !quic.closed {
				t.Fatalf("resume id=%x previous path closed=%v", resumed, quic.closed)
			}
			return tcp, nil
		},
		func(_ context.Context, path *FramedPath) error {
			attempts++
			if attempts == 1 {
				return path.Send([]byte("payload"))
			}
			frame, err := ReadFrame(tcp)
			if err != nil || frame.Kind != DataFrame || frame.Offset != 0 || string(frame.Payload) != "payload" {
				t.Fatalf("replayed frame=%+v err=%v", frame, err)
			}
			return nil
		},
	)
	if err != nil || attempts != 2 || !tcp.closed {
		t.Fatalf("fallback: attempts=%d closed=%v err=%v", attempts, tcp.closed, err)
	}
}

func TestTCPFallbackSkipsNonTransportError(t *testing.T) {
	transfer, err := NewTransfer(1024)
	if err != nil {
		t.Fatal(err)
	}
	wire := &fallbackWire{}
	err = RunWithTCPFallback(context.Background(), SessionID{}, transfer, wire,
		func() { _ = wire.Close() },
		func(context.Context, SessionID) (io.ReadWriteCloser, error) {
			t.Fatal("unexpected fallback")
			return nil, nil
		},
		func(_ context.Context, path *FramedPath) error {
			return path.Send(nil)
		},
	)
	if !errors.Is(err, ErrFrame) || !wire.closed {
		t.Fatalf("protocol error: closed=%v err=%v", wire.closed, err)
	}
}
