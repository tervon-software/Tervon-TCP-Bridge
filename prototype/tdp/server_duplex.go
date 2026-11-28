package tdp

import (
	"context"
	"errors"
	"time"
)

// ServeDuplexPath runs a negotiated attachment. On transport failure the
// backend and transfer state remain available for a subsequent RESUME.
// releaseTimeout bounds the write of RELEASED once all workers have stopped.
func (r *Registry) ServeDuplexPath(ctx context.Context, id SessionID, attachment *Attachment, wire DuplexWire, window int, releaseTimeout time.Duration) (result error) {
	if attachment == nil {
		return ErrSessionMissing
	}
	defer func() {
		r.DetachWithReason(id, attachment, PathFailureReason(result))
	}()
	if releaseTimeout <= 0 {
		return ErrRegistryConfig
	}

	backend, err := attachment.Backend()
	if err != nil {
		return err
	}
	transfer, err := attachment.Transfer(window)
	if err != nil {
		return err
	}
	err = runDuplex(ctx, wire, backend, transfer, true)
	switch {
	case err == nil:
		return r.End(id, attachment)
	case errors.Is(err, errPeerRelease):
		releaseCtx, cancel := context.WithTimeout(ctx, releaseTimeout)
		defer cancel()
		cleanup, err := pathDeadline(releaseCtx, wire)
		if err != nil {
			return err
		}
		defer cleanup()
		return r.ConfirmRelease(id, attachment, wire, Frame{Kind: ReleaseFrame})
	case PathFailureReason(err) == PathCloseBackendWrite:
		return errors.Join(err, r.EndWithReason(id, attachment, PathCloseBackendWrite))
	default:
		return err
	}
}
