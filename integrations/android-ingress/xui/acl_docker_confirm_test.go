package xui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestM6DockerConfirmExactGenerationWithoutRestart(t *testing.T) {
	home:=t.TempDir()
	bin:=filepath.Join(home,"bin")
	if err:=os.Mkdir(bin,0700);err!=nil{t.Fatal(err)}
	conf:=filepath.Join(home,"bridge-config")
	digest:=strings.Repeat("b",64)
	id:="abcdef123456"
	calls:=filepath.Join(home,"calls")
	script:=filepath.Join(bin,"docker")
	body:="#!/bin/sh\n"+
		"echo \"$1\" >> '"+calls+"'\n"+
		"case \"$1\" in\n"+
		"inspect) printf '%s\\n' '[{\"Config\":{\"Image\":\"tervon-bridge-android-ingress:m6-runtime-acl\",\"Labels\":{\"org.tervon.bridge.hot_acl\":\"m6-v1\"}},\"State\":{\"Running\":true},\"Mounts\":[{\"Type\":\"bind\",\"Source\":\""+conf+"\",\"Destination\":\"/config\",\"RW\":false}]}]' ;;\n"+
		"kill) printf 'signaled\\n' ;;\n"+
		"logs) printf '2026-11-21 M6_ACL_RELOAD=PASS digest="+digest+"\\n' ;;\n"+
		"*) exit 9 ;;\n"+
		"esac\n"
	if err:=os.WriteFile(script,[]byte(body),0700);err!=nil{t.Fatal(err)}
	t.Setenv("PATH",bin+":"+os.Getenv("PATH"))
	confirmer:=DockerACLConfirmer{ContainerID:id,HostConfigDirectory:conf}
	ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second)
	defer cancel()
	if err:=confirmer.Confirm(ctx,digest);err!=nil{t.Fatal(err)}
	got,err:=os.ReadFile(calls);if err!=nil{t.Fatal(err)}
	lines:=strings.Fields(string(got))
	if len(lines)!=3 || lines[0]!="inspect" || lines[1]!="kill" || lines[2]!="logs"{
		t.Fatalf("unexpected Docker command order: %q",got)
	}
	for _,op:=range lines {
		if op=="restart" || op=="stop" {t.Fatal("an existing Bridge was restarted")}
	}
}

func TestM6DockerConfirmRejectsInvalidTarget(t *testing.T){
	c:=DockerACLConfirmer{ContainerID:"not-a-container-id",HostConfigDirectory:"/config"}
	if err:=c.Confirm(context.Background(),strings.Repeat("0",64));err==nil{
		t.Fatal("accepted invalid container identity")
	}
	if err:=c.Confirm(context.Background(),"bad-digest");err==nil{
		t.Fatal("accepted invalid ACL digest")
	}
}
