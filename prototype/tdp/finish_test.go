package tdp

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type halfCloseSink struct {
	bytes.Buffer
	closes int
}

func (s *halfCloseSink) CloseWrite() error {
	s.closes++
	return nil
}

func TestFINReplayPreservesReverseDirection(t *testing.T) {
	sender, err := NewTransfer(16)
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := NewTransfer(16)
	if err != nil {
		t.Fatal(err)
	}
	data, err := sender.Queue([]byte("last"))
	if err != nil {
		t.Fatal(err)
	}
	fin := sender.FinishSend()
	if _, err := sender.Queue([]byte("extra")); !errors.Is(err, ErrSendFinished) {
		t.Fatalf("queued data after FIN: %v", err)
	}

	var sink halfCloseSink
	if _, err := receiver.Handle(&sink, fin); !errors.Is(err, ErrOffset) || sink.closes != 0 {
		t.Fatalf("premature FIN: closes=%d err=%v", sink.closes, err)
	}
	if _, err := receiver.Handle(&sink, data); err != nil {
		t.Fatal(err)
	}
	if _, err := receiver.Handle(&sink, fin); err != nil {
		t.Fatal(err)
	}

	// The FIN_ACK is lost. The next path replays unacknowledged DATA and FIN.
	var wire bytes.Buffer
	path := NewFramedPath(&wire, sender)
	if err := path.Replay(); err != nil {
		t.Fatal(err)
	}
	replayedData, err := ReadFrame(&wire)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := receiver.Handle(&sink, replayedData); err != nil {
		t.Fatal(err)
	}
	replayedFIN, err := ReadFrame(&wire)
	if err != nil || replayedFIN.Kind != FINFrame || replayedFIN.Offset != 4 {
		t.Fatalf("replayed FIN: %+v err=%v", replayedFIN, err)
	}
	ack, err := receiver.Handle(&sink, replayedFIN)
	if err != nil || ack == nil || ack.Kind != FINAckFrame ||
		sink.closes != 1 || sink.String() != "last" {
		t.Fatalf("FIN_ACK=%v closes=%d data=%q err=%v", ack, sink.closes, &sink, err)
	}
	if _, err := sender.Handle(io.Discard, *ack); err != nil {
		t.Fatal(err)
	}
	if err := path.Replay(); err != nil || wire.Len() != 0 {
		t.Fatalf("confirmed FIN replayed: bytes=%d err=%v", wire.Len(), err)
	}
	reverse, err := receiver.Queue([]byte("reply"))
	if err != nil {
		t.Fatal(err)
	}
	var reply bytes.Buffer
	if _, err := sender.Handle(&reply, reverse); err != nil || reply.String() != "reply" {
		t.Fatalf("reverse data=%q err=%v", &reply, err)
	}
}
