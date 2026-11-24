package xui

import (
 "context"
 "errors"
 "regexp"
 "strconv"
 "sync"
)

// ConfirmedRouteStore holds only route mappings acknowledged by the runtime.
// No fallback to a static M6 discovery map is permitted.
// The single coordinator owning this store MUST serialize runtime writes and
// ACK observation. Successful confirmation refers to a complete policy digest
// (routes AND ACL), never the old M6 ACL-only reload digest.
type ConfirmedRouteStore struct {
 mu sync.RWMutex
 current map[string]string
 digest string
 ready bool
}

var policyDigestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Snapshot returns a defensive copy. Absence of a confirmed generation means
// discovery must fail closed. A confirmed empty map means no ports are allowed.
func(s *ConfirmedRouteStore) Snapshot()(map[string]string,string,bool){
 if s==nil{return nil,"",false}
 s.mu.RLock()
 defer s.mu.RUnlock()
 if !s.ready{return nil,"",false}
 routes:=make(map[string]string,len(s.current))
 for k,v:=range s.current{routes[k]=v}
 return routes,s.digest,true
}

// PublishAfterCoreACK checks a complete proposed set before invoking a trusted
// confirmation callback. This callback MUST install/verify the exact combined
// ACL+Routes digest on the target Decapsulator before returning nil.
// Serial publication prevents out-of-order ACKs in this process.
// It cannot independently prove Xray runtime state, and must only receive
// routes already validated against the selected node and effective Xray.
func(s *ConfirmedRouteStore) PublishAfterCoreACK(ctx context.Context, routes map[string]string, digest string, confirm func(context.Context,string)error)error{
 if s==nil||confirm==nil{return errors.New("confirmed route publisher not configured")}
 if !policyDigestPattern.MatchString(digest)||routes==nil||len(routes)>4096{return errors.New("invalid route generation")}
 copied:=make(map[string]string,len(routes))
 for port,service:=range routes{
  n,err:=strconv.Atoi(port)
  if err!=nil||n<1||n>65535||strconv.Itoa(n)!=port||!routeServicePattern.MatchString(service){
   return errors.New("invalid discovered route")
  }
  copied[port]=service
 }
 if ctx==nil{return errors.New("route update context required")}
 s.mu.Lock()
 defer s.mu.Unlock()
 if err:=ctx.Err();err!=nil{return err}
 // After attempting a core update, failure/timeout has ambiguous outcome:
 // core may have applied a new policy without the ACK reaching us.
 // Fail closed for NEW discovery rather than advertise a possibly stale map.
 if s.ready && s.digest==digest{return nil}
 if err:=confirm(ctx,digest);err!=nil{
  s.ready=false
  return errors.New("Decapsulator policy generation not confirmed")
 }
 if err:=ctx.Err();err!=nil{
  s.ready=false
  return err
 }
 s.current=copied
 s.digest=digest
 s.ready=true
 return nil
}

// Invalidate suspends new discovery when the provider/Xray runtime snapshot is
// no longer verified. Existing core TCP connections are not changed.
func(s *ConfirmedRouteStore) Invalidate(){
 if s==nil{return}
 s.mu.Lock()
 defer s.mu.Unlock()
 s.ready=false
}
