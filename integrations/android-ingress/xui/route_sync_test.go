package xui

import (
 "context"
 "errors"
 "strings"
 "testing"
)

func TestSyncVerifiedRoutesRequiresAppliedXrayAndCoreACK(t *testing.T){
 ctx:=context.Background()
 store:=new(ConfirmedRouteStore)
 panel:=func(context.Context)([]PanelInbound,error){return []PanelInbound{
  {Port:30000,Protocol:"trojan",Network:"tcp",Enabled:true},
  {Port:55636,Protocol:"vless",Network:"xhttp",Enabled:true},
 },nil}
 effective:=func(context.Context)([]EffectiveInbound,error){return []EffectiveInbound{
  {Port:30000,Protocol:"trojan",Network:"tcp",Service:"vpn",Address:"host.docker.internal:30000"},
  {Port:55636,Protocol:"vless",Network:"xhttp",Service:"vpn-xhttp",Address:"host.docker.internal:55636"},
 },nil}
 auth:=map[string][]string{strings.Repeat("a",64):{"vpn","vpn-xhttp"}}
 called:=0
 apply:=func(_ context.Context,routes map[string][]PolicyBackend,authorized map[string][]string,received string)error{
  called++
  expected,err:=PolicyGenerationDigest(routes,authorized)
  if err!=nil||received!=expected{t.Fatal("wrong combined digest")}
  if routes["vpn"][0].Address!="host.docker.internal:30000"||routes["vpn-xhttp"][0].Address!="host.docker.internal:55636"{t.Fatal("incorrect candidate")}
  routes["vpn"][0].Address="malicious-local-mutation"
  return nil
 }
 if err:=SyncVerifiedRoutes(ctx,store,panel,effective,auth,apply);err!=nil{t.Fatal(err)}
 routes,digest,ready:=store.Snapshot()
 if !ready||len(digest)!=64||routes["30000"]!="vpn"||routes["55636"]!="vpn-xhttp"||called!=1{t.Fatal("confirmed snapshot absent")}
 // Source changes: the selected Xray runtime no longer matches the panel.
 wrong:=func(context.Context)([]EffectiveInbound,error){
  return []EffectiveInbound{{Port:30000,Protocol:"trojan",Network:"tcp",Service:"vpn",Address:"host.docker.internal:30000"}},nil
 }
 if err:=SyncVerifiedRoutes(ctx,store,panel,wrong,auth,apply);err==nil{t.Fatal("accepted missing active inbound")}
 if _,_,ready:=store.Snapshot();ready{t.Fatal("stale routes still published")}
 if called!=1{t.Fatal("called core on unverified Xray")}
}

func TestSyncVerifiedRoutesFailClosedWhenAPIOrACKFails(t *testing.T){
 ctx:=context.Background()
 store:=new(ConfirmedRouteStore)
 auth:=map[string][]string{strings.Repeat("a",64):{"vpn"}}
 goodPanel:=func(context.Context)([]PanelInbound,error){return []PanelInbound{{Port:55535,Protocol:"vless",Network:"tcp",Enabled:true}},nil}
 goodXray:=func(context.Context)([]EffectiveInbound,error){return []EffectiveInbound{{Port:55535,Protocol:"vless",Network:"tcp",Service:"vpn",Address:"host.docker.internal:55535"}},nil}
 successful:=func(context.Context,map[string][]PolicyBackend,map[string][]string,string)error{return nil}
 if err:=SyncVerifiedRoutes(ctx,store,goodPanel,goodXray,auth,successful);err!=nil{t.Fatal(err)}
 apiFail:=func(context.Context)([]PanelInbound,error){return nil,errors.New("panel failed")}
 if err:=SyncVerifiedRoutes(ctx,store,apiFail,goodXray,auth,successful);err==nil{t.Fatal("panel failure accepted")}
 if _,_,ok:=store.Snapshot();ok{t.Fatal("stale discovery remained enabled")}
 if err:=SyncVerifiedRoutes(ctx,store,goodPanel,goodXray,auth,func(context.Context,map[string][]PolicyBackend,map[string][]string,string)error{return errors.New("ACK missing")});err==nil{t.Fatal("missing ACK accepted")}
 if _,_,ok:=store.Snapshot();ok{t.Fatal("unconfirmed policy advertised")}
 if err:=SyncVerifiedRoutes(ctx,store,goodPanel,goodXray,auth,successful);err!=nil{t.Fatal(err)}
 if _,_,ok:=store.Snapshot();!ok{t.Fatal("confirmed policy not restored")}
}
