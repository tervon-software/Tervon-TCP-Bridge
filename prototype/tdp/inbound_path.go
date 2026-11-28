package tdp

import (
	"errors"
	"io"
)

// ServeInboundPath processes frames for an authenticated attachment. The
// caller owns wire, sets its deadlines, and must not use the path concurrently.
// This loop forwards DATA to the backend and processes ACK and RELEASE.
func (r *Registry) ServeInboundPath(id SessionID, attachment *Attachment, wire io.ReadWriter, window int) (result error) {
	if attachment == nil {
		return ErrSessionMissing
	}
	defer func() {
		r.DetachWithReason(id, attachment, PathFailureReason(result))
	}()

	backend, err := attachment.Backend()
	if err != nil {
		return err
	}
	transfer, err := attachment.Transfer(window)
	if err != nil {
		return err
	}
	path := NewFramedPath(wire, transfer)
	for {
		frame, err := ReadFrame(wire)
		if err != nil {
			err = pathIOError(err)
			if errors.Is(err, ErrPathIO) {
				return &pathFailure{reason: PathCloseTransportRead, err: err}
			}
			return err
		}
		if frame.Kind == ReleaseFrame {
			return r.ConfirmRelease(id, attachment, wire, frame)
		}
		ack, err := transfer.Handle(backend, frame)
		if err != nil {
			if ack != nil && !errors.Is(err, ErrOffset) {
				return errors.Join(
					&pathFailure{reason: PathCloseBackendWrite, err: err},
					r.EndWithReason(id, attachment, PathCloseBackendWrite),
				)
			}
			return err
		}
		if ack != nil {
			if err := path.SendACK(ack.Offset); err != nil {
				return err
			}
		}
	}
}
