package tdp

import (
	"errors"
	"io"
)

// FramedPath exchanges DATA and ACK on a negotiated transport. Its caller
// serializes operations, sets transport deadlines and stops all I/O before
// attaching the transfer to another path.
type FramedPath struct {
	wire     io.ReadWriter
	transfer *Transfer
}

func NewFramedPath(wire io.ReadWriter, transfer *Transfer) *FramedPath {
	return &FramedPath{wire: wire, transfer: transfer}
}

// Send retains payload before writing. If the write fails, the caller must
// discard the path and can replay the retained bytes on another path.
func (p *FramedPath) Send(payload []byte) error {
	frame, err := p.transfer.Queue(payload)
	if err != nil {
		return err
	}
	return p.write(frame)
}

// Replay writes all bytes still awaiting acknowledgment. Each frame retains
// its original offset, so the receiver can suppress duplicate delivery.
func (p *FramedPath) Replay() error {
	offset := p.transfer.send.base
	for {
		frame, err := p.transfer.Pending(offset)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := p.write(frame); err != nil {
			return err
		}
		offset += uint64(len(frame.Payload))
	}
}

// Receive processes one frame and returns the ACK, if any. The caller
// decides when to send it, including when delivery made partial progress.
func (p *FramedPath) Receive(dst io.Writer) (*Frame, error) {
	frame, err := ReadFrame(p.wire)
	if err != nil {
		err = pathIOError(err)
		if errors.Is(err, ErrPathIO) {
			err = &pathFailure{reason: PathCloseTransportRead, err: err}
		}
		return nil, err
	}
	ack, err := p.transfer.Handle(dst, frame)
	if ack != nil && err != nil && !errors.Is(err, ErrOffset) {
		err = &pathFailure{reason: PathCloseBackendWrite, err: err}
	}
	return ack, err
}

func (p *FramedPath) SendACK(offset uint64) error {
	if offset > p.transfer.ReceiveOffset() {
		return ErrOffset
	}
	return p.write(Frame{Kind: AckFrame, Offset: offset})
}

func (p *FramedPath) write(frame Frame) error {
	err := pathIOError(WriteFrame(p.wire, frame))
	if errors.Is(err, ErrPathIO) {
		return &pathFailure{reason: PathCloseTransportWrite, err: err}
	}
	return err
}
