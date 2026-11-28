package tdp

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

func TestTCPBackendRepliesAfterTDPFIN(t *testing.T) {
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
	defer backend.Close()
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

	registry, err := NewRegistry(1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	owner := [32]byte{1}
	id, attachment, err := registry.Open(owner, backend)
	if err != nil {
		t.Fatal(err)
	}
	server, err := attachment.Transfer(64)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewTransfer(64)
	if err != nil {
		t.Fatal(err)
	}

	var wire bytes.Buffer
	clientPath := NewFramedPath(&wire, client)
	if err := clientPath.Send([]byte("request")); err != nil {
		t.Fatal(err)
	}
	if err := clientPath.SendFIN(); err != nil {
		t.Fatal(err)
	}
	serverPath := NewFramedPath(&wire, server)
	for _, want := range []FrameKind{AckFrame, FINAckFrame} {
		ack, err := serverPath.Receive(backend)
		if err != nil || ack == nil || ack.Kind != want || ack.Offset != 7 {
			t.Fatalf("request acknowledgment: %+v err=%v", ack, err)
		}
		if _, err := client.Handle(nil, *ack); err != nil {
			t.Fatal(err)
		}
	}
	request, err := io.ReadAll(remote)
	if err != nil || string(request) != "request" {
		t.Fatalf("backend request=%q err=%v", request, err)
	}
	if _, err := io.WriteString(remote, "reply after EOF"); err != nil {
		t.Fatal(err)
	}
	if err := remote.CloseWrite(); err != nil {
		t.Fatal(err)
	}

	scratch := make([]byte, 4)
	var reply halfCloseSink
	for finished := false; !finished; {
		finished, err = serverPath.ForwardNext(backend, scratch)
		if err != nil {
			t.Fatal(err)
		}
		for wire.Len() > 0 {
			ack, err := clientPath.Receive(&reply)
			if err != nil || ack == nil {
				t.Fatalf("response acknowledgment: %+v err=%v", ack, err)
			}
			if _, err := server.Handle(nil, *ack); err != nil {
				t.Fatal(err)
			}
		}
	}
	if reply.String() != "reply after EOF" || reply.closes != 1 ||
		!client.finAcknowledged || !server.finAcknowledged {
		t.Fatalf("reply=%q closewrites=%d clientFIN=%v serverFIN=%v",
			&reply, reply.closes, client.finAcknowledged, server.finAcknowledged)
	}
	if err := registry.End(id, attachment); err != nil {
		t.Fatal(err)
	}
}
