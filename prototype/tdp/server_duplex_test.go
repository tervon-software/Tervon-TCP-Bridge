package tdp

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestServerDuplexPathKeepsBackendAcrossFailureAndRelease(t *testing.T) {
	deadline := time.Now().Add(5 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	backend, _ := duplexTCPPair(t, deadline)
	registry, err := NewRegistry(1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	owner := [32]byte{1}
	id, path, err := registry.Open(owner, backend)
	if err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		client, server := net.Pipe()
		done := make(chan error, 1)
		go func(attachment *Attachment) {
			defer server.Close()
			done <- registry.ServeDuplexPath(ctx, id, attachment, server, 256, time.Second)
		}(path)
		if release {
			if err := client.SetDeadline(deadline); err != nil {
				t.Fatal(err)
			}
			if err := RequestRelease(client); err != nil {
				t.Fatal(err)
			}
		}
		_ = client.Close()
		err = <-done
		if (release && err != nil) || (!release && !errors.Is(err, ErrPathIO)) {
			t.Fatalf("release=%v: %v", release, err)
		}
		path, err = registry.Resume(id, owner)
		if err != nil {
			t.Fatal(err)
		}
		if conn, err := path.Backend(); err != nil || conn != backend {
			t.Fatalf("backend changed after release=%v: %v", release, err)
		}
	}
	if err := registry.End(id, path); err != nil {
		t.Fatal(err)
	}
}
