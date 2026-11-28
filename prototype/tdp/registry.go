package tdp

import (
	"crypto/rand"
	"errors"
	"net"
	"sync"
	"time"
)

type SessionID [16]byte

var (
	ErrRegistryConfig = errors.New("tdp: invalid registry configuration")
	ErrRegistryFull   = errors.New("tdp: session limit reached")
	ErrSessionMissing = errors.New("tdp: session unavailable")
)

type registeredSession struct {
	session  *Session
	path     *Attachment
	deadline time.Time
}

// Registry owns registered sessions. Adapters must release paths through
// Registry.Detach and terminate sessions through Registry.End.
type Registry struct {
	mu       sync.Mutex
	sessions map[SessionID]*registeredSession
	limit    int
	grace    time.Duration
	now      func() time.Time
}

func NewRegistry(limit int, grace time.Duration) (*Registry, error) {
	if limit <= 0 || grace <= 0 {
		return nil, ErrRegistryConfig
	}
	return &Registry{
		sessions: make(map[SessionID]*registeredSession),
		limit:    limit,
		grace:    grace,
		now:      time.Now,
	}, nil
}

// Open takes ownership of backend on success only. The caller must verify
// the peer certificate and authorize the route before calling Open.
func (r *Registry) Open(peer [32]byte, backend net.Conn) (SessionID, *Attachment, error) {
	if backend == nil {
		return SessionID{}, nil, ErrRegistryConfig
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.sessions) >= r.limit {
		return SessionID{}, nil, ErrRegistryFull
	}

	for {
		var id SessionID
		if _, err := rand.Read(id[:]); err != nil {
			return SessionID{}, nil, err
		}
		if _, exists := r.sessions[id]; exists {
			continue
		}
		session := NewSession(peer, backend)
		path, err := session.Attach(peer)
		if err != nil {
			return SessionID{}, nil, err
		}
		r.sessions[id] = &registeredSession{session: session, path: path}
		return id, path, nil
	}
}

// Resume requires the verified fingerprint from the new transport.
func (r *Registry) Resume(id SessionID, peer [32]byte) (*Attachment, error) {
	r.mu.Lock()
	entry := r.sessions[id]
	if entry == nil || entry.session.owner != peer {
		r.mu.Unlock()
		return nil, ErrSessionMissing
	}
	if !entry.deadline.IsZero() && !r.now().Before(entry.deadline) {
		delete(r.sessions, id)
		r.mu.Unlock()
		_ = entry.session.Close()
		return nil, ErrSessionMissing
	}

	path, err := entry.session.Attach(peer)
	if err == nil {
		entry.path = path
		entry.deadline = time.Time{}
	}
	r.mu.Unlock()
	return path, err
}

// Detach starts the grace period after all backend I/O on path has stopped.
func (r *Registry) Detach(id SessionID, path *Attachment) {
	r.detach(id, path)
}

func (r *Registry) detach(id SessionID, path *Attachment) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.sessions[id]
	if entry == nil || path == nil || entry.path != path {
		return false
	}
	path.Detach()
	entry.path = nil
	entry.deadline = r.now().Add(r.grace)
	return true
}

// End removes the session and closes the backend for the current path.
// A stale path cannot close a session that has resumed elsewhere.
func (r *Registry) End(id SessionID, path *Attachment) error {
	r.mu.Lock()
	entry := r.sessions[id]
	if entry == nil || path == nil || entry.path != path {
		r.mu.Unlock()
		return ErrSessionMissing
	}
	delete(r.sessions, id)
	r.mu.Unlock()
	return entry.session.Close()
}

// Reap closes expired detached sessions. The serving adapter must call it
// periodically so abandoned sessions are reclaimed without incoming traffic.
func (r *Registry) Reap() int {
	r.mu.Lock()
	now := r.now()
	var expired []*Session
	for id, entry := range r.sessions {
		if !entry.deadline.IsZero() && !now.Before(entry.deadline) {
			delete(r.sessions, id)
			expired = append(expired, entry.session)
		}
	}
	r.mu.Unlock()

	for _, session := range expired {
		_ = session.Close()
	}
	return len(expired)
}
