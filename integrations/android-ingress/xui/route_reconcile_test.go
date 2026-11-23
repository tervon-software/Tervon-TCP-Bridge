package xui

import "testing"

func TestReconcileInbounds3xUI(t *testing.T){
 panel:=[]PanelInbound{
  {Port:55535,Protocol:"vless",Network:"tcp",Enabled:true},
  {Port:55636,Protocol:"vless",Network:"xhttp",Enabled:true},
  {Port:30000,Protocol:"trojan",Network:"tcp",Enabled:false},
 }
 live:=[]EffectiveInbound{
  {Port:55535,Protocol:"vless",Network:"tcp",Service:"node-tcp",Address:"127.0.0.1:55535"},
  {Port:55636,Protocol:"vless",Network:"xhttp",Service:"node-xhttp",Address:"127.0.0.1:55636"},
  {Port:30000,Protocol:"trojan",Network:"tcp",Service:"node-trojan",Address:"127.0.0.1:30000"},
 }
 routes,err:=ReconcileInbounds(panel,live)
 if err!=nil{t.Fatal(err)}
 if len(routes)!=2 || routes[0].Port!=55535 || routes[1].Port!=55636 {t.Fatalf("unexpected routes: %+v",routes)}
}

func TestReconcileInboundsRemnawaveNodeSelection(t *testing.T){
 // The Remnawave adapter supplies ONLY inbounds enabled for this node.
 // A profile's inactive inbound must not become a live Bridge route.
 selected:=[]PanelInbound{{Port:7771,Protocol:"vless",Network:"tcp",Enabled:true}}
 live:=[]EffectiveInbound{
  {Port:7771,Protocol:"vless",Network:"tcp",Service:"cascade-entry",Address:"127.0.0.1:7771"},
  {Port:7772,Protocol:"vless",Network:"tcp",Service:"other",Address:"127.0.0.1:7772"},
 }
 routes,err:=ReconcileInbounds(selected,live)
 if err!=nil{t.Fatal(err)}
 if len(routes)!=1 || routes[0].Service!="cascade-entry"{t.Fatalf("node selection ignored: %+v",routes)}
}

func TestReconcileInboundsRejectMismatchAndAmbiguity(t *testing.T){
 panel:=[]PanelInbound{{Port:55535,Protocol:"vless",Network:"tcp",Enabled:true}}
 live:=[]EffectiveInbound{{Port:55535,Protocol:"vless",Network:"tcp",Service:"vpn",Address:"127.0.0.1:55535"}}
 cases:=[]struct{name string;p []PanelInbound;l []EffectiveInbound;wantErr bool;count int}{
  {"identical",panel,live,false,1},
  {"empty",nil,nil,false,0},
  {"wrong-protocol",[]PanelInbound{{Port:55535,Protocol:"trojan",Network:"tcp",Enabled:true}},live,false,0},
  {"wrong-network",[]PanelInbound{{Port:55535,Protocol:"vless",Network:"xhttp",Enabled:true}},live,false,0},
  {"udp",[]PanelInbound{{Port:55535,Protocol:"vless",Network:"quic",Enabled:true}},[]EffectiveInbound{{Port:55535,Protocol:"vless",Network:"quic",Service:"vpn",Address:"127.0.0.1:55535"}},false,0},
  {"duplicate-panel",append(panel,panel...),live,true,0},
  {"duplicate-effective",panel,append(live,live...),true,0},
  {"wrong-backend",panel,[]EffectiveInbound{{Port:55535,Protocol:"vless",Network:"tcp",Service:"vpn",Address:"127.0.0.1:30000"}},true,0},
  {"bad-service",panel,[]EffectiveInbound{{Port:55535,Protocol:"vless",Network:"tcp",Service:"",Address:"127.0.0.1:55535"}},true,0},
 }
 for _,tc:=range cases{t.Run(tc.name,func(t *testing.T){
  got,err:=ReconcileInbounds(tc.p,tc.l)
  if (err!=nil)!=tc.wantErr{t.Fatalf("err=%v wantErr=%v",err,tc.wantErr)}
  if !tc.wantErr&&len(got)!=tc.count{t.Fatalf("unexpected routes %+v",got)}
 })}
}

func TestReconcileInboundsVerifiedXrayTransportAliases(t *testing.T) {
 cases:=[]struct{name,panelNetwork,liveNetwork string;want int}{
  {"xhttp-to-splithttp","xhttp","splithttp",1},
  {"splithttp-to-xhttp","splithttp","xhttp",1},
  {"raw-to-tcp","raw","tcp",1},
  {"tcp-to-raw","tcp","raw",1},
  {"case-insensitive","XHTTP","SplitHTTP",1},
  {"different-transports","ws","xhttp",0},
  {"unknown-transports","quic","quic",0},
  {"unknown-vs-tcp","newtransport","tcp",0},
 }
 for _,tc:=range cases{t.Run(tc.name,func(t *testing.T){
  panel:=[]PanelInbound{{Port:44443,Protocol:"vless",Network:tc.panelNetwork,Enabled:true}}
  live:=[]EffectiveInbound{{Port:44443,Protocol:"vless",Network:tc.liveNetwork,Service:"candidate",Address:"127.0.0.1:44443"}}
  got,err:=ReconcileInbounds(panel,live)
  if err!=nil{t.Fatal(err)}
  if len(got)!=tc.want{t.Fatalf("got %d routes, want %d",len(got),tc.want)}
 })}
}

func TestReconcileInboundsAliasDoesNotAuthorizeUnselectedBackend(t *testing.T) {
 panel:=[]PanelInbound{
  {Port:44443,Protocol:"vless",Network:"xhttp",Enabled:true},
  {Port:44444,Protocol:"trojan",Network:"tcp",Enabled:false},
 }
 live:=[]EffectiveInbound{
  {Port:44443,Protocol:"vless",Network:"splithttp",Service:"selected",Address:"127.0.0.1:44443"},
  {Port:44444,Protocol:"trojan",Network:"raw",Service:"disabled",Address:"127.0.0.1:44444"},
  {Port:44445,Protocol:"socks",Network:"tcp",Service:"unselected",Address:"127.0.0.1:44445"},
 }
 got,err:=ReconcileInbounds(panel,live)
 if err!=nil{t.Fatal(err)}
 if len(got)!=1||got[0].Service!="selected"{t.Fatalf("unexpected route selection: %+v",got)}
}

func TestFilterBridgePanelInbounds(t *testing.T) {
 in:=[]PanelInbound{
  {Port:10811,Protocol:"socks",Network:"tcp",Enabled:true},
  {Port:8080,Protocol:"http",Network:"tcp",Enabled:true},
  {Port:1234,Protocol:"dokodemo-door",Network:"tcp",Enabled:true},
  {Port:30000,Protocol:"vless",Network:"raw",Enabled:true},
  {Port:55535,Protocol:"VLESS",Network:"splithttp",Enabled:true},
  {Port:55636,Protocol:"trojan",Network:"tcp",Enabled:true},
  {Port:8443,Protocol:"shadowsocks",Network:"tcp",Enabled:true},
  {Port:9999,Protocol:"unknown",Network:"tcp",Enabled:true},
 }
 got:=FilterBridgePanelInbounds(in)
 if len(got)!=4 {t.Fatalf("expected 4 VPN inbounds, got %+v",got)}
 for i,port:=range []int{30000,55535,55636,8443} {
  if got[i].Port!=port {t.Fatalf("index %d: got port %d, want %d",i,got[i].Port,port)}
 }
 // Filter does not alter provider evidence or select disabled services.
 if len(in)!=8 || in[0].Protocol!="socks" {t.Fatal("input mutated")}
}
