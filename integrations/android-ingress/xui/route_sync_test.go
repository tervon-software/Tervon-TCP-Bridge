package xui

import (
 "context"
 "errors"
 "strings"
 "os"
 "path/filepath"
 "fmt"
 "encoding/json"
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
 if routes,_,ok:=store.Snapshot();!ok||routes["55535"]!="vpn"{t.Fatal("previously ACKed discovery lost on transient panel read error")}
 xrayFail:=func(context.Context)([]EffectiveInbound,error){return nil,errors.New("runtime query timeout")}
 if err:=SyncVerifiedRoutes(ctx,store,goodPanel,xrayFail,auth,successful);err==nil{t.Fatal("Xray query failure accepted")}
 if routes,_,ok:=store.Snapshot();!ok||routes["55535"]!="vpn"{t.Fatal("previously ACKed discovery lost on transient Xray query error")}
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

func TestSyncVerifiedRoutesExpandsAndPrunesExistingCertificateACL(t *testing.T){
 ctx:=context.Background()
 store:=new(ConfirmedRouteStore)
 fpA:=strings.Repeat("a",64)
 fpB:=strings.Repeat("b",64)
 auth:=map[string][]string{fpA:{"30000"},fpB:{"30000"}}
 ports:=[]int{30000,55535,55636}
 panel:=func(context.Context)([]PanelInbound,error){
  out:=make([]PanelInbound,0,len(ports))
  for _,port:=range ports{out=append(out,PanelInbound{Port:port,Protocol:"vless",Network:"tcp",Enabled:true})}
  return out,nil
 }
 applied:=func(context.Context)([]EffectiveInbound,error){
  out:=make([]EffectiveInbound,0,len(ports))
  for _,port:=range ports{out=append(out,EffectiveInbound{Port:port,Protocol:"vless",Network:"tcp",Service:fmt.Sprint(port),Address:fmt.Sprintf("127.0.0.1:%d",port)})}
  return out,nil
 }
 appliedCalls:=0
 apply:=func(_ context.Context,routes map[string][]PolicyBackend,acl map[string][]string,digest string)error{
  appliedCalls++
  if len(acl)!=2||len(routes)!=len(ports){t.Fatalf("unexpected policy size: %d %d",len(acl),len(routes))}
  for _,fp:=range []string{fpA,fpB}{
   if len(acl[fp])!=len(ports){t.Fatalf("missing verified port for %s: %+v",fp,acl[fp])}
   for _,port:=range ports{
    found:=false
    for _,service:=range acl[fp]{if service==fmt.Sprint(port){found=true}}
    if !found{t.Fatalf("port %d not authorized",port)}
   }
  }
  expected,err:=PolicyGenerationDigest(routes,acl)
  if err!=nil||digest!=expected{t.Fatal("invalid generation digest")}
  return nil
 }
 if err:=SyncVerifiedRoutes(ctx,store,panel,applied,auth,apply);err!=nil{t.Fatal(err)}
 if len(auth[fpA])!=1||len(auth[fpB])!=1{t.Fatal("mutated caller ACL")}
 ports=[]int{30000}
 if err:=SyncVerifiedRoutes(ctx,store,panel,applied,auth,apply);err!=nil{t.Fatal(err)}
 if appliedCalls!=2{t.Fatal("missing atomic update")}
 snapshot,_,ready:=store.Snapshot()
 if !ready||len(snapshot)!=1||snapshot["30000"]!="30000"{t.Fatalf("stale removed port: %+v",snapshot)}
}

func TestApplyCompletePolicyAtomicACKAndRollback(t *testing.T) {
 path:=filepath.Join(t.TempDir(),"server.json")
 fp:=strings.Repeat("a",64)
 original:=fmt.Sprintf(`{"mode":"server","listen":":443","routes":{"30000":[{"Name":"30000","Address":"127.0.0.1:30000"}]},"authorized":{"%s":["30000"]}}`,fp)
 if err:=os.WriteFile(path,[]byte(original),0600);err!=nil{t.Fatal(err)}
 routes:=map[string][]PolicyBackend{"55535":{{Name:"55535",Address:"127.0.0.1:55535"}}}
 acl:=map[string][]string{fp:{"55535"}}
 digest,err:=PolicyGenerationDigest(routes,acl)
 if err!=nil{t.Fatal(err)}
 calls:=0
 confirm:=func(_ context.Context,got string)error{
  calls++
  data,err:=os.ReadFile(path)
  if err!=nil{t.Fatal(err)}
  actual,err:=FullPolicyConfigDigest(data)
  if err!=nil||actual!=got{t.Fatalf("ACK digest mismatch: %v",err)}
  return nil
 }
 if err:=ApplyCompletePolicy(context.Background(),path,routes,acl,digest,confirm);err!=nil{t.Fatal(err)}
 if calls!=1{t.Fatal("missing ACK")}
 data,err:=os.ReadFile(path)
 if err!=nil||!strings.Contains(string(data),`"listen": ":443"`){t.Fatal("unrelated server fields lost")}
 prior:=append([]byte(nil),data...)
 failed:=func(_ context.Context,got string)error{
  calls++
  if got==digest{return errors.New("missing ACK")}
  return nil
 }
 nextRoutes:=map[string][]PolicyBackend{"55636":{{Name:"55636",Address:"127.0.0.1:55636"}}}
 nextACL:=map[string][]string{fp:{"55636"}}
 nextDigest,err:=PolicyGenerationDigest(nextRoutes,nextACL)
 if err!=nil{t.Fatal(err)}
 failed=func(_ context.Context,got string)error{
  calls++
  if got==nextDigest{return errors.New("missing ACK")}
  if got!=digest{t.Fatal("rollback ACK digest mismatch")}
  return nil
 }
 if err:=ApplyCompletePolicy(context.Background(),path,nextRoutes,nextACL,nextDigest,failed);err==nil{t.Fatal("unconfirmed policy accepted")}
 restored,err:=os.ReadFile(path)
 if err!=nil||string(restored)!=string(prior){t.Fatal("previous policy not restored")}
 if calls!=3{t.Fatalf("unexpected ACK calls: %d",calls)}
 if err:=ApplyCompletePolicy(context.Background(),path,nextRoutes,nextACL,digest,confirm);err==nil{t.Fatal("mismatched digest accepted")}
}

func TestRuntimePolicyCoordinatorEnrollmentUsesCurrentRoutes(t *testing.T) {
 path:=filepath.Join(t.TempDir(),"server.json")
 existing:=strings.Repeat("a",64)
 newcomer:=strings.Repeat("b",64)
 original:=fmt.Sprintf(`{"mode":"server","routes":{"30000":[{"Name":"30000","Address":"127.0.0.1:30000"}]},"authorized":{"%s":["30000"]}}`,existing)
 if err:=os.WriteFile(path,[]byte(original),0600);err!=nil{t.Fatal(err)}
 activator:=&HotACLActivator{Path:path,Services:[]string{"30000"},Digest:FullPolicyConfigDigest,
  Confirm:func(_ context.Context,digest string)error{
   data,err:=os.ReadFile(path)
   if err!=nil{return err}
   actual,err:=FullPolicyConfigDigest(data)
   if err!=nil||actual!=digest{return errors.New("wrong policy ACK")}
   return nil
  },
 }
 coordinator:=&RuntimePolicyCoordinator{ConfigPath:path,Activator:activator}
 next:=fmt.Sprintf(`{"mode":"server","routes":{"55535":[{"Name":"55535","Address":"127.0.0.1:55535"}]},"authorized":{"%s":["55535"]}}`,existing)
 if err:=os.WriteFile(path,[]byte(next),0600);err!=nil{t.Fatal(err)}
 if err:=coordinator.Activate(context.Background(),newcomer);err!=nil{t.Fatal(err)}
 data,err:=os.ReadFile(path)
 if err!=nil{t.Fatal(err)}
 raw,err:=decodeServerConfig(data)
 if err!=nil{t.Fatal(err)}
 var acl map[string][]string
 if err:=json.Unmarshal(raw["authorized"],&acl);err!=nil{t.Fatal(err)}
 if len(acl[newcomer])!=1||acl[newcomer][0]!="55535"{
  t.Fatalf("enrollment used stale service list: %v",acl[newcomer])
 }
}

func TestCoordinatorEnrollmentInvalidatesPreviousDiscoveryGeneration(t *testing.T) {
 path:=filepath.Join(t.TempDir(),"server.json")
 oldFP:=strings.Repeat("a",64)
 newFP:=strings.Repeat("b",64)
 original:=fmt.Sprintf(`{"mode":"server","routes":{"30000":[{"Name":"30000","Address":"127.0.0.1:30000"}]},"authorized":{"%s":["30000"]}}`,oldFP)
 if err:=os.WriteFile(path,[]byte(original),0600);err!=nil{t.Fatal(err)}
 store:=new(ConfirmedRouteStore)
 if err:=store.PublishAfterCoreACK(context.Background(),map[string]string{"30000":"30000"},strings.Repeat("a",64),func(context.Context,string)error{return nil});err!=nil{t.Fatal(err)}
 if _,_,ok:=store.Snapshot();!ok{t.Fatal("initial generation missing")}
 activator:=&HotACLActivator{Path:path,Digest:FullPolicyConfigDigest,
  Confirm:func(_ context.Context,digest string)error{
   data,err:=os.ReadFile(path)
   if err!=nil{return err}
   actual,err:=FullPolicyConfigDigest(data)
   if err!=nil||actual!=digest{return errors.New("ACK mismatch")}
   return nil
  },
 }
 coordinator:=&RuntimePolicyCoordinator{ConfigPath:path,Activator:activator,Store:store}
 if err:=coordinator.Activate(context.Background(),newFP);err!=nil{t.Fatal(err)}
 if _,_,ok:=store.Snapshot();ok{t.Fatal("stale discovery generation remained after ACL enrollment")}
 data,err:=os.ReadFile(path)
 if err!=nil{t.Fatal(err)}
 raw,err:=decodeServerConfig(data)
 if err!=nil{t.Fatal(err)}
 var acl map[string][]string
 if err:=json.Unmarshal(raw["authorized"],&acl);err!=nil{t.Fatal(err)}
 if len(acl[newFP])!=1||acl[newFP][0]!="30000"{t.Fatal("new fingerprint not authorized")}
}
