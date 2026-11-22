package xui

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const basicServerConfig = `{"mode":"server","listen":"0.0.0.0:8443","routes":{"vpn":[{"Name":"existing-vless","Address":"host.docker.internal:30000"}]},"authorized":{"0000000000000000000000000000000000000000000000000000000000000000":["vpn"]},"padding_enabled":true,"jitter_enabled":true}`
const deviceFingerprint = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestStagePerDeviceACL(t *testing.T) {
	before:=[]byte(basicServerConfig)
	out,err:=AddDeviceToACL(before,deviceFingerprint,"vpn")
	if err!=nil {t.Fatal(err)}
	if !bytes.Equal(before,[]byte(basicServerConfig)) {t.Fatal("original config modified")}
	var v struct {
		Mode string `json:"mode"`
		Authorized map[string][]string `json:"authorized"`
		PaddingEnabled bool `json:"padding_enabled"`
		JitterEnabled bool `json:"jitter_enabled"`
	}
	if err:=json.Unmarshal(out,&v);err!=nil {t.Fatal(err)}
	if len(v.Authorized)!=2 || len(v.Authorized[deviceFingerprint])!=1 || v.Authorized[deviceFingerprint][0]!="vpn" {
		t.Fatal("wrong ACL")
	}
	if !v.PaddingEnabled || !v.JitterEnabled {t.Fatal("transport flags changed")}
	if _,err:=AddDeviceToACL(out,deviceFingerprint,"vpn");err==nil {t.Fatal("accepted duplicate")}
}
func TestStageACLRejectsInvalidRequests(t *testing.T) {
	cases:=[]struct{name,conf,fp,service string}{
		{"invalid fingerprint",basicServerConfig,"broken","vpn"},
		{"uppercase fingerprint",basicServerConfig,strings.ToUpper(deviceFingerprint),"vpn"},
		{"unknown route",basicServerConfig,deviceFingerprint,"unknown"},
		{"wrong mode",strings.Replace(basicServerConfig,`"mode":"server"`,`"mode":"client"`,1),deviceFingerprint,"vpn"},
		{"invalid json","{",deviceFingerprint,"vpn"},
		{"trailing document",basicServerConfig+" {}",deviceFingerprint,"vpn"},
		{"trailing garbage",basicServerConfig+" garbage",deviceFingerprint,"vpn"},
		{"missing authorized",`{"mode":"server","routes":{"vpn":[{}]}}`,deviceFingerprint,"vpn"},
	}
	for _,tc:=range cases {
		t.Run(tc.name,func(t *testing.T){
			if _,err:=AddDeviceToACL([]byte(tc.conf),tc.fp,tc.service);err==nil {t.Fatal("accepted unsafe config/ACL")}
		})
	}
}

func TestStageDeviceForTCPAndXHTTP(t *testing.T) {
    configured := strings.Replace(basicServerConfig,
        `"vpn":[{"Name":"existing-vless","Address":"host.docker.internal:30000"}]`,
        `"vpn":[{"Name":"existing-vless","Address":"host.docker.internal:30000"}],"vpn-xhttp":[{"Name":"existing-xhttp","Address":"host.docker.internal:30443"}]`, 1)
    input := []byte(configured)
    data, err := AddDeviceToACLServices(input, deviceFingerprint, []string{"vpn", "vpn-xhttp"})
    if err != nil { t.Fatal(err) }
    if !bytes.Equal(input, []byte(configured)) { t.Fatal("server input mutated") }
    var result struct { Authorized map[string][]string `json:"authorized"`; Routes map[string]json.RawMessage `json:"routes"` }
    if err := json.Unmarshal(data, &result); err != nil { t.Fatal(err) }
    got := result.Authorized[deviceFingerprint]
    if len(got) != 2 || got[0] != "vpn" || got[1] != "vpn-xhttp" { t.Fatalf("wrong services: %v", got) }
    if len(result.Routes) != 2 { t.Fatal("routes modified") }
    for _, services := range [][]string{{}, {"vpn","vpn"}, {"vpn","missing"}, {"missing","vpn"}} {
        if _, err := AddDeviceToACLServices(input, deviceFingerprint, services); err == nil {
            t.Fatalf("invalid services accepted: %v", services)
        }
    }
}
