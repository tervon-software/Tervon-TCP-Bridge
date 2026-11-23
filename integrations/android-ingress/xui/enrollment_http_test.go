package xui

import (
    "bytes"
    "crypto/x509"
    "encoding/json"
    "encoding/pem"
    "io"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"
)

func TestBridgeEnrollmentHTTPDiscoveryAndCSR(t *testing.T) {
    now := time.Now().UTC().Truncate(time.Second)
    csr, caPEM, caKey := testCertificateInputs(t, now)
    var fingerprint string
    h := &BridgeEnrollmentHTTP{
        RemoteURL:"https://chosen-from-server.example:18443/v2/stream",
        ServerName:"chosen-from-server.example",
        ClientCAPEM:caPEM,ServerCAPEM:caPEM,ClientCAKey:caKey,
        Now:func()time.Time{return now},
        Activate:func(fp string)error{fingerprint=fp;return nil},
    }
    handler,err:=h.Handler()
    if err!=nil{t.Fatal(err)}
    discovery:=httptest.NewRecorder()
    handler.ServeHTTP(discovery,httptest.NewRequest(http.MethodGet,"https://sub.example/.well-known/tervon-bridge",nil))
    if discovery.Code!=http.StatusOK{t.Fatalf("GET %d",discovery.Code)}
    var data bridgeDiscoveryResponse
    if err:=json.Unmarshal(discovery.Body.Bytes(),&data);err!=nil{t.Fatal(err)}
    if !data.Available || data.RemoteURL!=h.RemoteURL || data.ServerName!=h.ServerName ||
        data.ClientCA!=string(caPEM) ||
        data.ServerCA!=string(caPEM) {t.Fatal("incorrect discovery")}
    if fingerprint!="" {t.Fatal("discovery activated client")}
    var fields map[string]any
    if err:=json.Unmarshal(discovery.Body.Bytes(),&fields);err!=nil{t.Fatal(err)}
    if _,hasService:=fields["service"];hasService{t.Fatal("registrar must not select a route")}

    request:=httptest.NewRequest(http.MethodPost,"https://sub.example/.well-known/tervon-bridge",bytes.NewReader(csr))
    request.Header.Set("Content-Type","application/pkcs10")
    response:=httptest.NewRecorder()
    handler.ServeHTTP(response,request)
    if response.Code!=http.StatusOK{t.Fatalf("POST %d %s",response.Code,response.Body.String())}
    var issued bridgeEnrollmentResponse
    if err:=json.Unmarshal(response.Body.Bytes(),&issued);err!=nil{t.Fatal(err)}
    block,_:=pem.Decode([]byte(issued.Certificate))
    if block==nil || fingerprint==""{t.Fatal("missing certificate or activated fingerprint")}
    leaf,err:=x509.ParseCertificate(block.Bytes)
    if err!=nil{t.Fatal(err)}
    if len(leaf.ExtKeyUsage)!=1 || leaf.ExtKeyUsage[0]!=x509.ExtKeyUsageClientAuth {
        t.Fatal("wrong leaf purpose")
    }
    if !strings.Contains(response.Header().Get("Cache-Control"),"no-store"){t.Fatal("missing no-store")}
}

func TestBridgeEnrollmentHTTPRefusesUnsafeRequestsAndActivationFailures(t *testing.T) {
    now:=time.Now().UTC().Truncate(time.Second)
    csr,caPEM,caKey:=testCertificateInputs(t,now)
    base:=&BridgeEnrollmentHTTP{
        RemoteURL:"https://dynamic.example/v2/stream",ServerName:"dynamic.example",
        ClientCAPEM:caPEM,ServerCAPEM:caPEM,ClientCAKey:caKey,
        Activate:func(string)error{return io.ErrUnexpectedEOF},
    }
    h,err:=base.Handler();if err!=nil{t.Fatal(err)}
    post:=func(body []byte,contentType string)int{
        req:=httptest.NewRequest(http.MethodPost,"https://sub.example/.well-known/tervon-bridge",bytes.NewReader(body))
        req.Header.Set("Content-Type",contentType)
        w:=httptest.NewRecorder();h.ServeHTTP(w,req);return w.Code
    }
    if got:=post(csr,"application/pkcs10");got!=http.StatusServiceUnavailable {
        t.Fatalf("ACL failure must fail enrollment: %d",got)
    }
    if got:=post([]byte("invalid"),"application/pkcs10");got!=http.StatusBadRequest {
        t.Fatalf("bad CSR returned %d",got)
    }
    if got:=post(csr,"text/plain");got!=http.StatusUnsupportedMediaType {
        t.Fatalf("wrong content type returned %d",got)
    }
    if got:=post(bytes.Repeat([]byte("x"),8193),"application/pkcs10");got!=http.StatusBadRequest {
        t.Fatalf("oversized request returned %d",got)
    }
    for _,method:=range []string{http.MethodDelete,http.MethodPut}{
        w:=httptest.NewRecorder()
        h.ServeHTTP(w,httptest.NewRequest(method,"https://sub.example/.well-known/tervon-bridge",nil))
        if w.Code!=http.StatusMethodNotAllowed{t.Fatalf("%s returned %d",method,w.Code)}
    }
    noBridge:=*base
    noBridge.Activate=nil
    if _,err:=noBridge.Handler();err==nil{t.Fatal("unconfigured Bridge accepted")}
    missingServerCA:=*base
    missingServerCA.ServerCAPEM=nil
    if _,err:=missingServerCA.Handler();err==nil{t.Fatal("missing server trust CA accepted")}
    badURL:=*base
    badURL.RemoteURL="http://dynamic.example/v2/stream"
    if _,err:=badURL.Handler();err==nil{t.Fatal("insecure endpoint accepted")}
}

func TestBridgeEnrollmentHTTPWithoutServerRouteCoordination(t *testing.T) {
    now := time.Now().UTC().Truncate(time.Second)
    csr, caPEM, caKey := testCertificateInputs(t, now)
    activations := 0
    h := &BridgeEnrollmentHTTP{
        RemoteURL:"https://bridge.example/v2/stream",
        ServerName:"bridge.example",
        ClientCAPEM:caPEM,ServerCAPEM:caPEM,ClientCAKey:caKey,
        Activate:func(string)error{activations++;return nil},
    }
    handler, err := h.Handler()
    if err != nil { t.Fatal(err) }
    discovery := httptest.NewRecorder()
    handler.ServeHTTP(discovery,httptest.NewRequest(http.MethodGet,"https://sub.example/.well-known/tervon-bridge",nil))
    if discovery.Code != http.StatusOK { t.Fatalf("Discovery failed without routing coordination: %d",discovery.Code) }
    var result map[string]any
    if err := json.Unmarshal(discovery.Body.Bytes(), &result); err != nil { t.Fatal(err) }
    if _,present := result["service"]; present { t.Fatal("registrar must not publish route selection") }
    if _,present := result["tcp_routes"]; present { t.Fatal("legacy routes leaked into Discovery") }
    if _,present := result["policy_generation"]; present { t.Fatal("server policy leaked into Discovery") }
    if activations != 0 { t.Fatal("GET must not activate ACL") }
    request := httptest.NewRequest(http.MethodPost,"https://sub.example/.well-known/tervon-bridge",bytes.NewReader(csr))
    request.Header.Set("Content-Type","application/pkcs10")
    enrolled := httptest.NewRecorder()
    handler.ServeHTTP(enrolled,request)
    if enrolled.Code != http.StatusOK { t.Fatalf("CSR enrollment failed: %d %s",enrolled.Code,enrolled.Body.String()) }
    if activations != 1 { t.Fatalf("ACL activation count %d",activations) }
}
