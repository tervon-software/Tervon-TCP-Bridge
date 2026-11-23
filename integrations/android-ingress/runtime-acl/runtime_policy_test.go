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

func TestM6FullPolicyChangesRoutesAndAuthorizationTogether(t *testing.T) {
    routes, acl, id := policyFixture()
    p, err := NewRuntimePolicy(routes, acl)
    if err != nil { t.Fatal(err) }
    nextRoutes := map[string][]Backend{
        "new-service": {{Name:"next", Address:"127.0.0.1:49001"}},
    }
    nextACL := map[string][]string{hex.EncodeToString(id[:]): {"new-service"}}
    if _,err := p.Replace(nextRoutes,nextACL);err!=nil { t.Fatal(err) }
    if old,ok := p.Lookup(id,"vpn");ok || len(old)>0 {
        t.Fatal("old service survived full policy replacement")
    }
    if fresh,ok := p.Lookup(id,"new-service");!ok || len(fresh)!=1 || fresh[0].Address!="127.0.0.1:49001" {
        t.Fatal("new ACL and new route not active in same generation")
    }
    before := p.Digest()
    if _,err := p.Replace(nextRoutes,acl);err==nil {
        t.Fatal("inconsistent routes/ACL pair incorrectly published")
    }
    if p.Digest()!=before { t.Fatal("rejected mixed policy advanced generation") }
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

// TestM6AndroidPortSelectorUsesOnlyCurrentAuthorizedPolicy verifies that
// the Android inbound port is a selector, never a dial target.
func TestM6AndroidPortSelectorUsesOnlyCurrentAuthorizedPolicy(t *testing.T) {
    _, _, id := policyFixture()
    other := sha256.Sum256([]byte("not-enrolled"))
    routes := map[string][]Backend{
        "vpn-tcp": {{Name:"vless-tcp", Address:"127.0.0.1:30000"}},
        "vpn-xhttp": {{Name:"vless-xhttp", Address:"127.0.0.1:55535"}},
        "vpn-trojan": {{Name:"trojan", Address:"127.0.0.1:55636"}},
    }
    acl := map[string][]string{hex.EncodeToString(id[:]): {"vpn-tcp", "vpn-xhttp", "vpn-trojan"}}
    policy, err := NewRuntimePolicy(routes, acl)
    if err != nil { t.Fatal(err) }
    for _, port := range []string{"30000", "55535", "55636"} {
        got, ok := policy.Lookup(id, port)
        if !ok || len(got) != 1 || got[0].Address != "127.0.0.1:"+port {
            t.Fatalf("wrong profile selection %s: %v %v", port, got, ok)
        }
    }
    for _, invalid := range []string{"0", "65536", "-1", "030000", "127.0.0.1:30000", "443", ""} {
        if got, ok := policy.Lookup(id, invalid); ok || len(got) > 0 {
            t.Fatalf("accepted unauthorized port %q", invalid)
        }
    }
    if got, ok := policy.Lookup(other, "55636"); ok || len(got) > 0 {
        t.Fatal("unenrolled certificate accessed an inbound")
    }
    // Existing Go/Linux clients may still request their named service.
    if got, ok := policy.Lookup(id, "vpn-xhttp"); !ok || len(got) != 1 {
        t.Fatal("named-service compatibility was lost")
    }
    // Remove and move a backend: the old port must immediately stop working.
    routes["vpn-xhttp"] = []Backend{{Name:"moved", Address:"127.0.0.1:55888"}}
    if _, err := policy.Replace(routes, acl); err != nil { t.Fatal(err) }
    if got, ok := policy.Lookup(id, "55535"); ok || len(got)>0 {
        t.Fatal("stale port survived policy replacement")
    }
    if got, ok := policy.Lookup(id, "55888"); !ok || len(got)!=1 || got[0].Address!="127.0.0.1:55888" {
        t.Fatal("new dynamic inbound port was not selected")
    }
    // The previous immutable generation cannot be changed by caller mutation.
    if got, ok := policy.Lookup(id,"30000"); !ok || got[0].Address!="127.0.0.1:30000" {
        t.Fatal("unrelated route lost on swap")
    }
}

func TestM6AndroidPortSelectorFailsClosedOnAmbiguousOrMixedRoutes(t *testing.T) {
    _, _, id := policyFixture()
    acl := map[string][]string{hex.EncodeToString(id[:]):{"a","b"}}
    routes := map[string][]Backend{
        "a": {{Address:"127.0.0.1:30000"}},
        "b": {{Address:"127.0.0.1:30000"}},
    }
    policy, err := NewRuntimePolicy(routes, acl)
    if err != nil { t.Fatal(err) }
    if got, ok := policy.Lookup(id,"30000"); ok || len(got)>0 {
        t.Fatal("ambiguous backend port selected")
    }
    routes["b"] = []Backend{{Address:"127.0.0.1:55535"},{Address:"127.0.0.1:30000"}}
    if _, err := policy.Replace(routes, acl); err != nil { t.Fatal(err) }
    if got, ok := policy.Lookup(id,"55535"); ok || len(got)>0 {
        t.Fatal("mixed-port backend selected")
    }
    if got, ok := policy.Lookup(id,"30000"); !ok || len(got)!=1 {
        t.Fatal("unique authorized port was not available")
    }
}

// A numeric service name must never override the actual backend port.
func TestM6AndroidNumericNameCannotBypassBackendPort(t *testing.T) {
    _, _, id := policyFixture()
    acl := map[string][]string{hex.EncodeToString(id[:]): {"30000"}}
    policy, err := NewRuntimePolicy(map[string][]Backend{
        "30000": {{Address:"127.0.0.1:55636"}},
    }, acl)
    if err != nil { t.Fatal(err) }
    if got,ok:=policy.Lookup(id,"30000");ok||len(got)>0 {
        t.Fatal("numeric service name selected wrong backend port")
    }
    if got,ok:=policy.Lookup(id,"55636");!ok||len(got)!=1 {
        t.Fatal("actual authorized backend port was not selectable")
    }
}
