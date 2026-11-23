package xui

import (
 "context"
 "encoding/json"
 "errors"
 "path/filepath"
 "regexp"
 "strings"
 "time"
)

// DockerPolicyConfirmer verifies only an explicitly provisioned v1.0.2
// candidate. Image/label/identity come from the approved deployment manifest,
// not defaults chosen by the controller. No production Docker is referenced.
type DockerPolicyConfirmer struct {
 ContainerID string
 HostConfigDirectory string
 ExpectedImage string
 ExpectedLabelKey string
 ExpectedLabelValue string
}

var fullDockerIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var dockerLabelKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,127}$`)

func(d DockerPolicyConfirmer) verifyIdentity(ctx context.Context)error{
 if !fullDockerIDPattern.MatchString(d.ContainerID) ||
  !filepath.IsAbs(d.HostConfigDirectory) ||
  filepath.Clean(d.HostConfigDirectory)!=d.HostConfigDirectory ||
  d.HostConfigDirectory=="/" ||
  d.ExpectedImage==""||strings.ContainsAny(d.ExpectedImage," \t\r\n")||
  d.ExpectedImage=="tervon-bridge-android-ingress:m6-runtime-acl"||
  !dockerLabelKeyPattern.MatchString(d.ExpectedLabelKey)||
  d.ExpectedLabelValue==""||strings.ContainsAny(d.ExpectedLabelValue,"\r\n"){
  return errors.New("v1.0.2 candidate identity not configured")
 }
 output,err:=dockerCommand(ctx,"inspect",d.ContainerID)
 if err!=nil{return err}
 var states []struct{
  ID string `json:"Id"`
  Config struct{
   Image string `json:"Image"`
   Labels map[string]string `json:"Labels"`
  } `json:"Config"`
  State struct{Running bool `json:"Running"`} `json:"State"`
  Mounts []struct{
   Type string `json:"Type"`
   Source string `json:"Source"`
   Destination string `json:"Destination"`
   RW bool `json:"RW"`
  } `json:"Mounts"`
 }
 if err:=json.Unmarshal(output,&states);err!=nil||len(states)!=1{
  return errors.New("v1.0.2 candidate inspect invalid")
 }
 target:=states[0]
 if target.ID!=d.ContainerID||!target.State.Running||
  target.Config.Image!=d.ExpectedImage||
  target.Config.Labels[d.ExpectedLabelKey]!=d.ExpectedLabelValue{
  return errors.New("v1.0.2 candidate container identity mismatch")
 }
 for _,m:=range target.Mounts{
  if m.Type=="bind"&&m.Source==d.HostConfigDirectory&&m.Destination=="/config"&&!m.RW{return nil}
 }
 return errors.New("v1.0.2 candidate configuration mount mismatch")
}

// policyACKInLogs requires the exact post-signal Docker timestamp and the
// precise policy-generation ACK line. ACL-only and stale entries are rejected.
func policyACKInLogs(output []byte,after time.Time,digest string)bool{
 expected:="M6_POLICY_RELOAD=PASS digest="+digest
 for _,line:=range strings.Split(string(output),"\n"){
  stamp,entry,ok:=strings.Cut(line," ")
  if !ok{continue}
  at,err:=time.Parse(time.RFC3339Nano,stamp)
  if err!=nil||at.Before(after){continue}
  message:=strings.TrimSpace(entry)
  if message==expected{return true}
  // Bridge emits the same ACK with the standard Go log prefix after Docker's
  // own RFC3339 timestamp. Accept only this exact timestamp format and exact
  // ACK payload; never accept substrings or stale Docker timestamps.
  if len(message)>20 && message[19]==' ' {
   if _,parseErr:=time.Parse("2006/01/02 15:04:05",message[:19]);parseErr==nil &&
    message[20:]==expected{return true}
  }
 }
 return false
}

// Confirm targets only the prevalidated candidate by its full container ID.
// SIGHUP does not restart services. A successful Docker signal alone is
// insufficient: exact post-signal policy ACK must appear in timestamped logs.
func(d DockerPolicyConfirmer) Confirm(ctx context.Context,digest string)error{
 if ctx==nil||!policyDigestPattern.MatchString(digest){return errors.New("invalid policy confirmation request")}
 if err:=d.verifyIdentity(ctx);err!=nil{return err}
 issuedAt:=time.Now().UTC()
 if _,err:=dockerCommand(ctx,"kill","--signal=HUP",d.ContainerID);err!=nil{return err}
 ticker:=time.NewTicker(125*time.Millisecond)
 defer ticker.Stop()
 since:=issuedAt.Format(time.RFC3339Nano)
 for{
  output,err:=dockerCommand(ctx,"logs","--timestamps","--since",since,d.ContainerID)
  if err!=nil{return err}
  if policyACKInLogs(output,issuedAt,digest){return nil}
  select{
  case <-ctx.Done():return errors.New("v1.0.2 policy generation acknowledgment unavailable")
  case <-ticker.C:
  }
 }
}
