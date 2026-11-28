package tdp

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"time"
)

// TCPProtocol identifies the experimental TDP path in TLS ALPN.
const TCPProtocol = "tervon-tdp/1"

var ErrPathProtocol = errors.New("tdp: invalid transport path handshake")
var ErrPathDenied = errors.New("tdp: transport path not authorized")

const (
	openPath   byte = 1
	resumePath byte = 2
	pathOK     byte = 0
)

// TCPServer accepts TDP paths for one configured backend service. Use a
// distinct registry for each service. Authorize is called on open and resume.
// DialBackend must dial only the configured backend.
type TCPServer struct {
	Registry    *Registry
	Authorize   func([32]byte) bool
	DialBackend func(context.Context) (net.Conn, error)
}

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
	state := conn.ConnectionState()
	peer, err := peerFingerprint(&state)
	if err != nil {
		return SessionID{}, nil, err
	}
	if state.NegotiatedProtocol != TCPProtocol {
		return SessionID{}, nil, ErrPathProtocol
	}
	if !s.Authorize(peer) {
		return SessionID{}, nil, ErrPathDenied
	}

	var request [1]byte
	var id SessionID
	var attachment *Attachment
	if _, err := io.ReadFull(conn, request[:]); err != nil {
		return id, nil, err
	}
	switch request[0] {
	case openPath:
		backend, err := s.DialBackend(ctx)
		if err != nil {
			return id, nil, err
		}
		id, attachment, err = s.Registry.Open(peer, backend)
		if err != nil {
			if backend != nil {
				_ = backend.Close()
			}
			return id, nil, err
		}
	case resumePath:
		if _, err := io.ReadFull(conn, id[:]); err != nil {
			return id, nil, err
		}
		attachment, err = s.Registry.Resume(id, peer)
		if err != nil {
			return id, nil, err
		}
	default:
		return id, nil, ErrPathProtocol
	}

	var reply [1 + len(SessionID{})]byte
	reply[0] = pathOK
	copy(reply[1:], id[:])
	if err := writePathMessage(conn, reply[:]); err != nil {
		if request[0] == openPath {
			_ = s.Registry.End(id, attachment)
		} else {
			s.Registry.Detach(id, attachment)
		}
		return SessionID{}, nil, err
	}
	return id, attachment, nil
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
	state := conn.ConnectionState()
	if state.NegotiatedProtocol != TCPProtocol || len(state.VerifiedChains) == 0 {
		return SessionID{}, ErrPathProtocol
	}
	request := []byte{openPath}
	if resume != nil {
		request = append([]byte{resumePath}, resume[:]...)
	}
	if err := writePathMessage(conn, request); err != nil {
		return SessionID{}, err
	}
	var reply [1 + len(SessionID{})]byte
	if _, err := io.ReadFull(conn, reply[:]); err != nil {
		return SessionID{}, err
	}
	id := SessionID(reply[1:])
	if reply[0] != pathOK || (resume != nil && id != *resume) {
		return SessionID{}, ErrPathProtocol
	}
	return id, nil
}

func writePathMessage(w io.Writer, data []byte) error {
	n, err := w.Write(data)
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}

// A handshake must be bounded. The caller retains control of subsequent
// stream deadlines once negotiation returns.
func pathDeadline(ctx context.Context, conn net.Conn) (func(), error) {
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
