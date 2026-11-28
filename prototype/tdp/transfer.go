package tdp

import "io"

// Transfer holds the send window and receive cursor of one logical stream
// endpoint. Calls must be serialized by the transport adapter.
type Transfer struct {
	send    *Replay
	receive Receiver
}

func NewTransfer(window int) (*Transfer, error) {
	send, err := NewReplay(window)
	if err != nil {
		return nil, err
	}
	return &Transfer{send: send}, nil
}

// Queue retains outbound bytes before the adapter attempts to send them.
func (t *Transfer) Queue(payload []byte) (Frame, error) {
	if len(payload) == 0 || len(payload) > MaxFramePayload {
		return Frame{}, ErrFrame
	}
	offset, err := t.send.Append(payload)
	if err != nil {
		return Frame{}, err
	}
	return t.Pending(offset)
}

// Pending returns at most one frame beginning at offset. io.EOF indicates
// that offset is exactly the end of the buffered stream.
func (t *Transfer) Pending(offset uint64) (Frame, error) {
	r := t.send
	if offset < r.base || offset-r.base > uint64(len(r.pending)) {
		return Frame{}, ErrOffset
	}
	pending := r.pending[int(offset-r.base):]
	if len(pending) == 0 {
		return Frame{}, io.EOF
	}
	pending = pending[:min(len(pending), MaxFramePayload)]
	return Frame{Kind: DataFrame, Offset: offset, Payload: append([]byte(nil), pending...)}, nil
}

// ReceiveOffset is the next byte expected from the peer.
func (t *Transfer) ReceiveOffset() uint64 {
	return t.receive.next
}

// Handle applies an inbound frame. A DATA frame produces a cumulative ACK
// even on partial delivery. An inbound ACK produces no response.
func (t *Transfer) Handle(dst io.Writer, frame Frame) (*Frame, error) {
	if !validFrame(frame.Kind, frame.Offset, len(frame.Payload)) {
		return nil, ErrFrame
	}
	if frame.Kind == AckFrame {
		return nil, t.send.Acknowledge(frame.Offset)
	}
	next, err := t.receive.Deliver(dst, frame.Offset, frame.Payload)
	return &Frame{Kind: AckFrame, Offset: next}, err
}
