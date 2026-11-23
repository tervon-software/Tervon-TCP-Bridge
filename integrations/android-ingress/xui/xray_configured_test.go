package xui

import (
 "errors"
 "fmt"
 "testing"
)

func xrayTestResolver(port int)(string,string,error){
 return fmt.Sprintf("svc-%d",port),fmt.Sprintf("127.0.0.1:%d",port),nil
}
func TestParseXrayConfiguredInbounds(t *testing.T){
 cfg:=[]byte(`{"inbounds":[{"protocol":"vless","port":55535,"streamSettings":{"network":"tcp","security":"tls"}},{"protocol":"trojan","port":30000,"streamSettings":{"network":"xhttp","security":"tls"}},{"protocol":"vless","port":44443,"streamSettings":{"network":"quic"}}]}`)
 out,err:=ParseXrayConfiguredInbounds(cfg,xrayTestResolver)
 if err!=nil{t.Fatal(err)}
 if len(out)!=2||out[0].Port!=55535||out[1].Network!="xhttp"{t.Fatalf("invalid candidates %+v",out)}
 routes,err:=ReconcileInbounds([]PanelInbound{{Port:55535,Protocol:"vless",Network:"tcp",Enabled:true},{Port:30000,Protocol:"trojan",Network:"xhttp",Enabled:true}},out)
 if err!=nil||len(routes)!=2{t.Fatalf("reconciliation failed: %v %+v",err,routes)}
}

func TestParseXrayConfiguredInboundsNegative(t *testing.T){
 for _,tc:=range []struct{name,body string}{
  {"duplicate",`{"inbounds":[{"port":55535,"protocol":"vless"},{"port":55535,"protocol":"trojan"}]}`},
  {"no-port",`{"inbounds":[{"protocol":"vless"}]}`},
  {"bad-port",`{"inbounds":[{"port":65536,"protocol":"vless"}]}`},
  {"second-document",`{"inbounds":[]} {"inbounds":[]}`},
  {"invalid-json",`{"inbounds":[`},
 } {t.Run(tc.name,func(t *testing.T){
  if _,err:=ParseXrayConfiguredInbounds([]byte(tc.body),xrayTestResolver);err==nil{t.Fatal("invalid Xray config accepted")}
 })}
 if _,err:=ParseXrayConfiguredInbounds([]byte(`{"inbounds":[]}`),nil);err==nil{t.Fatal("missing trusted resolver accepted")}
 if _,err:=ParseXrayConfiguredInbounds([]byte(`{"inbounds":[{"protocol":"vless","port":55535}]}`),func(int)(string,string,error){return "","",errors.New("not trusted")});err==nil{t.Fatal("untrusted resolution accepted")}
 if _,err:=ParseXrayConfiguredInbounds([]byte(`{"inbounds":[{"protocol":"vless","port":55535}]}`),func(int)(string,string,error){return "vpn","127.0.0.1:10000",nil});err==nil{t.Fatal("backend port mismatch accepted")}
}

func TestParseXrayConfiguredInboundsServiceListenersDoNotBlockVPN(t *testing.T){
 cfg:=[]byte(`{"inbounds":[{"protocol":"socks","port":10811,"streamSettings":{"network":"tcp"}},{"protocol":"http","port":8080,"streamSettings":{"network":"tcp"}},{"protocol":"vless","port":30000,"streamSettings":{"network":"tcp"}},{"protocol":"vless","port":55535,"streamSettings":{"network":"splithttp"}},{"protocol":"trojan","port":55636,"streamSettings":{"network":"tcp"}}]}`)
 count:=0
 out,err:=ParseXrayConfiguredInbounds(cfg,func(port int)(string,string,error){
  count++
  if port!=30000&&port!=55535&&port!=55636{return "","",errors.New("service inbound reached resolver")}
  return fmt.Sprintf("%d",port),fmt.Sprintf("127.0.0.1:%d",port),nil
 })
 if err!=nil{t.Fatal(err)}
 if count!=3||len(out)!=3{t.Fatalf("expected exactly 3 VPN TCP backends, count=%d routes=%+v",count,out)}
}
