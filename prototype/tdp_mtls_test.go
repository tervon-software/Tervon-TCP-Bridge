package bridge

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tervon-tcp-bridge/tdp"
)

func TestTDPResumeRequiresOriginalMTLSIdentity(t *testing.T) {
	owner, ownerPool := cert(t)
	other, _ := cert(t)
	otherLeaf, err := x509.ParseCertificate(other.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	clientRoots := ownerPool.Clone()
	clientRoots.AddCert(otherLeaf)

	states := make(chan tls.ConnectionState, 2)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		states <- *r.TLS
		w.WriteHeader(http.StatusNoContent)
	}))
	server.TLS = &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{owner},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientRoots,
	}
	server.StartTLS()
	defer server.Close()

	connect := func(identity tls.Certificate) tls.ConnectionState {
		t.Helper()
		transport := &http.Transport{TLSClientConfig: &tls.Config{
			RootCAs:      ownerPool,
			ServerName:   "example.com",
			Certificates: []tls.Certificate{identity},
			MinVersion:   tls.VersionTLS13,
		}}
		defer transport.CloseIdleConnections()
		client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
		response, err := client.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusNoContent {
			t.Fatalf("unexpected status: %d", response.StatusCode)
		}
		select {
		case state := <-states:
			return state
		default:
			t.Fatal("missing server TLS state")
			return tls.ConnectionState{}
		}
	}

	registry, err := tdp.NewRegistry(1, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	backend, remote := net.Pipe()
	defer backend.Close()
	defer remote.Close()

	firstState := connect(owner)
	unverified := firstState
	unverified.VerifiedChains = nil
	if _, _, err := registry.OpenMTLS(&unverified, backend); !errors.Is(err, tdp.ErrUnverifiedPeer) {
		t.Fatalf("open without a verified chain: %v", err)
	}
	id, first, err := registry.OpenMTLS(&firstState, backend)
	if err != nil {
		t.Fatal(err)
	}
	registry.Detach(id, first)

	otherState := connect(other)
	if _, err := registry.ResumeMTLS(id, &otherState); !errors.Is(err, tdp.ErrSessionMissing) {
		t.Fatalf("resume with different trusted certificate: %v", err)
	}
	ownerState := connect(owner)
	resumed, err := registry.ResumeMTLS(id, &ownerState)
	if err != nil {
		t.Fatal(err)
	}
	if conn, err := resumed.Backend(); err != nil || conn != backend {
		t.Fatalf("backend changed after resume: %v, %v", conn, err)
	}
	if err := remote.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := registry.End(id, resumed); err != nil {
		t.Fatal(err)
	}
	if _, err := remote.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("backend still open after End: %v", err)
	}
}
