package xui

import (
 "crypto/sha256"
 "encoding/json"
 "fmt"
 "strings"
 "testing"
)

func TestPolicyGenerationDigestMatchesCoreJSONContract(t *testing.T){
 routes:=map[string][]PolicyBackend{
  "vpn-tcp":{{Name:"main",Address:"host.docker.internal:55535"}},
  "vpn":{{Name:"primary",Address:"host.docker.internal:30000"}},
 }
 authorized:=map[string][]string{strings.Repeat("a",64):{"vpn","vpn-tcp"}}
 got,err:=PolicyGenerationDigest(routes,authorized)
 if err!=nil{t.Fatal(err)}
 // Independent encoding models the v1.0.2 core's actual bridge.Backend.
 type backend struct{Name,Address string}
 coreRoutes:=map[string][]backend{
  "vpn":{{"primary","host.docker.internal:30000"}},
  "vpn-tcp":{{"main","host.docker.internal:55535"}},
 }
 coreJSON,err:=json.Marshal(struct{
  Routes map[string][]backend `json:"routes"`
  Authorized map[string][]string `json:"authorized"`
 }{coreRoutes,authorized})
 if err!=nil{t.Fatal(err)}
 want:=fmt.Sprintf("%x",sha256.Sum256(coreJSON))
 if got!=want{t.Fatalf("core digest mismatch: %s != %s",got,want)}
 again,err:=PolicyGenerationDigest(routes,authorized)
 if err!=nil||got!=again{t.Fatal("non-deterministic generation")}
 routes["vpn"]=[]PolicyBackend{{Name:"primary",Address:"host.docker.internal:30001"}}
 different,err:=PolicyGenerationDigest(routes,authorized)
 if err!=nil||got==different{t.Fatal("route change did not change digest")}
}

func TestPolicyGenerationDigestRejectsInvalidInput(t *testing.T){
 auth:=map[string][]string{strings.Repeat("a",64):{"vpn"}}
 for _,routes:=range []map[string][]PolicyBackend{
  nil,
  {"vpn":nil},
  {"bad service":{{Name:"main",Address:"127.0.0.1:30000"}}},
  {"vpn":{{Name:"main",Address:"127.0.0.1:0"}}},
  {"vpn":{{Name:"main",Address:"host:bad"}}},
 }{
  if _,err:=PolicyGenerationDigest(routes,auth);err==nil{t.Fatalf("accepted invalid routes %+v",routes)}
 }
 if _,err:=PolicyGenerationDigest(map[string][]PolicyBackend{"vpn":{{Name:"main",Address:"127.0.0.1:30000"}}},nil);err==nil{
  t.Fatal("accepted missing ACL")
 }
}
