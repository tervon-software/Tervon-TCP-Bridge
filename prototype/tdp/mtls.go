package tdp

import (
	"crypto/sha256"
	"crypto/tls"
	"errors"
	"net"
)

var ErrUnverifiedPeer = errors.New("tdp: verified client certificate required")

func peerFingerprint(state *tls.ConnectionState) ([32]byte, error) {
	if state == nil || !state.HandshakeComplete ||
		len(state.VerifiedChains) == 0 || len(state.VerifiedChains[0]) == 0 {
		return [32]byte{}, ErrUnverifiedPeer
	}
	leaf := state.VerifiedChains[0][0]
	return sha256.Sum256(leaf.Raw), nil
}

// OpenMTLS takes ownership of backend on success. The server must authorize the
// requested route before calling it. state must come from the server side of
// the current connection configured with tls.RequireAndVerifyClientCert.
func (r *Registry) OpenMTLS(state *tls.ConnectionState, backend net.Conn) (SessionID, *Attachment, error) {
	peer, err := peerFingerprint(state)
	if err != nil {
		return SessionID{}, nil, err
	}
	return r.Open(peer, backend)
}

// ResumeMTLS uses the verified client identity of the new transport.
func (r *Registry) ResumeMTLS(id SessionID, state *tls.ConnectionState) (*Attachment, error) {
	peer, err := peerFingerprint(state)
	if err != nil {
		return nil, err
	}
	return r.Resume(id, peer)
}
