package tdp

import (
	"bytes"
	"errors"
	"io"
	"math"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	var wire bytes.Buffer
	frames := []Frame{
		{Kind: DataFrame, Offset: 13, Payload: bytes.Repeat([]byte{0xab}, MaxFramePayload)},
		{Kind: AckFrame, Offset: 13 + MaxFramePayload},
	}
	for _, frame := range frames {
		if err := WriteFrame(&wire, frame); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range frames {
		got, err := ReadFrame(&wire)
		if err != nil || got.Kind != want.Kind || got.Offset != want.Offset || !bytes.Equal(got.Payload, want.Payload) {
			t.Fatalf("frame mismatch: kind=%d offset=%d length=%d err=%v", got.Kind, got.Offset, len(got.Payload), err)
		}
	}
	if _, err := ReadFrame(&wire); !errors.Is(err, io.EOF) {
		t.Fatalf("end of stream: %v", err)
	}
}

func TestReadFrameRejectsMalformedInput(t *testing.T) {
	var wire bytes.Buffer
	if err := WriteFrame(&wire, Frame{Kind: DataFrame, Payload: []byte{42}}); err != nil {
		t.Fatal(err)
	}
	valid := wire.Bytes()
	tests := []struct {
		name   string
		change func([]byte) []byte
		want   error
	}{
		{"version", func(b []byte) []byte { b[0]++; return b }, ErrFrameVersion},
		{"kind", func(b []byte) []byte { b[1] = 255; return b }, ErrFrame},
		{"ack with payload", func(b []byte) []byte { b[1] = byte(AckFrame); return b }, ErrFrame},
		{"oversized", func(b []byte) []byte { b[2] = 255; return b }, ErrFrame},
		{"truncated header", func(b []byte) []byte { return b[:1] }, io.ErrUnexpectedEOF},
		{"truncated payload", func(b []byte) []byte { return b[:frameHeaderSize] }, io.ErrUnexpectedEOF},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ReadFrame(bytes.NewReader(tc.change(bytes.Clone(valid)))); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestWriteFrameRejectsOffsetOverflow(t *testing.T) {
	var wire bytes.Buffer
	err := WriteFrame(&wire, Frame{Kind: DataFrame, Offset: math.MaxUint64, Payload: []byte{1}})
	if !errors.Is(err, ErrFrame) || wire.Len() != 0 {
		t.Fatalf("invalid frame: err=%v written=%d", err, wire.Len())
	}
}
