//go:build linux

package main

import (
 "bytes"
 "crypto/ecdsa"
 "crypto/elliptic"
 "crypto/rand"
 "crypto/sha256"
 "crypto/x509"
 "crypto/x509/pkix"
 "encoding/json"
 "encoding/pem"
 "fmt"
 "math/big"
 "net"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "sync"
 "syscall"
 "testing"
 "time"
)

type policyACKBuffer struct {sync.Mutex; bytes.Buffer}
func(b *policyACKBuffer)Write(p []byte)(int,error){b.Lock();defer b.Unlock();return b.Buffer.Write(p)}
func(b *policyACKBuffer)String()string{b.Lock();defer b.Unlock();return b.Buffer.String()}

// Runs only an isolated process built from the staged v1.0.2 sources.
// It never signals, discovers or modifies a running service.
func TestV102RealProcessSIGHUPPolicyGenerationACK(t *testing.T){
 if testing.Short(){t.Skip("isolated subprocess integration test")}
 dir:=t.TempDir()
 key,err:=ecdsa.GenerateKey(elliptic.P256(),rand.Reader)
 if err!=nil{t.Fatal(err)}
 serial,err:=rand.Int(rand.Reader,new(big.Int).Lsh(big.NewInt(1),128))
 if err!=nil{t.Fatal(err)}
 template:=&x509.Certificate{SerialNumber:serial,Subject:pkix.Name{CommonName:"local-policy-ack-test"},
  NotBefore:time.Now().Add(-time.Minute),NotAfter:time.Now().Add(time.Hour),
  IsCA:true,BasicConstraintsValid:true,KeyUsage:x509.KeyUsageCertSign|x509.KeyUsageDigitalSignature,
  ExtKeyUsage:[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
 der,err:=x509.CreateCertificate(rand.Reader,template,template,&key.PublicKey,key)
 if err!=nil{t.Fatal(err)}
 keyDER,err:=x509.MarshalECPrivateKey(key)
 if err!=nil{t.Fatal(err)}
 certPath:=filepath.Join(dir,"server.pem")
 keyPath:=filepath.Join(dir,"server.key")
 caPath:=filepath.Join(dir,"ca.pem")
 certPEM:=pem.EncodeToMemory(&pem.Block{Type:"CERTIFICATE",Bytes:der})
 for path,data:=range map[string][]byte{
  certPath:certPEM,caPath:certPEM,
  keyPath:pem.EncodeToMemory(&pem.Block{Type:"EC PRIVATE KEY",Bytes:keyDER}),
 }{if err:=os.WriteFile(path,data,0600);err!=nil{t.Fatal(err)}}
 ln,err:=net.Listen("tcp","127.0.0.1:0")
 if err!=nil{t.Fatal(err)}
 address:=ln.Addr().String()
 ln.Close()
 configPath:=filepath.Join(dir,"server.json")
 routesA:=map[string][]map[string]string{"vpn":{{"Name":"initial","Address":"127.0.0.1:11111"}}}
 routesB:=map[string][]map[string]string{"vpn":{{"Name":"replaced","Address":"127.0.0.1:22222"}}}
 auth:=map[string][]string{strings.Repeat("a",64):{"vpn"}}
 makeConfig:=func(routes map[string][]map[string]string,streams int)map[string]any{
  return map[string]any{"mode":"server","listen":address,"cert_file":certPath,"key_file":keyPath,
   "ca_file":caPath,"max_streams":streams,"routes":routes,"authorized":auth}
 }
 writeConfig:=func(c map[string]any){
  t.Helper()
  data,err:=json.Marshal(c)
  if err!=nil{t.Fatal(err)}
  temp:=configPath+".new"
  if err:=os.WriteFile(temp,data,0600);err!=nil{t.Fatal(err)}
  if err:=os.Rename(temp,configPath);err!=nil{t.Fatal(err)}
 }
 digest:=func(routes map[string][]map[string]string)string{
  t.Helper()
  data,err:=json.Marshal(struct{
   Routes map[string][]map[string]string `json:"routes"`
   Authorized map[string][]string `json:"authorized"`
  }{routes,auth})
  if err!=nil{t.Fatal(err)}
  return fmt.Sprintf("%x",sha256.Sum256(data))
 }
 writeConfig(makeConfig(routesA,3))
 bin:=filepath.Join(dir,"isolated-bridge")
 build:=exec.Command("go","build","-o",bin,".")
 if output,err:=build.CombinedOutput();err!=nil{t.Fatalf("isolated build failed: %v: %s",err,output)}
 cmd:=exec.Command(bin,"-config",configPath)
 var logs policyACKBuffer
 cmd.Stdout=&logs;cmd.Stderr=&logs
 if err:=cmd.Start();err!=nil{t.Fatal(err)}
 defer func(){_ = cmd.Process.Kill();_ = cmd.Wait()}()
 until:=time.Now().Add(5*time.Second)
 for{
  conn,e:=net.DialTimeout("tcp",address,100*time.Millisecond)
  if e==nil{conn.Close();break}
  if time.Now().After(until){t.Fatalf("isolated server not listening: %s",logs.String())}
  time.Sleep(20*time.Millisecond)
 }
 waitFor:=func(marker string){
  t.Helper()
  until:=time.Now().Add(5*time.Second)
  for !strings.Contains(logs.String(),marker){
   if time.Now().After(until){t.Fatalf("missing %q: %s",marker,logs.String())}
   time.Sleep(20*time.Millisecond)
  }
 }
 writeConfig(makeConfig(routesB,3))
 if err:=cmd.Process.Signal(syscall.SIGHUP);err!=nil{t.Fatal(err)}
 waitFor("M6_POLICY_RELOAD=PASS digest="+digest(routesB))
 count:=strings.Count(logs.String(),"M6_POLICY_RELOAD=PASS digest=")
 writeConfig(makeConfig(routesA,4))
 if err:=cmd.Process.Signal(syscall.SIGHUP);err!=nil{t.Fatal(err)}
 waitFor("M6_POLICY_RELOAD=FAIL fixed config changed")
 if got:=strings.Count(logs.String(),"M6_POLICY_RELOAD=PASS digest=");got!=count{
  t.Fatalf("invalid fixed config acknowledged: %d != %d",got,count)
 }
 writeConfig(makeConfig(routesA,3))
 if err:=cmd.Process.Signal(syscall.SIGHUP);err!=nil{t.Fatal(err)}
 waitFor("M6_POLICY_RELOAD=PASS digest="+digest(routesA))
}
