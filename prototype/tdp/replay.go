// Package tdp contains stream state shared by TDP transport adapters.
package tdp

import (
	"errors"
	"math"
)

var (
	ErrWindowFull = errors.New("tdp: replay window full")
	ErrOffset     = errors.New("tdp: invalid stream offset")
)

// Replay retains unacknowledged bytes for one direction of one logical stream.
// Its owner must serialize calls. Acknowledgments must come from the authenticated
// peer and count bytes committed to the destination, including partial writes.
type Replay struct {
	limit   int
	base    uint64
	pending []byte
}

func NewReplay(limit int) (*Replay, error) {
	if limit <= 0 {
		return nil, ErrWindowFull
	}
	return &Replay{limit: limit}, nil
}

// Append copies payload and returns its starting offset. If the window is full,
// the caller must apply backpressure until the peer acknowledges more bytes.
func (r *Replay) Append(payload []byte) (uint64, error) {
	end := r.base + uint64(len(r.pending))
	if len(payload) > r.limit-len(r.pending) {
		return 0, ErrWindowFull
	}
	if uint64(len(payload)) > math.MaxUint64-end {
		return 0, ErrOffset
	}
	r.pending = append(r.pending, payload...)
	return end, nil
}

// Acknowledge advances the cumulative offset. Repeated and reordered ACKs are
// harmless. An ACK beyond the buffered end is invalid.
func (r *Replay) Acknowledge(next uint64) error {
	if next <= r.base {
		return nil
	}
	count := next - r.base
	if count > uint64(len(r.pending)) {
		return ErrOffset
	}
	r.pending = r.pending[int(count):]
	if len(r.pending) == 0 {
		r.pending = nil
	}
	r.base = next
	return nil
}

// From returns a copy of the bytes starting at the peer's resume offset.
func (r *Replay) From(offset uint64) ([]byte, error) {
	if offset < r.base || offset-r.base > uint64(len(r.pending)) {
		return nil, ErrOffset
	}
	return append([]byte(nil), r.pending[int(offset-r.base):]...), nil
}
