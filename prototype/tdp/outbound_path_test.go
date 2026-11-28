package tdp

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestForwardNextReplaysLastDataAndFINAfterWriteFailure(t *testing.T) {
	transfer, err := NewTransfer(32)
	if err != nil {
		t.Fatal(err)
	}
	source := &finalChunkReader{data: "reply"}
	failed := NewFramedPath(&fallbackWire{failWrites: true}, transfer)
	finished, err := failed.ForwardNext(source, make([]byte, 32))
	if !finished || !errors.Is(err, ErrPathIO) {
		t.Fatalf("finished=%v err=%v", finished, err)
	}

	var wire bytes.Buffer
	resumed := NewFramedPath(&wire, transfer)
	if err := resumed.Replay(); err != nil {
		t.Fatal(err)
	}
	data, err := ReadFrame(&wire)
	if err != nil || data.Kind != DataFrame || data.Offset != 0 || string(data.Payload) != "reply" {
		t.Fatalf("DATA=%+v err=%v", data, err)
	}
	fin, err := ReadFrame(&wire)
	if err != nil || fin.Kind != FINFrame || fin.Offset != 5 || wire.Len() != 0 {
		t.Fatalf("FIN=%+v remaining=%d err=%v", fin, wire.Len(), err)
	}
	if _, err := resumed.ForwardNext(source, make([]byte, 32)); !errors.Is(err, ErrSendFinished) {
		t.Fatalf("read after FIN: %v", err)
	}
}

func TestForwardNextSendsFINForEmptySource(t *testing.T) {
	transfer, err := NewTransfer(32)
	if err != nil {
		t.Fatal(err)
	}
	var wire bytes.Buffer
	path := NewFramedPath(&wire, transfer)
	finished, err := path.ForwardNext(bytes.NewReader(nil), make([]byte, 32))
	if !finished || err != nil {
		t.Fatalf("finished=%v err=%v", finished, err)
	}
	fin, err := ReadFrame(&wire)
	if err != nil || fin.Kind != FINFrame || fin.Offset != 0 {
		t.Fatalf("FIN=%+v err=%v", fin, err)
	}
	if _, err := ReadFrame(&wire); !errors.Is(err, io.EOF) {
		t.Fatalf("unexpected bytes after FIN: %v", err)
	}
}
