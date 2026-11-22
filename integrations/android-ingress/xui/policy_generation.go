package xui

import (
 "crypto/sha256"
 "encoding/json"
 "errors"
 "fmt"
 "net"
 "strconv"
)

// PolicyBackend mirrors the two exported Go fields of bridge.Backend.
// This is the exact shape marshaled by the staged v1.0.2 core ACK.
type PolicyBackend struct {
 Name string
 Address string
}

// PolicyGenerationDigest serializes exactly the combined policy data
// logged by cmd/bridge on successful M6_POLICY_RELOAD, with routes first
// and authorized second. It does not itself verify Docker applied it.
func PolicyGenerationDigest(routes map[string][]PolicyBackend, authorized map[string][]string)(string,error){
 if len(routes)==0||len(routes)>4096{return "",errors.New("invalid policy routes")}
 for name,backends:=range routes{
  if !routeServicePattern.MatchString(name)||len(backends)==0||len(backends)>64{return "",errors.New("invalid policy service or backends")}
  for _,backend:=range backends{
   host,rawPort,err:=net.SplitHostPort(backend.Address)
   port,parseErr:=strconv.Atoi(rawPort)
   if err!=nil||host==""||parseErr!=nil||port<1||port>65535{return "",errors.New("invalid policy backend")}
  }
 }
 if len(authorized)==0{return "",errors.New("missing policy authorization")}
 // Do not reorder fields, change json tags on Backend, or normalize ACL
 // arrays: the digest must match the bytes emitted by the bridge core.
 payload,err:=json.Marshal(struct{
  Routes map[string][]PolicyBackend `json:"routes"`
  Authorized map[string][]string `json:"authorized"`
 }{routes,authorized})
 if err!=nil{return "",errors.New("policy serialization failed")}
 sum:=sha256.Sum256(payload)
 return fmt.Sprintf("%x",sum),nil
}
