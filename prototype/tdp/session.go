package tdp

import (
	"errors"
	"net"
	"sync"
)

var (
	ErrSessionOwner  = errors.New("tdp: session belongs to another peer")
	ErrSessionBusy   = errors.New("tdp: session already has an attached path")
	ErrSessionClosed = errors.New("tdp: session closed")
)

// Session owns a backend connection independently of its transport path.
// The caller must verify mTLS and authorize the route before creating a
// session. The session registry is responsible for expiration and cleanup.
type Session struct {
	mu      sync.Mutex
	owner   [32]byte
	backend net.Conn
	active  *Attachment
	closed  bool
}

// NewSession takes ownership of backend. owner is the SHA-256 fingerprint
// of the verified client certificate.
func NewSession(owner [32]byte, backend net.Conn) *Session {
	return &Session{owner: owner, backend: backend}
}

// Attachment grants a single transport path use of the backend.
type Attachment struct {
	session *Session
}

// Attach requires the fingerprint from the current path's verified mTLS
// connection. Only one transport path may handle the session at a time.
func (s *Session) Attach(peer [32]byte) (*Attachment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if peer != s.owner {
		return nil, ErrSessionOwner
	}
	if s.closed {
		return nil, ErrSessionClosed
	}
	if s.active != nil {
		return nil, ErrSessionBusy
	}
	attachment := &Attachment{session: s}
	s.active = attachment
	return attachment, nil
}

// Backend returns the connection for the attached path. The path must finish
// all backend I/O before calling Detach.
func (a *Attachment) Backend() (net.Conn, error) {
	s := a.session
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.active != a {
		return nil, ErrSessionClosed
	}
	return s.backend, nil
}

// Detach releases the path after its backend I/O has stopped. The backend
// stays open for the next path. A stale detach cannot release a newer path.
func (a *Attachment) Detach() {
	s := a.session
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active == a {
		s.active = nil
	}
}

// Close ends the logical session and interrupts any pending backend I/O.
func (s *Session) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.active = nil
	s.mu.Unlock()
	return s.backend.Close()
}
