package tdp

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

const (
	frameVersion    = 1
	frameHeaderSize = 12
	MaxFramePayload = 32 * 1024
)

type FrameKind byte

const (
	DataFrame FrameKind = 1
	AckFrame  FrameKind = 2
)

var ErrFrame = errors.New("tdp: invalid frame")
var ErrFrameVersion = errors.New("tdp: unsupported frame version")

// Frame carries data at Offset or acknowledges all bytes before Offset.
// Transport adapters must authenticate peers before processing frames.
type Frame struct {
	Kind    FrameKind
	Offset  uint64
	Payload []byte
}

func validFrame(kind FrameKind, offset uint64, size int) bool {
	switch kind {
	case DataFrame:
		return size > 0 && size <= MaxFramePayload && uint64(size) <= math.MaxUint64-offset
	case AckFrame:
		return size == 0
	default:
		return false
	}
}

// ReadFrame reads one frame and validates the header before allocating payload.
func ReadFrame(r io.Reader) (Frame, error) {
	var header [frameHeaderSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Frame{}, err
	}
	if header[0] != frameVersion {
		return Frame{}, ErrFrameVersion
	}
	kind := FrameKind(header[1])
	size := int(binary.BigEndian.Uint16(header[2:4]))
	offset := binary.BigEndian.Uint64(header[4:])
	if !validFrame(kind, offset, size) {
		return Frame{}, ErrFrame
	}
	frame := Frame{Kind: kind, Offset: offset}
	if size != 0 {
		frame.Payload = make([]byte, size)
		if _, err := io.ReadFull(r, frame.Payload); err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return Frame{}, err
		}
	}
	return frame, nil
}

// WriteFrame writes one frame. Callers must serialize writes to each stream.
// After a write error the stream must be discarded because a partial frame
// may have reached the peer.
func WriteFrame(w io.Writer, frame Frame) error {
	size := len(frame.Payload)
	if !validFrame(frame.Kind, frame.Offset, size) {
		return ErrFrame
	}
	wire := make([]byte, frameHeaderSize+size)
	wire[0] = frameVersion
	wire[1] = byte(frame.Kind)
	binary.BigEndian.PutUint16(wire[2:4], uint16(size))
	binary.BigEndian.PutUint64(wire[4:frameHeaderSize], frame.Offset)
	copy(wire[frameHeaderSize:], frame.Payload)
	n, err := w.Write(wire)
	if err == nil && n != len(wire) {
		return io.ErrShortWrite
	}
	return err
}
