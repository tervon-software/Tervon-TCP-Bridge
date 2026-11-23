package xui

import (
 "context"
 "fmt"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "time"
)

func TestPolicyACKInLogsExactPostSignalGeneration(t *testing.T){
 d:=strings.Repeat("a",64)
 start:=time.Now().UTC()
 old:=start.Add(-time.Second).Format(time.RFC3339Nano)
 newer:=start.Add(time.Millisecond).Format(time.RFC3339Nano)
 for _,entry:=range []string{
  old+" M6_POLICY_RELOAD=PASS digest="+d,
  newer+" M6_ACL_RELOAD=PASS digest="+d,
  newer+" M6_POLICY_RELOAD=PASS digest="+strings.Repeat("b",64),
  newer+" prefix M6_POLICY_RELOAD=PASS digest="+d,
  newer+" M6_POLICY_RELOAD=PASS digest="+d+" suffix",
  "without-timestamp M6_POLICY_RELOAD=PASS digest="+d,
 }{
  if policyACKInLogs([]byte(entry),start,d){t.Fatalf("accepted wrong/stale ACK: %s",entry)}
 }
 if !policyACKInLogs([]byte(newer+" M6_POLICY_RELOAD=PASS digest="+d+"\n"),start,d){
  t.Fatal("rejected exact post-signal policy ACK")
 }
 goPrefix:="2026/11/23 02:18:34 "
 if !policyACKInLogs([]byte(newer+" "+goPrefix+"M6_POLICY_RELOAD=PASS digest="+d+"\n"),start,d){
  t.Fatal("rejected real Docker+Go timestamp format")
 }
 for _,entry:=range []string{
  old+" "+goPrefix+"M6_POLICY_RELOAD=PASS digest="+d,
  newer+" "+goPrefix+"M6_ACL_RELOAD=PASS digest="+d,
  newer+" "+goPrefix+"M6_POLICY_RELOAD=PASS digest="+strings.Repeat("b",64),
  newer+" "+goPrefix+"prefix M6_POLICY_RELOAD=PASS digest="+d,
  newer+" "+goPrefix+"M6_POLICY_RELOAD=PASS digest="+d+" suffix",
  newer+" 2026/13/23 02:18:34 M6_POLICY_RELOAD=PASS digest="+d,
  newer+" 2026/11/23 02:18:34  M6_POLICY_RELOAD=PASS digest="+d,
 } {
  if policyACKInLogs([]byte(entry),start,d){t.Fatalf("accepted invalid Docker+Go ACK: %s",entry)}
 }
}

func policyDockerFixture(t *testing.T,inspectBody,logBody string)(DockerPolicyConfirmer,string){
 t.Helper()
 root:=t.TempDir()
 bin:=filepath.Join(root,"bin")
 if err:=os.Mkdir(bin,0700);err!=nil{t.Fatal(err)}
 calls:=filepath.Join(root,"calls")
 script:=filepath.Join(bin,"docker")
 body:=fmt.Sprintf("#!/bin/sh\necho \"$1\" >> %q\ncase \"$1\" in\ninspect) printf '%%s\\n' %q ;;\nkill) echo signaled ;;\nlogs) printf '%%s\\n' %q ;;\n*) exit 2 ;;\nesac\n",calls,inspectBody,logBody)
 if err:=os.WriteFile(script,[]byte(body),0700);err!=nil{t.Fatal(err)}
 t.Setenv("PATH",bin+":"+os.Getenv("PATH"))
 return DockerPolicyConfirmer{
  ContainerID:strings.Repeat("a",64),
  HostConfigDirectory:filepath.Join(root,"config"),
  ExpectedImage:"test-only/bridge-policy:isolated",
  ExpectedLabelKey:"org.example.test.policy",
  ExpectedLabelValue:"isolated-candidate",
 },calls
}
func policyInspectJSON(conf,containerID,image,labelKey,labelValue string,rw bool)string{
 return fmt.Sprintf(`[{"Id":%q,"Config":{"Image":%q,"Labels":{%q:%q}},"State":{"Running":true},"Mounts":[{"Type":"bind","Source":%q,"Destination":"/config","RW":%t}]}]`,containerID,image,labelKey,labelValue,conf,rw)
}

func TestV102DockerPolicyConfirmerMockExactTarget(t *testing.T){
 digest:=strings.Repeat("c",64)
 stamp:=time.Now().UTC().Add(time.Second).Format(time.RFC3339Nano)
 c,calls:=policyDockerFixture(t,"ignored",stamp+" M6_POLICY_RELOAD=PASS digest="+digest)
 inspect:=policyInspectJSON(c.HostConfigDirectory,c.ContainerID,c.ExpectedImage,c.ExpectedLabelKey,c.ExpectedLabelValue,false)
 // Rewrite fixture with accurate inspect for the explicit candidate.
 c,calls=policyDockerFixture(t,inspect,stamp+" M6_POLICY_RELOAD=PASS digest="+digest)
 // The fixture's temporary config path changed, adjust inspect to match.
 inspect=policyInspectJSON(c.HostConfigDirectory,c.ContainerID,c.ExpectedImage,c.ExpectedLabelKey,c.ExpectedLabelValue,false)
 script:=filepath.Join(filepath.Dir(calls),"bin","docker")
 body:=fmt.Sprintf("#!/bin/sh\necho \"$1\" >> %q\ncase \"$1\" in\ninspect) printf '%%s\\n' %q ;;\nkill) echo signaled ;;\nlogs) printf '%%s\\n' %q ;;\n*) exit 2 ;;\nesac\n",calls,inspect,stamp+" M6_POLICY_RELOAD=PASS digest="+digest)
 if err:=os.WriteFile(script,[]byte(body),0700);err!=nil{t.Fatal(err)}
 ctx,cancel:=context.WithTimeout(context.Background(),time.Second)
 defer cancel()
 if err:=c.Confirm(ctx,digest);err!=nil{t.Fatal(err)}
 output,err:=os.ReadFile(calls)
 if err!=nil{t.Fatal(err)}
 if strings.Join(strings.Fields(string(output)),",")!="inspect,kill,logs"{t.Fatalf("unexpected operations: %s",output)}
}

func TestV102DockerPolicyConfirmerRejectsLegacyAndWrongACK(t *testing.T){
 digest:=strings.Repeat("c",64)
 c,_:=policyDockerFixture(t,"",time.Now().UTC().Add(time.Second).Format(time.RFC3339Nano)+" M6_ACL_RELOAD=PASS digest="+digest)
 inspect:=policyInspectJSON(c.HostConfigDirectory,c.ContainerID,c.ExpectedImage,c.ExpectedLabelKey,c.ExpectedLabelValue,false)
 // replace mock with accurate inspect
 script:=filepath.Join(filepath.Dir(c.HostConfigDirectory),"bin","docker")
 body:=fmt.Sprintf("#!/bin/sh\ncase \"$1\" in\ninspect) printf '%%s\\n' %q ;;\nkill) echo signaled ;;\nlogs) printf '%%s\\n' %q ;;\n*) exit 2 ;;\nesac\n",inspect,time.Now().UTC().Add(time.Second).Format(time.RFC3339Nano)+" M6_ACL_RELOAD=PASS digest="+digest)
 if err:=os.WriteFile(script,[]byte(body),0700);err!=nil{t.Fatal(err)}
 ctx,cancel:=context.WithTimeout(context.Background(),300*time.Millisecond)
 defer cancel()
 if err:=c.Confirm(ctx,digest);err==nil{t.Fatal("accepted ACL-only ACK")}

 legacy:=c;legacy.ExpectedImage="tervon-bridge-android-ingress:m6-runtime-acl"
 if err:=legacy.Confirm(context.Background(),digest);err==nil{t.Fatal("accepted legacy image")}
 invalid:=c;invalid.ContainerID="abcdef123456"
 if err:=invalid.Confirm(context.Background(),digest);err==nil{t.Fatal("accepted abbreviated ID")}
 badMount:=policyInspectJSON(c.HostConfigDirectory,c.ContainerID,c.ExpectedImage,c.ExpectedLabelKey,c.ExpectedLabelValue,true)
 if err:=os.WriteFile(script,[]byte(fmt.Sprintf("#!/bin/sh\ncase \"$1\" in\ninspect) printf '%%s\\n' %q ;;\n*) exit 9 ;;\nesac\n",badMount)),0700);err!=nil{t.Fatal(err)}
 if err:=c.Confirm(context.Background(),digest);err==nil{t.Fatal("accepted writable config bind")}
}
