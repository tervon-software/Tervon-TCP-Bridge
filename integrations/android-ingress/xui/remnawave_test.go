package xui

import (
 "context"
 "fmt"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
)
const nodeID="123e4567-e89b-12d3-a456-426614174000"
const profileID="223e4567-e89b-12d3-a456-426614174000"
const activeID="323e4567-e89b-12d3-a456-426614174000"
const inactiveID="423e4567-e89b-12d3-a456-426614174000"

func remnaTestClient(t *testing.T,node,profile string)(*RemnawaveClient,func()){
 t.Helper()
 s:=httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.Header.Get("Authorization")!="Bearer dummy-token"{t.Error("invalid bearer")}
  switch r.URL.Path{
  case "/api/nodes/"+nodeID:fmt.Fprint(w,node)
  case "/api/config-profiles/"+profileID+"/inbounds":fmt.Fprint(w,profile)
  default:w.WriteHeader(http.StatusNotFound)
  }
 }))
 c,err:=NewRemnawaveClient(s.URL,"dummy-token")
 if err!=nil{t.Fatal(err)}
 c.httpClient=s.Client()
 return c,s.Close
}
func TestRemnawaveSelectedNodeFiltersInactive(t *testing.T){
 node:=fmt.Sprintf(`{"response":{"uuid":%q,"isDisabled":false,"configProfile":{"activeConfigProfileUuid":%q,"activeInbounds":[{"uuid":%q,"profileUuid":%q,"port":7771,"type":"vless","network":"tcp"}]}}}`,nodeID,profileID,activeID,profileID)
 profile:=fmt.Sprintf(`{"response":[{"uuid":%q,"profileUuid":%q,"port":7771,"type":"vless","network":"tcp"},{"uuid":%q,"profileUuid":%q,"port":7772,"type":"trojan","network":"tcp"}]}`,activeID,profileID,inactiveID,profileID)
 c,closeFn:=remnaTestClient(t,node,profile);defer closeFn()
 found,err:=c.SelectedNodeInbounds(context.Background(),nodeID)
 if err!=nil{t.Fatal(err)}
 if len(found)!=1||found[0].Port!=7771||!found[0].Enabled{t.Fatalf("incorrect node-scoped snapshot %+v",found)}
}
func TestRemnawaveRejectsInconsistentNode(t *testing.T){
 validProfile:=fmt.Sprintf(`{"response":[{"uuid":%q,"profileUuid":%q,"port":7771,"type":"vless","network":"tcp"}]}`,activeID,profileID)
 for _,tc:=range []struct{name,node,profile string}{
  {"disabled",fmt.Sprintf(`{"response":{"uuid":%q,"isDisabled":true,"configProfile":{"activeConfigProfileUuid":%q,"activeInbounds":[]}}}`,nodeID,profileID),validProfile},
  {"unassigned",fmt.Sprintf(`{"response":{"uuid":%q,"isDisabled":false,"configProfile":{"activeConfigProfileUuid":"","activeInbounds":[]}}}`,nodeID),validProfile},
  {"missing-selected-inbound",fmt.Sprintf(`{"response":{"uuid":%q,"isDisabled":false,"configProfile":{"activeConfigProfileUuid":%q,"activeInbounds":[{"uuid":%q}]}}}`,nodeID,profileID,inactiveID),validProfile},
  {"changed-port",fmt.Sprintf(`{"response":{"uuid":%q,"isDisabled":false,"configProfile":{"activeConfigProfileUuid":%q,"activeInbounds":[{"uuid":%q,"port":7772}]}}}`,nodeID,profileID,activeID),validProfile},
  {"malformed-profile",fmt.Sprintf(`{"response":{"uuid":%q,"isDisabled":false,"configProfile":{"activeConfigProfileUuid":%q,"activeInbounds":[{"uuid":%q}]}}}`,nodeID,profileID,activeID),`{"response":{"foo":4}}`},
 }{t.Run(tc.name,func(t *testing.T){
  c,closeFn:=remnaTestClient(t,tc.node,tc.profile);defer closeFn()
  if _,err:=c.SelectedNodeInbounds(context.Background(),nodeID);err==nil{t.Fatal("accepted inconsistent node")}
 })}
}
func TestRemnawaveDoesNotExposeAPISecret(t *testing.T){
 c,closeFn:=remnaTestClient(t,`{"error":"dummy-token"}`,`{"response":[]}`);defer closeFn()
 _,err:=c.SelectedNodeInbounds(context.Background(),nodeID)
 if err==nil||strings.Contains(err.Error(),"dummy-token"){t.Fatalf("unsafe error: %v",err)}
 if _,err:=c.SelectedNodeInbounds(context.Background(),"../escape");err==nil{t.Fatal("accepted invalid node ID")}
}
