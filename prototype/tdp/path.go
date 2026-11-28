package tdp

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
)

// PathProtocol identifies the experimental TDP version in TLS ALPN.
const PathProtocol = "tervon-tdp/1"

var ErrPathProtocol = errors.New("tdp: invalid transport path handshake")
var ErrPathDenied = errors.New("tdp: transport path not authorized")

const (
	openPath   byte = 1
	resumePath byte = 2
	probePath  byte = 3
	pathOK     byte = 0
)

// PathServer accepts TDP paths for one configured backend service. Use a
// distinct registry for each service. Authorize is called on open and resume.
// DialBackend must dial only the configured backend.
type PathServer struct {
	Registry    *Registry
	Authorize   func([32]byte) bool
	DialBackend func(context.Context) (net.Conn, error)
}

// acceptPath handles negotiation after the transport's TLS handshake.
// The transport adapter is responsible for bounding the stream I/O.
func (s PathServer) acceptPath(ctx context.Context, conn io.ReadWriter, state tls.ConnectionState, allowProbe bool) (SessionID, *Attachment, error) {
	if s.Registry == nil || s.Authorize == nil || s.DialBackend == nil {
		return SessionID{}, nil, ErrRegistryConfig
	}
	peer, err := peerFingerprint(&state)
	if err != nil {
		return SessionID{}, nil, err
	}
	if state.NegotiatedProtocol != PathProtocol {
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
	case probePath:
		if !allowProbe {
			return id, nil, ErrPathProtocol
		}
		return id, nil, writePathMessage(conn, []byte{pathOK})
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

func openPathSession(conn io.ReadWriter, state tls.ConnectionState, resume *SessionID) (SessionID, error) {
	if !state.HandshakeComplete || state.NegotiatedProtocol != PathProtocol || len(state.VerifiedChains) == 0 {
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
