package tdp

import (
	"errors"
	"io"
)

var ErrSendFinished = errors.New("tdp: send direction already finished")
var ErrHalfClose = errors.New("tdp: destination does not support half-close")

// FinishSend records the final send offset. Call after queuing the last DATA.
func (t *Transfer) FinishSend() Frame {
	if !t.finishedSend {
		t.finOffset = t.send.base + uint64(len(t.send.pending))
		t.finishedSend = true
	}
	return Frame{Kind: FINFrame, Offset: t.finOffset}
}

// SendFIN sends the logical end of this direction over the attached path.
func (p *FramedPath) SendFIN() error {
	return p.write(p.transfer.FinishSend())
}

func (t *Transfer) receiveFIN(dst io.Writer, offset uint64) (*Frame, error) {
	if offset != t.receive.next {
		return nil, ErrOffset
	}
	if !t.finishedReceive {
		half, ok := dst.(interface{ CloseWrite() error })
		if !ok {
			return nil, ErrHalfClose
		}
		if err := half.CloseWrite(); err != nil {
			return nil, &pathFailure{reason: PathCloseBackendWrite, err: err}
		}
		t.finishedReceive = true
	}
	return &Frame{Kind: FINAckFrame, Offset: offset}, nil
}

func (t *Transfer) acknowledgeFIN(offset uint64) error {
	if !t.finishedSend || offset != t.finOffset {
		return ErrOffset
	}
	if err := t.send.Acknowledge(offset); err != nil {
		return err
	}
	t.finAcknowledged = true
	return nil
}
