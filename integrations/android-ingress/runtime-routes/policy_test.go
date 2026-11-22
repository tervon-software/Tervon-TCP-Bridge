package bridge

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"testing"
)

func policyFixture() (*RuntimePolicy, [sha256.Size]byte, map[string][]string, map[string][]Backend) {
	id := sha256.Sum256([]byte("v102-fixture"))
	acl := map[string][]string{hex.EncodeToString(id[:]): {"vpn"}}
	routes := map[string][]Backend{"vpn": {{Name: "old", Address: "127.0.0.1:30000"}}}
	p, err := NewRuntimePolicy(acl, routes)
	if err != nil { panic(err) }
	return p,id,acl,routes
}

func TestV102ReplaceAddChangeRemove(t *testing.T) {
	p,id,acl,routes := policyFixture()
	old,ok:=p.Select(id,"vpn")
	if !ok || old[0].Address!="127.0.0.1:30000" { t.Fatal("initial snapshot wrong") }
	routes["vpn"][0].Address="127.0.0.1:1"
	if list,_:=p.Select(id,"vpn");list[0].Address!="127.0.0.1:30000"{t.Fatal("caller mutated policy")}
	routes2:=map[string][]Backend{"vpn":{{Name:"new",Address:"127.0.0.1:55535"}},"xhttp":{{Name:"new2",Address:"127.0.0.1:55636"}}}
	acl2:=map[string][]string{hex.EncodeToString(id[:]):{"vpn","xhttp"}}
	if err:=p.Replace(acl2,routes2);err!=nil{t.Fatal(err)}
	if list,ok:=p.Select(id,"xhttp");!ok||list[0].Address!="127.0.0.1:55636"{t.Fatal("new backend unavailable")}
	if list,ok:=p.Select(id,"vpn");!ok||list[0].Address!="127.0.0.1:55535"{t.Fatal("port not replaced")}
	if err:=p.Replace(acl2,map[string][]Backend{"vpn":{{Address:"127.0.0.1:0"}}});err==nil{t.Fatal("orphan ACL accepted")}
	if list,_:=p.Select(id,"xhttp");list[0].Address!="127.0.0.1:55636"{t.Fatal("failed update changed policy")}
	if err:=p.Replace(acl,map[string][]Backend{"vpn":{{Address:"127.0.0.1:30000"}}});err!=nil{t.Fatal(err)}
	if _,ok:=p.Select(id,"xhttp");ok{t.Fatal("deleted service still authorized")}
}

func TestV102ConcurrentAtomicSnapshots(t *testing.T){
	p,id,_,_:=policyFixture()
	a:=map[string][]string{hex.EncodeToString(id[:]):{"vpn"}}
	old:=map[string][]Backend{"vpn":{{Address:"127.0.0.1:30000"}}}
	newRoute:=map[string][]Backend{"vpn":{{Address:"127.0.0.1:55535"}}}
	var wg sync.WaitGroup
	for j:=0;j<12;j++{
		wg.Add(1)
		go func(){
			defer wg.Done()
			for i:=0;i<2500;i++{
				v,ok:=p.Select(id,"vpn")
				if !ok||len(v)!=1||(v[0].Address!="127.0.0.1:30000"&&v[0].Address!="127.0.0.1:55535"){
					t.Error("partial snapshot observed");return
				}
				v[0].Address="attacker"
			}
		}()
	}
	wg.Add(1)
	go func(){defer wg.Done();for i:=0;i<500;i++{if e:=p.Replace(a,newRoute);e!=nil{t.Error(e);return};if e:=p.Replace(a,old);e!=nil{t.Error(e);return}}}()
	wg.Wait()
}

func TestV102RejectMalformedPolicy(t *testing.T){
	p,id,a,_:=policyFixture()
	for _,routes:=range []map[string][]Backend{
		{"vpn":{{Address:"bad"}}},
		{"vpn":{}},
		{"vpn":{{Address:""}}},
	}{
		if err:=p.Replace(a,routes);err==nil{t.Fatal("accepted invalid route")}
	}
	if v,ok:=p.Select(id,"vpn");!ok||v[0].Address!="127.0.0.1:30000"{t.Fatal("invalid candidate changed policy")}
}
