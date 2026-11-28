package tdp

import (
	"bytes"
	"errors"
	"testing"
)

func TestReplayResume(t *testing.T) {
	replay, err := NewReplay(8)
	if err != nil {
		t.Fatal(err)
	}
	original := []byte("abcdefgh")
	offset, err := replay.Append(original)
	if offset != 0 || err != nil {
		t.Fatalf("append: %d, %v", offset, err)
	}
	if _, err := replay.Append([]byte("i")); !errors.Is(err, ErrWindowFull) {
		t.Fatalf("full window: %v", err)
	}
	for _, ack := range []uint64{3, 2, 3} {
		if err := replay.Acknowledge(ack); err != nil {
			t.Fatal(err)
		}
	}
	if err := replay.Acknowledge(9); !errors.Is(err, ErrOffset) {
		t.Fatalf("future ACK: %v", err)
	}
	if _, err := replay.From(2); !errors.Is(err, ErrOffset) {
		t.Fatalf("expired resume: %v", err)
	}
	tail, err := replay.From(5)
	if err != nil || !bytes.Equal(append(bytes.Clone(original[:5]), tail...), original) {
		t.Fatalf("resume: %q, %v", tail, err)
	}
	tail[0] = 'X'
	tail, err = replay.From(5)
	if err != nil || string(tail) != "fgh" {
		t.Fatalf("replay storage changed: %q, %v", tail, err)
	}
	offset, err = replay.Append([]byte("ijk"))
	if offset != 8 || err != nil {
		t.Fatalf("append after ACK: %d, %v", offset, err)
	}
}
