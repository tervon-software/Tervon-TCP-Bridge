package xui

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestHotACLActivatorPersistsOnlyACLAndConfirms(t *testing.T) {
	root:=t.TempDir()
	path:=filepath.Join(root,"server.json")
	if err:=os.WriteFile(path,[]byte(basicServerConfig),0640);err!=nil{t.Fatal(err)}
	var observed []string
	a:=&HotACLActivator{
		Path:path,Service:"vpn",
		Confirm:func(_ context.Context,digest string)error{
			data,err:=os.ReadFile(path)
			if err!=nil{return err}
			actual,err:=aclDigest(data)
			if err!=nil{return err}
			if actual!=digest{return errors.New("not the active ACL digest")}
			observed=append(observed,digest)
			return nil
		},
	}
	if err:=a.Activate(context.Background(),deviceFingerprint);err!=nil{t.Fatal(err)}
	if len(observed)!=1 || len(observed[0])!=64{t.Fatal("no exact-generation confirmation")}
	result,err:=os.ReadFile(path);if err!=nil{t.Fatal(err)}
	if !bytes.Contains(result,[]byte(deviceFingerprint)){t.Fatal("device not persisted")}
	if !bytes.Contains(result,[]byte(`"jitter_enabled": true`)){
		t.Fatal("unrelated transport configuration changed")
	}
	fi,err:=os.Stat(path);if err!=nil{t.Fatal(err)}
	if fi.Mode().Perm()!=0640{t.Fatalf("permissions changed: %o",fi.Mode().Perm())}
	if err:=a.Activate(context.Background(),deviceFingerprint);err==nil{
		t.Fatal("duplicate device unexpectedly accepted")
	}
}

func TestHotACLActivatorFailsClosedAndRollsBack(t *testing.T){
	root:=t.TempDir()
	path:=filepath.Join(root,"server.json")
	original:=[]byte(basicServerConfig)
	if err:=os.WriteFile(path,original,0600);err!=nil{t.Fatal(err)}
	var n int
	a:=&HotACLActivator{Path:path,Service:"vpn",Confirm:func(_ context.Context,digest string)error{
		n++
		if n==1{return errors.New("failed to apply")}
		actual,err:=aclDigest(original)
		if err!=nil{return err}
		if actual!=digest{return errors.New("bad rollback digest")}
		return nil
	}}
	err:=a.Activate(context.Background(),deviceFingerprint)
	if err==nil || !strings.Contains(err.Error(),"previous ACL restored"){
		t.Fatalf("missing failure and rollback: %v",err)
	}
	got,err:=os.ReadFile(path);if err!=nil{t.Fatal(err)}
	if !bytes.Equal(got,original){t.Fatal("persistent ACL was not restored")}
	if n!=2{t.Fatalf("unexpected signal attempts: %d",n)}
}

func TestHotACLActivatorSerializesConcurrentEnrollments(t *testing.T){
	root:=t.TempDir()
	path:=filepath.Join(root,"server.json")
	if err:=os.WriteFile(path,[]byte(basicServerConfig),0600);err!=nil{t.Fatal(err)}
	a:=&HotACLActivator{Path:path,Service:"vpn",Confirm:func(_ context.Context,digest string)error{
		current,err:=os.ReadFile(path)
		if err!=nil{return err}
		actual,err:=aclDigest(current)
		if err!=nil{return err}
		if actual!=digest{return errors.New("stale digest")}
		return nil
	}}
	fps:=[]string{deviceFingerprint,strings.Repeat("b",64),strings.Repeat("c",64)}
	var wg sync.WaitGroup
	errs:=make(chan error,len(fps))
	for _,fp:=range fps {
		wg.Add(1);go func(fp string){defer wg.Done();errs<-a.Activate(context.Background(),fp)}(fp)
	}
	wg.Wait();close(errs)
	for err:=range errs{if err!=nil{t.Fatal(err)}}
	data,err:=os.ReadFile(path);if err!=nil{t.Fatal(err)}
	for _,fp:=range fps{
		if !bytes.Contains(data,[]byte(fp)){t.Fatalf("lost enrollment %s",fp)}
	}
}
