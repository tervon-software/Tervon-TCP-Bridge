package xui

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "net/url"
 "os"
 "regexp"
 "strings"
 "time"
)

var remnawaveUUID = regexp.MustCompile("(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

// RemnawaveClient reads node-scoped inbound selections without modifying
// the panel, Bridge, or Xray. It deliberately never logs URLs or credentials.
type RemnawaveClient struct {
 base *url.URL
 token string
 httpClient *http.Client
}

func NewRemnawaveClientFromFile(baseURL, credentialPath string) (*RemnawaveClient,error) {
 data,err:=os.ReadFile(credentialPath)
 if err!=nil{return nil,errors.New("cannot read Remnawave API credential")}
 return NewRemnawaveClient(baseURL,strings.TrimSpace(string(data)))
}

func NewRemnawaveClient(baseURL, token string) (*RemnawaveClient,error) {
 u,err:=url.Parse(baseURL)
 if err!=nil || u.Scheme!="https" || u.Host=="" || u.User!=nil || u.RawQuery!="" || u.Fragment!="" || u.RawPath!="" || u.Opaque!="" || (u.Path!=""&&u.Path!="/"){
  return nil,errors.New("Remnawave API requires an HTTPS origin")
 }
 if token=="" || strings.ContainsAny(token," \t\r\n"){return nil,errors.New("invalid Remnawave API token")}
 u.Path=""
 return &RemnawaveClient{base:u,token:token,httpClient:&http.Client{Timeout:10*time.Second,CheckRedirect:func(_ *http.Request,_ []*http.Request)error{return http.ErrUseLastResponse}}},nil
}

func(c *RemnawaveClient) get(ctx context.Context,path string,dst any)error{
 if c==nil||c.base==nil||c.httpClient==nil{return errors.New("Remnawave API client unavailable")}
 u:=*c.base
 u.Path=path
 req,err:=http.NewRequestWithContext(ctx,http.MethodGet,u.String(),nil)
 if err!=nil{return errors.New("Remnawave API request invalid")}
 req.Header.Set("Authorization","Bearer "+c.token)
 req.Header.Set("Accept","application/json")
 response,err:=c.httpClient.Do(req)
 if err!=nil{return errors.New("Remnawave API connection unavailable")}
 defer response.Body.Close()
 if response.StatusCode!=http.StatusOK{return fmt.Errorf("Remnawave API HTTP %d",response.StatusCode)}
 var envelope struct{Response json.RawMessage `json:"response"`}
 data,err:=io.ReadAll(io.LimitReader(response.Body,maxResponseBytes+1))
 if err!=nil||len(data)>maxResponseBytes{return errors.New("Remnawave API response too large or unreadable")}
 if err:=json.Unmarshal(data,&envelope);err!=nil||len(envelope.Response)==0||string(envelope.Response)=="null"{return errors.New("Remnawave API response invalid")}
 if err:=json.Unmarshal(envelope.Response,dst);err!=nil{return errors.New("Remnawave API payload schema mismatch")}
 return nil
}

type remnaInbound struct{
 UUID string `json:"uuid"`
 ProfileUUID string `json:"profileUuid"`
 Port *int `json:"port"`
 Type string `json:"type"`
 Network string `json:"network"`
}
type remnaNode struct{
 UUID string `json:"uuid"`
 IsDisabled *bool `json:"isDisabled"`
 ConfigProfile struct{
  ActiveConfigProfileUUID string `json:"activeConfigProfileUuid"`
  ActiveInbounds []remnaInbound `json:"activeInbounds"`
 } `json:"configProfile"`
}

// SelectedNodeInbounds verifies Node UUID -> assigned profile -> selected
// inbound UUIDs. It MUST still pass ReconcileInbounds against effective
// Xray before being considered a Bridge backend; no routes are published here.
func(c *RemnawaveClient) SelectedNodeInbounds(ctx context.Context,nodeUUID string)([]PanelInbound,error){
 if !remnawaveUUID.MatchString(nodeUUID){return nil,errors.New("invalid Remnawave node UUID")}
 var node remnaNode
 if err:=c.get(ctx,"/api/nodes/"+nodeUUID,&node);err!=nil{return nil,err}
 if !strings.EqualFold(node.UUID,nodeUUID)||node.IsDisabled==nil{return nil,errors.New("Remnawave node identity mismatch")}
 if *node.IsDisabled{return nil,errors.New("Remnawave node disabled")}
 profile:=node.ConfigProfile.ActiveConfigProfileUUID
 if !remnawaveUUID.MatchString(profile)||node.ConfigProfile.ActiveInbounds==nil{return nil,errors.New("Remnawave node profile unassigned")}
 if len(node.ConfigProfile.ActiveInbounds)>4096{return nil,errors.New("Remnawave selected inbound count exceeded")}
 var list []remnaInbound
 if err:=c.get(ctx,"/api/config-profiles/"+profile+"/inbounds",&list);err!=nil{return nil,err}
 if list==nil||len(list)>4096{return nil,errors.New("Remnawave profile inbound list missing")}
 byID:=make(map[string]remnaInbound,len(list))
 for _,in:=range list{
  if !remnawaveUUID.MatchString(in.UUID)||!strings.EqualFold(in.ProfileUUID,profile){return nil,errors.New("Remnawave inbound profile mismatch")}
  id:=strings.ToLower(in.UUID)
  if _,exists:=byID[id];exists{return nil,errors.New("duplicate Remnawave profile inbound UUID")}
  byID[id]=in
 }
 selected:=make([]PanelInbound,0,len(node.ConfigProfile.ActiveInbounds))
 seen:=make(map[string]bool,len(node.ConfigProfile.ActiveInbounds))
 for _,active:=range node.ConfigProfile.ActiveInbounds{
  id:=strings.ToLower(active.UUID)
  if !remnawaveUUID.MatchString(id)||seen[id]{return nil,errors.New("invalid or repeated Remnawave active inbound")}
  seen[id]=true
  in,ok:=byID[id]
  if !ok||(active.ProfileUUID!=""&&!strings.EqualFold(active.ProfileUUID,profile)){return nil,errors.New("Remnawave active inbound not in node profile")}
  if in.Port==nil||*in.Port<1||*in.Port>65535||strings.TrimSpace(in.Type)==""||strings.TrimSpace(in.Network)==""{return nil,errors.New("Remnawave selected inbound invalid")}
  if (active.Port!=nil&&*active.Port!=*in.Port)||(active.Type!=""&&!strings.EqualFold(active.Type,in.Type))||(active.Network!=""&&!strings.EqualFold(active.Network,in.Network)){return nil,errors.New("Remnawave inbound changed during read")}
  selected=append(selected,PanelInbound{Port:*in.Port,Protocol:strings.ToLower(in.Type),Network:strings.ToLower(in.Network),Enabled:true})
 }
 return selected,nil
}
