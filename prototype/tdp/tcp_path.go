package tdp

import (
	"context"
	"crypto/tls"
	"time"
)

// TCPProtocol identifies the experimental TDP path in TLS ALPN.
const TCPProtocol = PathProtocol

// TCPServer accepts TLS-over-TCP paths for one configured backend service.
type TCPServer PathServer

// AcceptPath performs mTLS and attaches the connection to a logical session.
// The caller owns conn and must close it on failure. On success the caller
// must detach or end the attachment after all backend I/O has stopped.
func (s TCPServer) AcceptPath(ctx context.Context, conn *tls.Conn) (SessionID, *Attachment, error) {
	if s.Registry == nil || s.Authorize == nil || s.DialBackend == nil {
		return SessionID{}, nil, ErrRegistryConfig
	}
	cleanup, err := pathDeadline(ctx, conn)
	if err != nil {
		return SessionID{}, nil, err
	}
	defer cleanup()
	if err := conn.HandshakeContext(ctx); err != nil {
		return SessionID{}, nil, err
	}
	return PathServer(s).acceptPath(ctx, conn, conn.ConnectionState(), false)
}

// OpenTCPPath negotiates a new session or resumes the supplied session ID.
// conn must use a verifying TLS client configuration.
func OpenTCPPath(ctx context.Context, conn *tls.Conn, resume *SessionID) (SessionID, error) {
	cleanup, err := pathDeadline(ctx, conn)
	if err != nil {
		return SessionID{}, err
	}
	defer cleanup()
	if err := conn.HandshakeContext(ctx); err != nil {
		return SessionID{}, err
	}
	return openPathSession(conn, conn.ConnectionState(), resume)
}

// Negotiation must be bounded. The caller retains control of subsequent
// stream deadlines once negotiation returns.
func pathDeadline(ctx context.Context, conn interface{ SetDeadline(time.Time) error }) (func(), error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return nil, ErrPathProtocol
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, err
	}
	done := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		_ = conn.SetDeadline(time.Now())
		close(done)
	})
	return func() {
		if !stop() {
			<-done
		}
		_ = conn.SetDeadline(time.Time{})
	}, nil
}
