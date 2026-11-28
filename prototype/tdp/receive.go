package tdp

import (
	"io"
	"math"
)

// Receiver forwards one direction of a logical stream to a persistent
// destination. Its owner must serialize calls, including across path changes.
type Receiver struct {
	next uint64
}

// Deliver writes only bytes that have not already reached dst. The returned
// offset is the cumulative acknowledgment, even after a partial write.
func (r *Receiver) Deliver(dst io.Writer, offset uint64, payload []byte) (uint64, error) {
	if offset > r.next || uint64(len(payload)) > math.MaxUint64-offset {
		return r.next, ErrOffset
	}
	overlap := r.next - offset
	if overlap >= uint64(len(payload)) {
		return r.next, nil
	}
	payload = payload[int(overlap):]
	n, err := dst.Write(payload)
	r.next += uint64(n)
	if n != len(payload) && err == nil {
		err = io.ErrShortWrite
	}
	return r.next, err
}
