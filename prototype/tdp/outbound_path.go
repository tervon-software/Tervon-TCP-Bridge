package tdp

import (
	"errors"
	"io"
)

// ForwardNext reads and forwards up to one DATA frame. At source EOF it
// records and sends FIN after the final DATA. The caller serializes access,
// handles inbound ACKs and resumes on a new path after transport failure.
func (p *FramedPath) ForwardNext(src io.Reader, scratch []byte) (finished bool, err error) {
	data, readErr := p.transfer.ReadOutbound(src, scratch)
	finished = errors.Is(readErr, io.EOF)
	var fin Frame
	if finished {
		// Record FIN before attempting the last write so replay can recover both.
		fin = p.transfer.FinishSend()
	}
	if data.Kind == DataFrame {
		if err := p.write(data); err != nil {
			if !finished && readErr != nil {
				return false, errors.Join(err, readErr)
			}
			return finished, err
		}
	}
	if finished {
		return true, p.write(fin)
	}
	return false, readErr
}
