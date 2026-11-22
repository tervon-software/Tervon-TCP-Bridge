package xui

import (
 "context"
 "errors"
 "strings"
 "sync"
 "testing"
)

func TestConfirmedRouteStoreRequiresCoreACK(t *testing.T){
 store:=new(ConfirmedRouteStore)
 if _,_,ok:=store.Snapshot();ok{t.Fatal("unconfirmed routes advertised")}
 d1:=strings.Repeat("a",64)
 routes:=map[string]string{"55535":"vpn-tcp"}
 called:=0
 if err:=store.PublishAfterCoreACK(context.Background(),routes,d1,func(_ context.Context,d string)error{
  called++
  if d!=d1{t.Fatal("wrong digest")}
  return nil
 });err!=nil{t.Fatal(err)}
 if called!=1{t.Fatal("missing core ACK")}
 routes["55535"]="untrusted-mutation"
 got,digest,ok:=store.Snapshot()
 if !ok||digest!=d1||got["55535"]!="vpn-tcp"{t.Fatalf("snapshot not isolated: %v %v %v",got,digest,ok)}
 got["55535"]="mutate-returned-map"
 again,_,_:=store.Snapshot()
 if again["55535"]!="vpn-tcp"{t.Fatal("snapshot exposed internal map")}
}

func TestConfirmedRouteStoreRejectsBadGenerationWithoutMutation(t *testing.T){
 store:=new(ConfirmedRouteStore)
 d1:=strings.Repeat("a",64)
 if err:=store.PublishAfterCoreACK(context.Background(),map[string]string{"30000":"vpn"},d1,func(context.Context,string)error{return nil});err!=nil{t.Fatal(err)}
 for _,caseRoutes:=range []map[string]string{{},{"0":"vpn"},{"055535":"vpn"},{"65536":"vpn"},{"55535":"bad space"}}{
  if err:=store.PublishAfterCoreACK(context.Background(),caseRoutes,strings.Repeat("b",64),func(context.Context,string)error{t.Fatal("called ACK on invalid snapshot");return nil});err==nil{t.Fatal("invalid update accepted")}
 }
 if err:=store.PublishAfterCoreACK(context.Background(),map[string]string{"55535":"vpn-tcp"},"bad-digest",func(context.Context,string)error{return nil});err==nil{t.Fatal("bad digest accepted")}
 if err:=store.PublishAfterCoreACK(context.Background(),map[string]string{"55535":"vpn-tcp"},strings.Repeat("b",64),func(context.Context,string)error{return errors.New("core refused")});err==nil{t.Fatal("missing core ACK accepted")}
 got,d,ok:=store.Snapshot()
 if !ok||d!=d1||len(got)!=1||got["30000"]!="vpn"{t.Fatalf("previous acknowledged snapshot lost: %v %s %v",got,d,ok)}
}

func TestConfirmedRouteStoreConcurrentReaders(t *testing.T){
 store:=new(ConfirmedRouteStore)
 d1:=strings.Repeat("a",64)
 if err:=store.PublishAfterCoreACK(context.Background(),map[string]string{"30000":"vpn"},d1,func(context.Context,string)error{return nil});err!=nil{t.Fatal(err)}
 var wg sync.WaitGroup
 for i:=0;i<12;i++{wg.Add(1);go func(){
  defer wg.Done()
  for j:=0;j<100;j++{
   routes,d,ok:=store.Snapshot()
   if !ok||len(routes)!=1||routes["30000"]!="vpn"&&routes["55535"]!="vpn-tcp"||d!=d1&&d!=strings.Repeat("b",64){t.Error("inconsistent snapshot");return}
  }
 }()}
 if err:=store.PublishAfterCoreACK(context.Background(),map[string]string{"55535":"vpn-tcp"},strings.Repeat("b",64),func(context.Context,string)error{return nil});err!=nil{t.Fatal(err)}
 wg.Wait()
}
