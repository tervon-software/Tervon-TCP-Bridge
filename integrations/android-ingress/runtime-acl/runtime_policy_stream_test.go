package bridge

import (
    "bytes"
    "crypto/sha256"
    "crypto/tls"
    "encoding/hex"
    "io"
    "net"
    "net/http/httptest"
    "testing"
    "time"
)

func policyEcho(t *testing.T, label byte) (string, func()) {
    t.Helper()
    ln, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil { t.Fatal(err) }
    go func() {
        for {
            conn, err := ln.Accept()
            if err != nil { return }
            go func() {
                defer conn.Close()
                data := make([]byte, 4096)
                for {
                    n, err := conn.Read(data)
                    if n > 0 {
                        response := append([]byte{label}, data[:n]...)
                        if _, e := conn.Write(response); e != nil { return }
                    }
                    if err != nil { return }
                }
            }()
        }
    }()
    return ln.Addr().String(), func(){ _ = ln.Close() }
}

func TestM6PolicyActiveHTTP2StreamSurvivesRouteAndACLSwap(t *testing.T) {
    a, stopA := policyEcho(t, 'A')
    defer stopA()
    b, stopB := policyEcho(t, 'B')
    defer stopB()
    cert, roots := cert(t)
    id := sha256.Sum256(cert.Certificate[0])
    fingerprint := hex.EncodeToString(id[:])
    first := map[string][]Backend{"vpn": {{Name:"old", Address:a}}}
    second := map[string][]Backend{"vpn": {{Name:"new", Address:b}}}
    allowed := map[string][]string{fingerprint: {"vpn"}}
    policy, err := NewRuntimePolicy(first, allowed)
    if err != nil { t.Fatal(err) }
    server := &Server{Routes:first, PolicyLookup:policy.Lookup}
    https := httptest.NewUnstartedServer(server)
    https.EnableHTTP2 = true
    https.TLS = &tls.Config{
        MinVersion:tls.VersionTLS13, Certificates:[]tls.Certificate{cert},
        ClientAuth:tls.RequireAndVerifyClientCert, ClientCAs:roots,
        NextProtos:[]string{"h2"},
    }
    https.StartTLS()
    defer https.Close()
    client := NewClient(&tls.Config{
        MinVersion:tls.VersionTLS13, RootCAs:roots,
        ServerName:"example.com", Certificates:[]tls.Certificate{cert},
        NextProtos:[]string{"h2"},
    }, https.URL+"/v1/stream", "vpn")
    defer client.HTTP.CloseIdleConnections()

    open := func() (net.Conn,chan error) {
        t.Helper()
        local, peer := net.Pipe()
        done := make(chan error, 1)
        go func(){done <- client.Handle(local)}()
        _ = peer.SetDeadline(time.Now().Add(15*time.Second))
        return peer, done
    }
    roundTrip := func(peer net.Conn, label byte, text string) {
        t.Helper()
        data := []byte(text)
        if _,err := peer.Write(data); err != nil { t.Fatal(err) }
        received := make([]byte,len(data)+1)
        if _,err := io.ReadFull(peer,received); err != nil { t.Fatal(err) }
        if !bytes.Equal(received,append([]byte{label},data...)) {
            t.Fatalf("stream expected backend %q, got %q",label,received)
        }
    }
    active, activeDone := open()
    defer active.Close()
    roundTrip(active,'A',"old-connection")
    if _,err := policy.Replace(second,allowed);err!=nil {t.Fatal(err)}
    // Existing stream keeps its old dialed socket after full policy swap.
    roundTrip(active,'A',"still-old")
    fresh, freshDone := open()
    roundTrip(fresh,'B',"new-connection")
    _ = fresh.Close()
    select {
    case <-freshDone:
    case <-time.After(5*time.Second): t.Fatal("new stream did not close")
    }
    // Remove the former device's admission while keeping new route valid.
    another := sha256.Sum256([]byte("another-device"))
    deniedACL := map[string][]string{hex.EncodeToString(another[:]):{"vpn"}}
    if _,err := policy.Replace(second, deniedACL);err!=nil {t.Fatal(err)}
    roundTrip(active,'A',"after-revoke")
    refused, refusedDone := open()
    _,_ = refused.Write([]byte("denied"))
    select {
    case err := <-refusedDone:
        if err == nil {t.Fatal("new revoked stream unexpectedly succeeded")}
    case <-time.After(5*time.Second): t.Fatal("revoked stream did not terminate")
    }
    _ = refused.Close()
    if _,err:=policy.Replace(first,allowed);err!=nil {t.Fatal(err)}
    reallowed, reallowedDone := open()
    roundTrip(reallowed,'A',"restored")
    _ = reallowed.Close()
    select {
    case <-reallowedDone:
    case <-time.After(5*time.Second):t.Fatal("restored stream did not close")
    }
    _ = active.Close()
    select {
    case <-activeDone:
    case <-time.After(5*time.Second):t.Fatal("existing stream did not close")
    }
}


func TestM6AndroidPortSelectorThreeH2Backends(t *testing.T) {
    a, stopA := policyEcho(t, 'X')
    defer stopA()
    b, stopB := policyEcho(t, 'V')
    defer stopB()
    c, stopC := policyEcho(t, 'T')
    defer stopC()
    _, portA, _ := net.SplitHostPort(a)
    _, portB, _ := net.SplitHostPort(b)
    _, portC, _ := net.SplitHostPort(c)
    certificate, roots := cert(t)
    id := sha256.Sum256(certificate.Certificate[0])
    acl := map[string][]string{hex.EncodeToString(id[:]): {"xhttp", "vless-tcp", "trojan-tcp"}}
    routes := map[string][]Backend{
        "xhttp": {{Name:"xhttp", Address:a}},
        "vless-tcp": {{Name:"vless-tcp", Address:b}},
        "trojan-tcp": {{Name:"trojan-tcp", Address:c}},
    }
    policy, err := NewRuntimePolicy(routes, acl)
    if err != nil { t.Fatal(err) }
    server := &Server{Routes:routes, PolicyLookup:policy.Lookup}
    https := httptest.NewUnstartedServer(server)
    https.EnableHTTP2 = true
    https.TLS = &tls.Config{
        MinVersion:tls.VersionTLS13, Certificates:[]tls.Certificate{certificate},
        ClientAuth:tls.RequireAndVerifyClientCert, ClientCAs:roots,
        NextProtos:[]string{"h2"},
    }
    https.StartTLS()
    defer https.Close()
    for _, item := range []struct {port string; label byte}{
        {portA,'X'}, {portB,'V'}, {portC,'T'},
    } {
        t.Run(item.port,func(t *testing.T) {
            client := NewClient(&tls.Config{
                MinVersion:tls.VersionTLS13, RootCAs:roots,
                ServerName:"example.com", Certificates:[]tls.Certificate{certificate},
                NextProtos:[]string{"h2"},
            }, https.URL+"/v1/stream", item.port)
            defer client.HTTP.CloseIdleConnections()
            local, peer := net.Pipe()
            done := make(chan error,1)
            go func(){done <- client.Handle(local)}()
            defer peer.Close()
            if err := peer.SetDeadline(time.Now().Add(15*time.Second)); err != nil { t.Fatal(err) }
            payload := []byte("profile-dynamic")
            if _,err := peer.Write(payload);err != nil {t.Fatal(err)}
            received := make([]byte,len(payload)+1)
            if _,err := io.ReadFull(peer,received);err != nil {t.Fatal(err)}
            expected := append([]byte{item.label},payload...)
            if !bytes.Equal(received,expected) {t.Fatalf("port %s reached wrong backend",item.port)}
            _ = peer.Close()
            select {
            case <-done:
            case <-time.After(5*time.Second):t.Fatal("port-selected H2 stream did not terminate")
            }
        })
    }
}
