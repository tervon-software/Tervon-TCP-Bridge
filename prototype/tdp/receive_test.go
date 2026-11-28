package tdp

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type limitedWriter struct {
	dst *bytes.Buffer
}

func (w limitedWriter) Write(p []byte) (int, error) {
	return w.dst.Write(p[:2])
}

func TestReceiverPartialWriteAndReplay(t *testing.T) {
	var dst bytes.Buffer
	var receiver Receiver
	payload := []byte("abcdef")
	ack, err := receiver.Deliver(limitedWriter{&dst}, 0, payload)
	if ack != 2 || !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("partial write: ACK=%d err=%v", ack, err)
	}
	ack, err = receiver.Deliver(&dst, 3, payload[3:])
	if ack != 2 || !errors.Is(err, ErrOffset) || dst.String() != "ab" {
		t.Fatalf("gap: ACK=%d err=%v output=%q", ack, err, &dst)
	}
	for _, offset := range []uint64{0, 2} {
		ack, err = receiver.Deliver(&dst, offset, payload[offset:])
		if ack != 6 || err != nil || !bytes.Equal(dst.Bytes(), payload) {
			t.Fatalf("replay at %d: ACK=%d err=%v output=%q", offset, ack, err, &dst)
		}
	}
}
