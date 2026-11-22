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

func v102TaggedTCP(t *testing.T, marker byte) (string,func()) {
	t.Helper()
	ln,err:=net.Listen("tcp","127.0.0.1:0")
	if err!=nil{t.Fatal(err)}
	done:=make(chan struct{})
	go func(){
		defer close(done)
		for{
			c,err:=ln.Accept()
			if err!=nil{return}
			go func(conn net.Conn){
				defer conn.Close()
				buf:=make([]byte,4096)
				for {
					n,err:=conn.Read(buf)
					if n>0 {
						frame:=append([]byte{marker},buf[:n]...)
						if _,e:=conn.Write(frame);e!=nil{return}
					}
					if err!=nil{return}
				}
			}(c)
		}
	}()
	return ln.Addr().String(),func(){ln.Close();<-done}
}

func TestV102ExistingStreamKeepsBackendAfterAtomicRouteSwap(t *testing.T){
	a,stopA:=v102TaggedTCP(t,'A')
	defer stopA()
	b,stopB:=v102TaggedTCP(t,'B')
	defer stopB()
	cert,roots:=cert(t)
	id:=sha256.Sum256(cert.Certificate[0])
	acl:=map[string][]string{hex.EncodeToString(id[:]):{"vpn"}}
	p,err:=NewRuntimePolicy(acl,map[string][]Backend{"vpn":{{Name:"A",Address:a}}})
	if err!=nil{t.Fatal(err)}
	server:=&Server{RouteLookup:p.Select}
	https:=httptest.NewUnstartedServer(server)
	https.EnableHTTP2=true
	https.TLS=&tls.Config{
		MinVersion:tls.VersionTLS13, Certificates:[]tls.Certificate{cert},
		ClientAuth:tls.RequireAndVerifyClientCert, ClientCAs:roots, NextProtos:[]string{"h2"},
	}
	https.StartTLS()
	defer https.Close()
	client:=NewClient(&tls.Config{
		MinVersion:tls.VersionTLS13, RootCAs:roots,
		ServerName:"example.com",Certificates:[]tls.Certificate{cert},
		NextProtos:[]string{"h2"},
	},https.URL+"/v1/stream","vpn")
	defer client.HTTP.CloseIdleConnections()
	connect:=func()(net.Conn,chan error){
		local,peer:=net.Pipe()
		done:=make(chan error,1)
		go func(){done<-client.Handle(local)}()
		_ = peer.SetDeadline(time.Now().Add(12*time.Second))
		return peer,done
	}
	check:=func(peer net.Conn,marker byte){
		t.Helper()
		payload:=[]byte("hello")
		if _,err:=peer.Write(payload);err!=nil{t.Fatal(err)}
		out:=make([]byte,len(payload)+1)
		if _,err:=io.ReadFull(peer,out);err!=nil{t.Fatal(err)}
		if out[0]!=marker||!bytes.Equal(out[1:],payload){t.Fatalf("wrong backend: %q",out)}
	}
	original,originalDone:=connect()
	defer original.Close()
	check(original,'A')
	if err:=p.Replace(acl,map[string][]Backend{"vpn":{{Name:"B",Address:b}}});err!=nil{t.Fatal(err)}
	check(original,'A')
	next,nextDone:=connect()
	check(next,'B')
	next.Close()
	select{
	case <-nextDone:
	case <-time.After(5*time.Second):t.Fatal("new stream stuck")
	}
	check(original,'A')
	original.Close()
	select{
	case <-originalDone:
	case <-time.After(5*time.Second):t.Fatal("old stream stuck after route switch")
	}
}
