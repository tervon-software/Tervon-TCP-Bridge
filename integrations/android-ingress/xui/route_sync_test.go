package xui

import (
 "context"
 "errors"
 "strings"
 "os"
 "path/filepath"
 "fmt"
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

func TestSyncVerifiedRoutesIgnoresServiceInbounds(t *testing.T) {
 ctx:=context.Background()
 store:=new(ConfirmedRouteStore)
 panel:=func(context.Context)([]PanelInbound,error){return []PanelInbound{
  {Port:10811,Protocol:"socks",Network:"tcp",Enabled:true},
  {Port:8080,Protocol:"http",Network:"tcp",Enabled:true},
  {Port:30000,Protocol:"vless",Network:"tcp",Enabled:true},
  {Port:55535,Protocol:"vless",Network:"splithttp",Enabled:true},
  {Port:55636,Protocol:"trojan",Network:"tcp",Enabled:true},
 },nil}
 effective:=func(context.Context)([]EffectiveInbound,error){return []EffectiveInbound{
  {Port:10811,Protocol:"socks",Network:"tcp",Service:"socks",Address:"127.0.0.1:10811"},
  {Port:30000,Protocol:"vless",Network:"raw",Service:"30000",Address:"127.0.0.1:30000"},
  {Port:55535,Protocol:"vless",Network:"xhttp",Service:"55535",Address:"127.0.0.1:55535"},
  {Port:55636,Protocol:"trojan",Network:"tcp",Service:"55636",Address:"127.0.0.1:55636"},
 },nil}
 auth:=map[string][]string{strings.Repeat("a",64):{"30000","55535","55636"}}
 called:=0
 apply:=func(_ context.Context,routes map[string][]PolicyBackend,acl map[string][]string,digest string)error{
  called++
  if len(routes)!=3||routes["socks"]!=nil {t.Fatalf("service route leaked: %+v",routes)}
  expected,err:=PolicyGenerationDigest(routes,acl)
  if err!=nil||expected!=digest {t.Fatal("incorrect policy generation digest")}
  return nil
 }
 if err:=SyncVerifiedRoutes(ctx,store,panel,effective,auth,apply);err!=nil {t.Fatal(err)}
 got,_,ready:=store.Snapshot()
 if !ready||called!=1||len(got)!=3||got["30000"]!="30000"||got["55535"]!="55535"||got["55636"]!="55636" {
  t.Fatalf("unexpected confirmed routes: %+v, ready=%v, applied=%d",got,ready,called)
 }
 if _,exists:=got["10811"];exists {t.Fatal("SOCKS route published")}
}

func TestRuntimePolicyCoordinatorSerializesACLAndRouteSync(t *testing.T) {
 ctx:=context.Background()
 path:=filepath.Join(t.TempDir(),"server.json")
 fp:=strings.Repeat("a",64)
 existing:=strings.Repeat("b",64)
 initial:=fmt.Sprintf(`{"mode":"server","routes":{"30000":[{"Name":"30000","Address":"127.0.0.1:30000"}]},"authorized":{"%s":["30000"]}}`,existing)
 if err:=os.WriteFile(path,[]byte(initial),0600);err!=nil{t.Fatal(err)}
 started:=make(chan struct{})
 release:=make(chan struct{})
 activator:=&HotACLActivator{
  Path:path,Services:[]string{"30000"},Digest:FullPolicyConfigDigest,
  Confirm:func(context.Context,string)error{close(started);<-release;return nil},
 }
 store:=new(ConfirmedRouteStore)
 coordinator:=&RuntimePolicyCoordinator{
  ConfigPath:path,Activator:activator,Store:store,
  Panel:func(context.Context)([]PanelInbound,error){return []PanelInbound{{Port:30000,Protocol:"vless",Network:"tcp",Enabled:true}},nil},
  Applied:func(context.Context)([]EffectiveInbound,error){return []EffectiveInbound{{Port:30000,Protocol:"vless",Network:"tcp",Service:"30000",Address:"127.0.0.1:30000"}},nil},
  Apply:func(_ context.Context,_ map[string][]PolicyBackend,acl map[string][]string,_ string)error{
   if _,ok:=acl[fp];!ok{t.Error("concurrent ACL enrollment lost")}
   if _,ok:=acl[existing];!ok{t.Error("existing ACL lost")}
   return nil
  },
 }
 activated:=make(chan error,1)
 go func(){activated<-coordinator.Activate(ctx,fp)}()
 <-started
 synced:=make(chan error,1)
 go func(){synced<-coordinator.Sync(ctx)}()
 close(release)
 if err:=<-activated;err!=nil{t.Fatal(err)}
 if err:=<-synced;err!=nil{t.Fatal(err)}
 if _,_,ready:=store.Snapshot();!ready{t.Fatal("confirmed route generation absent")}
}
