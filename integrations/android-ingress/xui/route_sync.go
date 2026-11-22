package xui

import (
 "context"
 "errors"
 "strconv"
)

// RouteEvidenceSource is the existing provider adapter or runtime Xray
// verifier supplied by the trusted server coordinator. An Xray JSON parser
// alone is NOT a runtime verifier: the callback must prove the active
// process is using the supplied configuration and backend is reachable.
type RouteEvidenceSource func(context.Context)([]PanelInbound,error)
type AppliedXrayEvidence func(context.Context)([]EffectiveInbound,error)

// RoutePolicyApply must atomically stage EXACTLY the provided complete
// routes+ACL configuration and obtain a real generation ACK from the
// preapproved Decapsulator candidate (not the legacy ACL-only image).
// Return nil ONLY if the actual core reported the exact expected digest.
// This is an isolated coordinator seam, not a Docker deployment command.
type RoutePolicyApply func(context.Context,map[string][]PolicyBackend,map[string][]string,string)error

func copiedAuthorizations(in map[string][]string)map[string][]string{
 out:=make(map[string][]string,len(in))
 for k,v:=range in{out[k]=append([]string(nil),v...)}
 return out
}

func copiedPolicyRoutes(in map[string][]PolicyBackend)map[string][]PolicyBackend{
 out:=make(map[string][]PolicyBackend,len(in))
 for k,v:=range in{out[k]=append([]PolicyBackend(nil),v...)}
 return out
}

// SyncVerifiedRoutes uses the existing ReconcileInbounds, core digest and
// ConfirmedRouteStore. It does not poll, call Docker or mutate live Xray.
// The owner must serialize all policy/ACL writers and guarantee the apply
// callback uses the same immutable generation sent for ACK verification.
func SyncVerifiedRoutes(ctx context.Context,store *ConfirmedRouteStore,
 panel RouteEvidenceSource,applied AppliedXrayEvidence,
 authorized map[string][]string,apply RoutePolicyApply)error{
 if store==nil{return errors.New("confirmed route store missing")}
 fail:=func(err error)error{store.Invalidate();return err}
 if ctx==nil||panel==nil||applied==nil||apply==nil{
  return fail(errors.New("route synchronization dependency missing"))
 }
 p,err:=panel(ctx)
 if err!=nil{return fail(errors.New("panel inbound evidence unavailable"))}
 e,err:=applied(ctx)
 if err!=nil{return fail(errors.New("applied Xray evidence unavailable"))}
 verified,err:=ReconcileInbounds(p,e)
 if err!=nil{return fail(err)}
 active:=0
 for _,in:=range p{if in.Enabled{active++}}
 if active==0||len(verified)!=active{
  return fail(errors.New("selected panel inbounds not fully verified by applied Xray"))
 }
 discovered:=make(map[string]string,len(verified))
 routes:=make(map[string][]PolicyBackend,len(verified))
 services:=make(map[string]bool,len(verified))
 for _,v:=range verified{
  key:=strconv.Itoa(v.Port)
  if services[v.Service]{return fail(errors.New("ambiguous Bridge service for multiple ports"))}
  services[v.Service]=true
  discovered[key]=v.Service
  routes[v.Service]=[]PolicyBackend{{Name:v.Service,Address:v.Address}}
 }
 // The complete candidate ACL must authorize only services that have a
 // resolved route; no missing ACL route can enter the acknowledged policy.
 if len(authorized)==0{return fail(errors.New("empty Bridge ACL"))}
 for _,allowed:=range authorized{
  if len(allowed)==0{return fail(errors.New("empty Bridge ACL entry"))}
  for _,service:=range allowed{
   if !services[service]{return fail(errors.New("authorized Bridge service has no verified route"))}
  }
 }
 auth:=copiedAuthorizations(authorized)
 digest,err:=PolicyGenerationDigest(routes,auth)
 if err!=nil{return fail(err)}
 return store.PublishAfterCoreACK(ctx,discovered,digest,func(ctx context.Context,expected string)error{
  return apply(ctx,copiedPolicyRoutes(routes),copiedAuthorizations(auth),expected)
 })
}
