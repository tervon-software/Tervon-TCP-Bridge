package tdp

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type finalChunkReader struct {
	data string
}

func (r *finalChunkReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	if r.data == "" {
		return n, io.EOF
	}
	return n, nil
}

func TestReadOutboundBackpressureAndFinalBytes(t *testing.T) {
	transfer, err := NewTransfer(4)
	if err != nil {
		t.Fatal(err)
	}
	scratch := make([]byte, 16)
	source := &finalChunkReader{data: "abcdef"}

	first, err := transfer.ReadOutbound(source, scratch)
	if err != nil || first.Kind != DataFrame || first.Offset != 0 ||
		string(first.Payload) != "abcd" || source.data != "ef" {
		t.Fatalf("first frame=%+v remaining=%q err=%v", first, source.data, err)
	}
	if _, err := transfer.ReadOutbound(source, scratch); !errors.Is(err, ErrWindowFull) ||
		source.data != "ef" {
		t.Fatalf("full window consumed source: remaining=%q err=%v", source.data, err)
	}
	if _, err := transfer.Handle(nil, Frame{Kind: AckFrame, Offset: 2}); err != nil {
		t.Fatal(err)
	}

	last, err := transfer.ReadOutbound(source, scratch)
	if !errors.Is(err, io.EOF) || last.Kind != DataFrame ||
		last.Offset != 4 || string(last.Payload) != "ef" {
		t.Fatalf("final frame=%+v err=%v", last, err)
	}
	scratch[0] = 'X'
	replay, err := transfer.Pending(2)
	if err != nil || replay.Offset != 2 || string(replay.Payload) != "cdef" {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
}

func TestReadOutboundLimitsFrameSize(t *testing.T) {
	transfer, err := NewTransfer(2 * MaxFramePayload)
	if err != nil {
		t.Fatal(err)
	}
	source := strings.NewReader(strings.Repeat("x", 2*MaxFramePayload))
	frame, err := transfer.ReadOutbound(source, make([]byte, 2*MaxFramePayload))
	if err != nil || len(frame.Payload) != MaxFramePayload || source.Len() != MaxFramePayload {
		t.Fatalf("size=%d remaining=%d err=%v", len(frame.Payload), source.Len(), err)
	}
}
