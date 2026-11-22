package bridge

import (
	"crypto/sha256"
	"errors"
	"net"
	"strconv"
	"sync/atomic"
)

// RuntimePolicy publishes an immutable, mutually consistent ACL+backend snapshot.
// Stream admission obtains authorization and backend selection from one Load.
// Existing streams retain their already-open backend TCP socket.
type RuntimePolicy struct {
	current atomic.Pointer[policySnapshot]
}

type policySnapshot struct {
	acl    *aclSnapshot
	routes map[string][]Backend
}

func validatedPolicy(authorized map[string][]string, routes map[string][]Backend) (*policySnapshot, error) {
	if len(routes) == 0 || len(routes) > 4096 {
		return nil, errors.New("invalid route count")
	}
	copied := make(map[string][]Backend, len(routes))
	for service, backends := range routes {
		if len(service) == 0 || len(service) > 64 || len(backends) == 0 || len(backends) > 64 {
			return nil, errors.New("invalid service or backend list")
		}
		group := make([]Backend, len(backends))
		copy(group, backends)
		for _, backend := range group {
			if backend.Address == "" {
				return nil, errors.New("empty backend address")
			}
			host, port, err := net.SplitHostPort(backend.Address)
			n, parseErr := strconv.Atoi(port)
			if err != nil || host == "" || parseErr != nil || n < 1 || n > 65535 {
				return nil, errors.New("invalid backend address")
			}
			// The trusted control-plane adapter must validate the endpoint
			// against actual Xray. Never accept an arbitrary client dial target.
		}
		copied[service] = group
	}
	acl, err := parseACL(authorized, copied)
	if err != nil {
		return nil, err
	}
	return &policySnapshot{acl: acl, routes: copied}, nil
}

func NewRuntimePolicy(authorized map[string][]string, routes map[string][]Backend) (*RuntimePolicy, error) {
	p, err := validatedPolicy(authorized, routes)
	if err != nil { return nil, err }
	r := new(RuntimePolicy)
	r.current.Store(p)
	return r, nil
}

// Select is a single linearizable authorization+route read. Returned backends
// are copied so callers cannot mutate the published policy.
func (r *RuntimePolicy) Select(fingerprint [sha256.Size]byte, service string) ([]Backend, bool) {
	if r == nil { return nil, false }
	p := r.current.Load()
	if p == nil || !p.acl.services[fingerprint][service] { return nil, false }
	list := p.routes[service]
	if len(list) == 0 { return nil, false }
	dst := make([]Backend, len(list))
	copy(dst, list)
	return dst, true
}

// Replace validates the entire candidate first. Invalid updates leave the old
// ACL and backend map intact, including during concurrent stream admissions.
func (r *RuntimePolicy) Replace(authorized map[string][]string, routes map[string][]Backend) error {
	if r == nil { return errors.New("runtime policy unavailable") }
	p, err := validatedPolicy(authorized, routes)
	if err != nil { return err }
	r.current.Store(p)
	return nil
}
