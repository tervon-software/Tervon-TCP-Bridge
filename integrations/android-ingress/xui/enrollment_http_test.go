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
        Service:"vpn",ClientCAPEM:caPEM,ServerCAPEM:caPEM,ClientCAKey:caKey,
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
        data.Service!="vpn" || data.ClientCA!=string(caPEM) ||
        data.ServerCA!=string(caPEM) {t.Fatal("incorrect discovery")}
    if len(data.TransportServices) != 0 { t.Fatal("legacy discovery unexpectedly advertised XHTTP") }
    if fingerprint!="" {t.Fatal("discovery activated client")}
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
        Service:"vpn",ClientCAPEM:caPEM,ServerCAPEM:caPEM,ClientCAKey:caKey,
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

func TestDiscoveryTransportCapabilities(t *testing.T) {
    now:=time.Now().UTC().Truncate(time.Second)
    _,caPEM,caKey:=testCertificateInputs(t,now)
    h:=&BridgeEnrollmentHTTP{
        RemoteURL:"https://bridge.example/v2/stream", ServerName:"bridge.example",
        Service:"vpn", TransportServices:map[string]string{"tcp":"vpn","xhttp":"vpn-xhttp","trojan_tcp":"vpn-trojan","trojan_xhttp":"vpn-trojan-xhttp"},
        ClientCAPEM:caPEM, ServerCAPEM:caPEM, ClientCAKey:caKey,
        Activate:func(string)error{return nil},
    }
    handler,err:=h.Handler()
    if err!=nil {t.Fatal(err)}
    rr:=httptest.NewRecorder()
    handler.ServeHTTP(rr,httptest.NewRequest(http.MethodGet,"https://sub.example/.well-known/tervon-bridge",nil))
    if rr.Code!=http.StatusOK { t.Fatalf("bad status %d",rr.Code) }
    var resp bridgeDiscoveryResponse
    if err:=json.Unmarshal(rr.Body.Bytes(),&resp);err!=nil {t.Fatal(err)}
    if resp.TransportServices["tcp"]!="vpn" || resp.TransportServices["xhttp"]!="vpn-xhttp" ||
        resp.TransportServices["trojan_tcp"]!="vpn-trojan" ||
        resp.TransportServices["trojan_xhttp"]!="vpn-trojan-xhttp" {
        t.Fatalf("incomplete capability map: %v",resp.TransportServices)
    }
    wrong:=*h
    wrong.TransportServices=map[string]string{"tcp":"vpn-xhttp"}
    if _,err:=wrong.Handler();err==nil {t.Fatal("accepted mismatched TCP service")}
    wrong=*h
    wrong.TransportServices=map[string]string{"tcp":"vpn","udp":"vpn-xhttp"}
    if _,err:=wrong.Handler();err==nil {t.Fatal("accepted unexpected UDP transport")}
}
