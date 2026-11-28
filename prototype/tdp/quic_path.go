package tdp

import (
	"context"
	"io"

	"github.com/quic-go/quic-go"
)

// AcceptQUICPath answers probes and accepts one session stream on a
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
	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			return nil, SessionID{}, nil, err
		}
		id, attachment, err := s.acceptQUICStream(ctx, stream, state)
		if err != nil {
			cancelQUICPath(stream)
			return nil, SessionID{}, nil, err
		}
		if attachment != nil {
			return stream, id, attachment, nil
		}
		if err := stream.Close(); err != nil {
			cancelQUICPath(stream)
			return nil, SessionID{}, nil, err
		}
	}
}

func (s PathServer) acceptQUICStream(ctx context.Context, stream *quic.Stream, state quic.ConnectionState) (SessionID, *Attachment, error) {
	cleanup, err := pathDeadline(ctx, stream)
	if err != nil {
		return SessionID{}, nil, err
	}
	defer cleanup()
	return s.acceptPath(ctx, stream, state.TLS, true)
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

// ProbeQUICPath verifies a TDP round trip on an authenticated QUIC connection.
// The probe consumes its own stream and does not open or attach a session.
// ctx must provide a deadline.
func ProbeQUICPath(ctx context.Context, conn *quic.Conn) error {
	state := conn.ConnectionState()
	if state.Used0RTT || !state.TLS.HandshakeComplete ||
		state.TLS.NegotiatedProtocol != PathProtocol ||
		len(state.TLS.VerifiedChains) == 0 {
		return ErrPathProtocol
	}
	if _, ok := ctx.Deadline(); !ok {
		return ErrPathProtocol
	}
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return err
	}
	cleanup, err := pathDeadline(ctx, stream)
	if err != nil {
		cancelQUICPath(stream)
		return err
	}
	defer cleanup()
	if err := writePathMessage(stream, []byte{probePath}); err != nil {
		cancelQUICPath(stream)
		return err
	}
	if err := stream.Close(); err != nil {
		cancelQUICPath(stream)
		return err
	}
	var reply [1]byte
	if _, err := io.ReadFull(stream, reply[:]); err != nil {
		stream.CancelRead(1)
		return err
	}
	if reply[0] != pathOK {
		stream.CancelRead(1)
		return ErrPathProtocol
	}
	var extra [1]byte
	if n, err := io.ReadFull(stream, extra[:]); n != 0 || err != io.EOF {
		stream.CancelRead(1)
		if err != nil {
			return err
		}
		return ErrPathProtocol
	}
	return nil
}
