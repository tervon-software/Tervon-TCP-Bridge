package bridge

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "net"
    "regexp"
    "strconv"
    "sync/atomic"
)

// RuntimePolicy owns one immutable generation of backend routes and mTLS ACL.
// Each new stream obtains its authorization and route from the SAME snapshot.
type RuntimePolicy struct {
    current atomic.Pointer[policySnapshot]
}

type policySnapshot struct {
    routes map[string][]Backend
    authorized map[[sha256.Size]byte]map[string]bool
    digest string
}

var policyServiceName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

func buildPolicy(routes map[string][]Backend, authorized map[string][]string) (*policySnapshot, error) {
    if len(routes) == 0 || len(routes) > 4096 || len(authorized) == 0 {
        return nil, errors.New("invalid policy sizes")
    }
    copiedRoutes := make(map[string][]Backend, len(routes))
    for service, backends := range routes {
        if !policyServiceName.MatchString(service) || len(backends) == 0 || len(backends) > 64 {
            return nil, errors.New("invalid service or backend count")
        }
        copiedRoutes[service] = make([]Backend, len(backends))
        for i, backend := range backends {
            host, rawPort, err := net.SplitHostPort(backend.Address)
            if err != nil || host == "" {
                return nil, errors.New("invalid backend address")
            }
            port, err := strconv.Atoi(rawPort)
            if err != nil || port < 1 || port > 65535 {
                return nil, errors.New("invalid backend port")
            }
            copiedRoutes[service][i] = backend
        }
    }
    copiedACL := make(map[[sha256.Size]byte]map[string]bool, len(authorized))
    for fingerprint, services := range authorized {
        decoded, err := hex.DecodeString(fingerprint)
        if err != nil || len(decoded) != sha256.Size || len(services) == 0 {
            return nil, errors.New("invalid certificate fingerprint or services")
        }
        var id [sha256.Size]byte
        copy(id[:], decoded)
        if _, exists := copiedACL[id]; exists {
            return nil, errors.New("duplicate decoded certificate fingerprint")
        }
        serviceSet := make(map[string]bool, len(services))
        for _, service := range services {
            if _, exists := copiedRoutes[service]; !exists {
                return nil, errors.New("authorized service has no route")
            }
            serviceSet[service] = true
        }
        copiedACL[id] = serviceSet
    }
    // Snapshot the ACL input as well before calculating the generation digest.
    // A caller changing its input slices after publication cannot change it.
    canonicalACL := make(map[string][]string, len(authorized))
    for fingerprint, services := range authorized {
        canonicalACL[fingerprint] = append([]string(nil), services...)
    }
    // The coordinator hashes exactly this public JSON shape: routes then
    // authorized; Backend has exported Name and Address in that order.
    canonical := struct {
        Routes map[string][]Backend `json:"routes"`
        Authorized map[string][]string `json:"authorized"`
    }{copiedRoutes, canonicalACL}
    encoded, err := json.Marshal(canonical)
    if err != nil {
        return nil, errors.New("policy generation encoding failed")
    }
    digest := fmt.Sprintf("%x", sha256.Sum256(encoded))
    return &policySnapshot{routes: copiedRoutes, authorized: copiedACL, digest: digest}, nil
}

func NewRuntimePolicy(routes map[string][]Backend, authorized map[string][]string) (*RuntimePolicy, error) {
    snapshot, err := buildPolicy(routes, authorized)
    if err != nil { return nil, err }
    policy := new(RuntimePolicy)
    policy.current.Store(snapshot)
    return policy, nil
}

// Lookup checks both admission and backend list against a single generation.
// Return a copy so callers cannot mutate the published immutable snapshot.
func (policy *RuntimePolicy) Lookup(id [sha256.Size]byte, service string) ([]Backend, bool) {
    if policy == nil { return nil, false }
    snapshot := policy.current.Load()
    if snapshot == nil { return nil, false }
    // Preserve the existing named-service contract for Linux clients.
    if snapshot.authorized[id][service] {
        backends := snapshot.routes[service]
        if len(backends) == 0 { return nil, false }
        return append([]Backend(nil), backends...), true
    }
    // Android selects the active Xray TCP profile's remote inbound port.
    // Resolve that port ONLY against the current immutable generation of
    // already-authorized Bridge routes. Never dial a client-supplied address.
    port, err := strconv.Atoi(service)
    if err != nil || port < 1 || port > 65535 || strconv.Itoa(port) != service {
        return nil, false
    }
    var selected []Backend
    for name, backends := range snapshot.routes {
        if !snapshot.authorized[id][name] { continue }
        // A service with mixed backend ports cannot be selected by a single
        // inbound-port alias; fail closed rather than route unpredictably.
        if len(backends) == 0 { continue }
        matches := true
        for _, backend := range backends {
            _, rawPort, splitErr := net.SplitHostPort(backend.Address)
            n, parseErr := strconv.Atoi(rawPort)
            if splitErr != nil || parseErr != nil || n != port {
                matches = false
                break
            }
        }
        if !matches { continue }
        // Two different authorized services for the same port are ambiguous.
        if selected != nil { return nil, false }
        selected = backends
    }
    if selected == nil { return nil, false }
    return append([]Backend(nil), selected...), true
}

// Replace builds and validates a complete candidate before atomic publish.
// A rejected candidate leaves the previous routes and ACL wholly intact.
func (policy *RuntimePolicy) Replace(routes map[string][]Backend, authorized map[string][]string) (string, error) {
    if policy == nil { return "", errors.New("runtime policy unavailable") }
    snapshot, err := buildPolicy(routes, authorized)
    if err != nil { return "", err }
    policy.current.Store(snapshot)
    return snapshot.digest, nil
}

func (policy *RuntimePolicy) Digest() string {
    if policy == nil { return "" }
    snapshot := policy.current.Load()
    if snapshot == nil { return "" }
    return snapshot.digest
}
