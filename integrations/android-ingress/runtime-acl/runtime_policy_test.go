package bridge

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "sync"
    "testing"
)

func policyFixture() (map[string][]Backend, map[string][]string, [32]byte) {
    id := sha256.Sum256([]byte("test-device"))
    routes := map[string][]Backend{"vpn": {{Name: "first", Address: "127.0.0.1:30000"}}}
    acl := map[string][]string{hex.EncodeToString(id[:]): {"vpn"}}
    return routes, acl, id
}

func TestM6FullPolicyReplaceAndRollback(t *testing.T) {
    routes, acl, id := policyFixture()
    p, err := NewRuntimePolicy(routes, acl)
    if err != nil { t.Fatal(err) }
    old := p.Digest()
    candidates, allowed := p.Lookup(id, "vpn")
    if !allowed || len(candidates) != 1 || candidates[0].Address != "127.0.0.1:30000" {
        t.Fatal("initial snapshot mismatch")
    }
    updated := map[string][]Backend{"vpn": {{Name: "second", Address: "127.0.0.1:30001"}}}
    expected, err := p.Replace(updated, acl)
    if err != nil || expected == old { t.Fatal("route replacement failed", err) }
    candidates, allowed = p.Lookup(id, "vpn")
    if !allowed || candidates[0].Address != "127.0.0.1:30001" {
        t.Fatal("old backend selected after policy replacement")
    }
    if _, err := p.Replace(map[string][]Backend{"different": {{Address:"127.0.0.1:30002"}}}, acl); err == nil {
        t.Fatal("accepted ACL service missing from new routes")
    }
    if p.Digest() != expected {
        t.Fatal("rejected policy changed current generation")
    }
    candidates, allowed = p.Lookup(id, "vpn")
    if !allowed || candidates[0].Address != "127.0.0.1:30001" {
        t.Fatal("rollback did not preserve previous snapshot")
    }
    if _, err := p.Replace(updated, map[string][]string{"bad-fingerprint":{"vpn"}}); err == nil {
        t.Fatal("accepted malformed ACL")
    }
    if p.Digest() != expected { t.Fatal("invalid ACL changed published generation") }
    // A previously admitted stream retains its already selected backend.
    if routes["vpn"][0].Address != "127.0.0.1:30000" {
        t.Fatal("policy replacement mutated an existing route")
    }
}

func TestM6FullPolicyDigestMatchesCoordinatorContract(t *testing.T) {
    routes, acl, id := policyFixture()
    p, err := NewRuntimePolicy(routes, acl)
    if err != nil { t.Fatal(err) }
    payload, err := json.Marshal(struct {
        Routes map[string][]Backend `json:"routes"`
        Authorized map[string][]string `json:"authorized"`
    }{routes,acl})
    if err != nil { t.Fatal(err) }
    expected := fmt.Sprintf("%x", sha256.Sum256(payload))
    if p.Digest() != expected { t.Fatal("core digest does not match controller JSON") }
    // Copy isolation: caller mutations cannot rewrite the published snapshot.
    routes["vpn"][0].Address = "127.0.0.1:1"
    acl[hex.EncodeToString(id[:])][0] = "changed"
    got, allowed := p.Lookup(id, "vpn")
    if !allowed || got[0].Address != "127.0.0.1:30000" || p.Digest() != expected {
        t.Fatal("caller mutated published generation")
    }
}

func TestM6FullPolicyConcurrentReadAndSwap(t *testing.T) {
    routes, acl, id := policyFixture()
    p, err := NewRuntimePolicy(routes, acl)
    if err != nil { t.Fatal(err) }
    next := map[string][]Backend{"vpn": {{Address:"127.0.0.1:30001"}}}
    var wg sync.WaitGroup
    for i:=0; i<16; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j:=0; j<2000; j++ {
                list, ok := p.Lookup(id, "vpn")
                if !ok || len(list)!=1 ||
                    (list[0].Address!="127.0.0.1:30000" && list[0].Address!="127.0.0.1:30001") {
                    t.Error("mixed or missing policy snapshot")
                    return
                }
            }
        }()
    }
    wg.Add(1)
    go func(){
        defer wg.Done()
        for j:=0; j<300; j++ {
            if _,e:=p.Replace(next,acl);e!=nil {t.Error(e);return}
            if _,e:=p.Replace(routes,acl);e!=nil {t.Error(e);return}
        }
    }()
    wg.Wait()
}

func TestM6FullPolicyRejectsInvalidRoutes(t *testing.T) {
    _, acl, _ := policyFixture()
    for _,routes := range []map[string][]Backend{
        nil, {}, {"vpn":nil},
        {"bad service": {{Address:"127.0.0.1:30000"}}},
        {"vpn": {{Address:"no-port"}}},
        {"vpn": {{Address:"127.0.0.1:0"}}},
    } {
        if _,err := NewRuntimePolicy(routes,acl);err==nil {
            t.Fatalf("accepted invalid routes: %+v",routes)
        }
    }
}
