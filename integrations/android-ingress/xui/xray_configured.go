package xui

import (
 "bytes"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net"
 "strconv"
 "strings"
)

// XrayBackendResolver is an operator-controlled function that checks whether
// a local Bridge backend is reachable from the Bridge process namespace.
// The Xray configuration alone is not allowed to choose arbitrary dial targets.
type XrayBackendResolver func(port int) (service string, backendAddress string, err error)

// ParseXrayConfiguredInbounds extracts candidates from a locally supplied
// Xray JSON config. This is CONFIGURED_ONLY: it cannot prove that the currently
// running Xray process has actually applied this exact configuration.
// A caller MUST provide additional runtime validation before publishing routes.
func ParseXrayConfiguredInbounds(data []byte, resolve XrayBackendResolver) ([]EffectiveInbound, error) {
 if resolve == nil {return nil,errors.New("Xray backend resolver required")}
 if len(data)==0 || len(data)>8<<20{return nil,errors.New("Xray config size invalid")}
 d:=json.NewDecoder(bytes.NewReader(data))
 var cfg struct{Inbounds []struct{
  Protocol string `json:"protocol"`
  Port json.RawMessage `json:"port"`
  StreamSettings struct{Network string `json:"network"`} `json:"streamSettings"`
 } `json:"inbounds"`}
 if err:=d.Decode(&cfg);err!=nil{return nil,errors.New("Xray config invalid JSON")}
 var trailing any
 if err:=d.Decode(&trailing);err!=io.EOF{return nil,errors.New("Xray config has trailing data")}
 if len(cfg.Inbounds)>4096{return nil,errors.New("Xray inbound count exceeded")}
 out:=make([]EffectiveInbound,0,len(cfg.Inbounds))
 seen:=make(map[int]bool,len(cfg.Inbounds))
 for _,in:=range cfg.Inbounds{
  if len(in.Port)==0 || strings.TrimSpace(in.Protocol)=="" {return nil,errors.New("Xray inbound identity or port missing")}
  var port int
  if err:=json.Unmarshal(in.Port,&port);err!=nil||port<1||port>65535{return nil,errors.New("Xray inbound TCP port invalid")}
  network:=strings.ToLower(in.StreamSettings.Network)
  if network==""{network="tcp"}
  switch network {
  case "tcp","raw","ws","grpc","httpupgrade","http","h2","xhttp","splithttp":
  default: continue
  }
  // Only VPN TCP inbounds are candidates for Bridge routing. Service\n  // listeners must never require a backend resolver or block VPN ports.\n  if !bridgeEligibleProtocol(in.Protocol) {continue}\n  if seen[port]{return nil,errors.New("duplicate TCP Xray inbound port")}
  seen[port]=true
  service,address,err:=resolve(port)
  if err!=nil{return nil,errors.New("trusted Xray backend resolution failed")}
  if !routeServicePattern.MatchString(service){return nil,errors.New("invalid resolved Xray service")}
  host,rawPort,e:=net.SplitHostPort(address)
  n,parseErr:=strconv.Atoi(rawPort)
  if e!=nil||parseErr!=nil||host==""||n!=port{return nil,fmt.Errorf("resolved Xray backend address mismatch at port %d",port)}
  out=append(out,EffectiveInbound{
   Port:port,Protocol:strings.ToLower(in.Protocol),Network:network,Service:service,Address:address,
  })
 }
 return out,nil
}
