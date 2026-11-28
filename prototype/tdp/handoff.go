package tdp

import "io"

// RequestRelease asks the server to detach the current path so the client can
// resume over another transport. The caller must pause other frame I/O and
// set a deadline before calling.
func RequestRelease(wire io.ReadWriter) error {
	if err := WriteFrame(wire, Frame{Kind: ReleaseFrame}); err != nil {
		return pathIOError(err)
	}
	reply, err := ReadFrame(wire)
	if err != nil {
		return pathIOError(err)
	}
	if reply.Kind != ReleasedFrame {
		return ErrPathProtocol
	}
	return nil
}

// ConfirmRelease handles a RELEASE read by the current path's server loop.
// Call only after all other path and backend I/O has stopped. The owning
// attachment is released before confirmation, including if the reply fails.
func (r *Registry) ConfirmRelease(id SessionID, path *Attachment, wire io.Writer, request Frame) error {
	if request.Kind != ReleaseFrame || !validFrame(request.Kind, request.Offset, len(request.Payload)) {
		return ErrPathProtocol
	}
	if !r.detach(id, path) {
		return ErrSessionMissing
	}
	return pathIOError(WriteFrame(wire, Frame{Kind: ReleasedFrame}))
}
