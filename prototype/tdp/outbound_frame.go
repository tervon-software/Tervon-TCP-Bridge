package tdp

import (
	"io"
	"math"
)

// ReadOutbound reads at most one DATA frame and retains it in the replay
// window before returning. The caller must serialize access to the Transfer.
// A returned DATA frame is valid even when the reader also returns an error.
func (t *Transfer) ReadOutbound(src io.Reader, scratch []byte) (Frame, error) {
	if t.finishedSend {
		return Frame{}, ErrSendFinished
	}
	if len(scratch) == 0 {
		return Frame{}, io.ErrShortBuffer
	}
	send := t.send
	room := send.limit - len(send.pending)
	if room == 0 {
		return Frame{}, ErrWindowFull
	}
	size := min(room, len(scratch), MaxFramePayload)
	end := send.base + uint64(len(send.pending))
	if remaining := uint64(math.MaxUint64) - end; uint64(size) > remaining {
		size = int(remaining)
		if size == 0 {
			return Frame{}, ErrOffset
		}
	}

	n, readErr := src.Read(scratch[:size])
	if n == 0 {
		return Frame{}, readErr
	}
	frame, err := t.Queue(scratch[:n])
	if err != nil {
		return Frame{}, err
	}
	return frame, readErr
}
