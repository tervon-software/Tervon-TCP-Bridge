package tdp

import (
	"context"

	"github.com/quic-go/quic-go"
)

// AcceptQUICPath accepts one client-initiated bidirectional stream on a
// completed QUIC connection. Configure the listener to require verified mTLS
// and disable 0-RTT. The caller owns conn and must detach or end the attachment.
func (s PathServer) AcceptQUICPath(ctx context.Context, conn *quic.Conn) (*quic.Stream, SessionID, *Attachment, error) {
	state := conn.ConnectionState()
	if state.Used0RTT || !state.TLS.HandshakeComplete {
		return nil, SessionID{}, nil, ErrPathProtocol
	}
	if _, ok := ctx.Deadline(); !ok {
		return nil, SessionID{}, nil, ErrPathProtocol
	}
	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		return nil, SessionID{}, nil, err
	}
	id, attachment, err := s.acceptQUICStream(ctx, stream, state)
	if err != nil {
		cancelQUICPath(stream)
		return nil, SessionID{}, nil, err
	}
	return stream, id, attachment, nil
}

func (s PathServer) acceptQUICStream(ctx context.Context, stream *quic.Stream, state quic.ConnectionState) (SessionID, *Attachment, error) {
	cleanup, err := pathDeadline(ctx, stream)
	if err != nil {
		return SessionID{}, nil, err
	}
	defer cleanup()
	return s.acceptPath(ctx, stream, state.TLS)
}

// OpenQUICPath opens or resumes a TDP session over a verified QUIC connection.
func OpenQUICPath(ctx context.Context, conn *quic.Conn, resume *SessionID) (*quic.Stream, SessionID, error) {
	state := conn.ConnectionState()
	if state.Used0RTT || !state.TLS.HandshakeComplete {
		return nil, SessionID{}, ErrPathProtocol
	}
	if _, ok := ctx.Deadline(); !ok {
		return nil, SessionID{}, ErrPathProtocol
	}
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, SessionID{}, err
	}
	id, err := openQUICStream(ctx, stream, state, resume)
	if err != nil {
		cancelQUICPath(stream)
		return nil, SessionID{}, err
	}
	return stream, id, nil
}

func openQUICStream(ctx context.Context, stream *quic.Stream, state quic.ConnectionState, resume *SessionID) (SessionID, error) {
	cleanup, err := pathDeadline(ctx, stream)
	if err != nil {
		return SessionID{}, err
	}
	defer cleanup()
	return openPathSession(stream, state.TLS, resume)
}

func cancelQUICPath(stream *quic.Stream) {
	stream.CancelRead(1)
	stream.CancelWrite(1)
}
