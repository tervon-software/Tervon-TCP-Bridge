package xui

import (
 "errors"
 "fmt"
 "net"
 "regexp"
 "strconv"
 "strings"
)

// PanelInbound is a provider-neutral, redacted inbound projection.
// For Remnawave the provider adapter MUST filter by the selected node's
// assigned profile and active inbounds before calling ReconcileInbounds.
type PanelInbound struct {
 Port int
 Protocol string
 Network string
 Enabled bool
}

// EffectiveInbound describes the effective node-local Xray listener.
// It is independent of panel state and cannot be supplied by Android.
type EffectiveInbound struct {
 Port int
 Protocol string
 Network string
 Service string // server-selected opaque Bridge service identifier
 Address string // verified backend address, not a client-provided target
}

// VerifiedRoute can be published only AFTER the exact route+ACL generation
// has been acknowledged by the running Decapsulator.
type VerifiedRoute struct {
 Port int
 Service string
 Address string
}

var routeServicePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// ReconcileInbounds matches a provider-selected set with the effective Xray
// listeners. It does NOT publish routes, mutate ACL or assume panel == Xray.
// Duplicate/ambiguous ports fail closed; mismatches are not advertised.
// The caller MUST check which node this snapshot belongs to, and separately
// verify Bridge backend locality and the generation acknowledged by the core.
func ReconcileInbounds(panel []PanelInbound, effective []EffectiveInbound) ([]VerifiedRoute, error) {
 seenPanel := make(map[int]struct{},len(panel))
 seenEffective := make(map[int]struct{},len(effective))
 active := make(map[int]PanelInbound,len(panel))
 for _, p := range panel {
  if p.Port < 1 || p.Port > 65535 || strings.TrimSpace(p.Protocol)=="" || strings.TrimSpace(p.Network)=="" {
   return nil,errors.New("invalid panel inbound")
  }
  if _,exists:=seenPanel[p.Port];exists{return nil,errors.New("duplicate panel inbound port")}
  seenPanel[p.Port]=struct{}{}
  if p.Enabled {active[p.Port]=p}
 }
 for _, x := range effective {
  if x.Port < 1 || x.Port > 65535 || strings.TrimSpace(x.Protocol)=="" || strings.TrimSpace(x.Network)=="" ||
   !routeServicePattern.MatchString(x.Service) {
   return nil,errors.New("invalid effective inbound")
  }
  host, port,err:=net.SplitHostPort(x.Address)
  n,convErr:=strconv.Atoi(port)
  if err!=nil || convErr!=nil || n!=x.Port || host=="" {
   return nil,errors.New("effective inbound backend mismatch")
  }
  if _,exists:=seenEffective[x.Port];exists{return nil,errors.New("duplicate effective inbound port")}
  seenEffective[x.Port]=struct{}{}
 }
 routes:=make([]VerifiedRoute,0,len(active))
 for _, x := range effective {
  p,ok:=active[x.Port]
  if !ok {continue}
  if !strings.EqualFold(p.Protocol,x.Protocol) || !strings.EqualFold(p.Network,x.Network) {
   continue
  }
  switch strings.ToLower(x.Network) {
  case "tcp","raw","ws","grpc","httpupgrade","http","h2","xhttp","splithttp":
  default: continue // UDP and unrecognized transports are never treated as TCP
  }
  routes=append(routes,VerifiedRoute{Port:x.Port,Service:x.Service,Address:x.Address})
 }
 if len(routes)>4096{return nil,fmt.Errorf("too many effective routes")}
 return routes,nil
}
