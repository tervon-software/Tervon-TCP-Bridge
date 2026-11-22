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
    "context"
)

func TestBridgeEnrollmentHTTPDiscoveryAndCSR(t *testing.T) {
    now := time.Now().UTC().Truncate(time.Second)
    csr, caPEM, caKey := testCertificateInputs(t, now)
    var fingerprint string
    h := &BridgeEnrollmentHTTP{
        RemoteURL:"https://chosen-from-server.example:18443/v2/stream",
        ServerName:"chosen-from-server.example",
        Service:"vpn",ConfirmedRoutes:confirmedTestStore(t,"vpn"),ClientCAPEM:caPEM,ServerCAPEM:caPEM,ClientCAKey:caKey,
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
        Service:"vpn",ConfirmedRoutes:confirmedTestStore(t,"vpn"),ClientCAPEM:caPEM,ServerCAPEM:caPEM,ClientCAKey:caKey,
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

func confirmedTestStore(t *testing.T, service string) *ConfirmedRouteStore {
 t.Helper()
 s:=new(ConfirmedRouteStore)
 if err:=s.PublishAfterCoreACK(context.Background(),map[string]string{"443":service},strings.Repeat("a",64),func(context.Context,string)error{return nil});err!=nil{t.Fatal(err)}
 return s
}

func TestDiscoveryNoStaticRouteAndAmbiguousServiceFailsClosed(t *testing.T) {
 now:=time.Now().UTC().Truncate(time.Second)
 _,caPEM,caKey:=testCertificateInputs(t,now)
 h:=&BridgeEnrollmentHTTP{
  RemoteURL:"https://bridge.example/v2/stream",ServerName:"bridge.example",
  Service:"vpn",ConfirmedRoutes:confirmedTestStore(t,"vpn"),
  ClientCAPEM:caPEM,ServerCAPEM:caPEM,ClientCAKey:caKey,
  Activate:func(string)error{return nil},
 }
 handler,err:=h.Handler();if err!=nil{t.Fatal(err)}
 w:=httptest.NewRecorder()
 handler.ServeHTTP(w,httptest.NewRequest(http.MethodGet,"https://sub.example/.well-known/tervon-bridge",nil))
 if w.Code!=http.StatusOK{t.Fatalf("confirmed discovery: %d",w.Code)}
 var v map[string]any
 if err:=json.Unmarshal(w.Body.Bytes(),&v);err!=nil{t.Fatal(err)}
 if _,ok:=v["tcp_routes"];ok{t.Fatal("removed legacy field still advertised")}
 if v["policy_generation"]!=strings.Repeat("a",64){t.Fatal("incorrect generation")}
 noStore:=*h;noStore.ConfirmedRoutes=nil
 if _,err:=noStore.Handler();err==nil{t.Fatal("nil confirmed store accepted")}
 multi:=new(ConfirmedRouteStore)
 if err:=multi.PublishAfterCoreACK(context.Background(),map[string]string{"443":"vpn","8443":"another"},strings.Repeat("b",64),func(context.Context,string)error{return nil});err!=nil{t.Fatal(err)}
 ambiguous:=*h;ambiguous.ConfirmedRoutes=multi
 handler,err=ambiguous.Handler();if err!=nil{t.Fatal(err)}
 for _,method:=range []string{http.MethodGet,http.MethodPost}{
  w=httptest.NewRecorder()
  req:=httptest.NewRequest(method,"https://sub.example/.well-known/tervon-bridge",nil)
  handler.ServeHTTP(w,req)
  if w.Code!=http.StatusServiceUnavailable{t.Fatalf("ambiguous %s accepted %d",method,w.Code)}
 }
}

func TestDiscoveryV102OnlyAdvertisesAcknowledgedGeneration(t *testing.T){
 now:=time.Now().UTC().Truncate(time.Second)
 _,caPEM,caKey:=testCertificateInputs(t,now)
 store:=new(ConfirmedRouteStore)
 h:=&BridgeEnrollmentHTTP{
  RemoteURL:"https://bridge.example/v2/stream",ServerName:"bridge.example",
  Service:"vpn-tcp",
  ConfirmedRoutes:store,ClientCAPEM:caPEM,ServerCAPEM:caPEM,
  ClientCAKey:caKey,Activate:func(string)error{return nil},
 }
 handler,err:=h.Handler()
 if err!=nil{t.Fatal(err)}
 discover:=func()(int,bridgeDiscoveryResponse){
  w:=httptest.NewRecorder()
  handler.ServeHTTP(w,httptest.NewRequest(http.MethodGet,"https://sub.example/.well-known/tervon-bridge",nil))
  var payload bridgeDiscoveryResponse
  if w.Code==http.StatusOK{
   if err:=json.Unmarshal(w.Body.Bytes(),&payload);err!=nil{t.Fatal(err)}
  }
  return w.Code,payload
 }
 if code,_:=discover();code!=http.StatusServiceUnavailable{t.Fatalf("unconfirmed routes advertised: %d",code)}
 digest:=strings.Repeat("a",64)
 if err:=store.PublishAfterCoreACK(context.Background(),map[string]string{"55535":"vpn-tcp"},digest,func(_ context.Context,got string)error{
  if got!=digest{t.Fatal("wrong digest")}
  return nil
 });err!=nil{t.Fatal(err)}
 status,resp:=discover()
 if status!=http.StatusOK||resp.PolicyGeneration!=digest||resp.Service!="vpn-tcp"{
  t.Fatalf("confirmed generation not advertised: %d %+v",status,resp)
 }
 if err:=store.PublishAfterCoreACK(context.Background(),map[string]string{"55636":"xhttp"},strings.Repeat("b",64),func(context.Context,string)error{return io.ErrUnexpectedEOF});err==nil{
  t.Fatal("failed ACK accepted")
 }
 status,resp=discover()
 if status!=http.StatusServiceUnavailable{
  t.Fatalf("ambiguous core ACK must disable discovery: %d %+v",status,resp)
 }
}
