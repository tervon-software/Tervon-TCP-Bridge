package tdp

import (
	"context"
	"errors"
	"io"
	"math"
	"net"
	"sync"
	"time"
)

// DuplexWire is a negotiated path whose blocked I/O can be interrupted.
type DuplexWire interface {
	io.ReadWriter
	SetDeadline(time.Time) error
}

// ErrPathReleased indicates the server confirmed that the current attachment
// was released. The caller can now resume the same transfer on another path.
var ErrPathReleased = errors.New("tdp: path released")

var errPeerRelease = errors.New("tdp: peer requested path release")

type duplexRead struct {
	data []byte
	err  error
}

type duplexFrame struct {
	frame Frame
	err   error
}

// RunDuplex forwards an attached path in both directions until FIN exchange
// completes or I/O fails. It owns the wire, local deadlines and transfer
// state until return. The caller discards the wire on return and may reattach
// the same local connection and transfer after a transport failure.
func RunDuplex(ctx context.Context, wire DuplexWire, local net.Conn, transfer *Transfer) error {
	return RunDuplexWithRelease(ctx, wire, local, transfer, nil)
}

// RunDuplexWithRelease requests handoff when release becomes readable.
// The caller should bound the path lifetime with ctx or wire deadlines.
// Once RELEASE is sent, a failed exchange requires renegotiation.
func RunDuplexWithRelease(ctx context.Context, wire DuplexWire, local net.Conn, transfer *Transfer, release <-chan struct{}) error {
	return runDuplex(ctx, wire, local, transfer, false, release)
}

func runDuplex(ctx context.Context, wire DuplexWire, local net.Conn, transfer *Transfer, allowRelease bool, release <-chan struct{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	interruptDone := make(chan struct{})
	stopInterrupt := context.AfterFunc(ctx, func() {
		now := time.Now()
		_ = wire.SetDeadline(now)
		_ = local.SetDeadline(now)
		close(interruptDone)
	})

	budget := make(chan int)
	localRead := make(chan duplexRead, 1)
	received := make(chan duplexFrame)
	write := make(chan Frame)
	written := make(chan error, 1)
	var workers sync.WaitGroup
	workers.Add(3)
	go func() {
		defer workers.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case limit := <-budget:
				data := make([]byte, limit)
				n, err := local.Read(data)
				localRead <- duplexRead{data[:n], err}
			}
		}
	}()
	go func() {
		defer workers.Done()
		for {
			frame, err := ReadFrame(wire)
			select {
			case <-ctx.Done():
				return
			case received <- duplexFrame{frame, err}:
			}
			if err != nil {
				return
			}
		}
	}()
	go func() {
		defer workers.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case frame := <-write:
				written <- pathIOError(WriteFrame(wire, frame))
			}
		}
	}()

	err := driveDuplex(ctx, local, transfer, budget, localRead, received, write, written, allowRelease, release)
	cancel()
	if !stopInterrupt() {
		<-interruptDone
	} else {
		now := time.Now()
		_ = wire.SetDeadline(now)
		_ = local.SetDeadline(now)
	}
	workers.Wait()
	// A read may have completed just as the path failed. Preserve those bytes
	// before returning ownership of the local connection to the caller.
	select {
	case result := <-localRead:
		var timeout net.Error
		if errors.As(result.err, &timeout) && timeout.Timeout() {
			result.err = nil
		}
		if readErr := queueLocalRead(transfer, result); readErr != nil {
			err = errors.Join(err, readErr)
		}
	default:
	}
	return errors.Join(err, local.SetDeadline(time.Time{}))
}

func queueLocalRead(transfer *Transfer, result duplexRead) error {
	if len(result.data) != 0 {
		if _, err := transfer.Queue(result.data); err != nil {
			return err
		}
	}
	if errors.Is(result.err, io.EOF) {
		transfer.FinishSend()
		return nil
	}
	return result.err
}

func driveDuplex(ctx context.Context, local net.Conn, transfer *Transfer,
	budget chan<- int, localRead <-chan duplexRead, received <-chan duplexFrame,
	write chan<- Frame, written <-chan error, allowRelease bool, release <-chan struct{},
) error {
	offset := transfer.send.base
	var ack *Frame
	var reading, writing, finSent bool
	var releasing, releaseSent bool
	for {
		if !releasing && transfer.finAcknowledged && transfer.finishedReceive && ack == nil && !writing {
			return nil
		}
		send := transfer.send
		var request chan<- int
		room := min(send.limit-len(send.pending), MaxFramePayload)
		remaining := math.MaxUint64 - (send.base + uint64(len(send.pending)))
		if uint64(room) > remaining {
			room = int(remaining)
		}
		if !releasing && !reading && !transfer.finishedSend && room > 0 {
			request = budget
		}
		var output chan<- Frame
		var frame Frame
		if !writing && !releaseSent {
			switch {
			case ack != nil:
				frame = *ack
				output = write
			case releasing:
				frame = Frame{Kind: ReleaseFrame}
				output = write
			case offset < send.base+uint64(len(send.pending)):
				var err error
				frame, err = transfer.Pending(max(offset, send.base))
				if err != nil {
					return err
				}
				output = write
			case transfer.finishedSend && !transfer.finAcknowledged && !finSent:
				frame = transfer.FinishSend()
				output = write
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
			releasing = true
			release = nil
		case request <- room:
			reading = true
		case result := <-localRead:
			reading = false
			if err := queueLocalRead(transfer, result); err != nil {
				return err
			}
		case output <- frame:
			writing = true
			switch frame.Kind {
			case DataFrame:
				offset = frame.Offset + uint64(len(frame.Payload))
			case FINFrame:
				finSent = true
			case ReleaseFrame:
				releaseSent = true
			default:
				ack = nil
			}
		case err := <-written:
			writing = false
			if err != nil {
				return &pathFailure{reason: PathCloseTransportWrite, err: err}
			}
		case in := <-received:
			if in.err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				err := pathIOError(in.err)
				if errors.Is(err, ErrPathIO) {
					return &pathFailure{reason: PathCloseTransportRead, err: err}
				}
				return err
			}
			if releaseSent && in.frame.Kind == ReleasedFrame {
				return ErrPathReleased
			}
			if allowRelease && in.frame.Kind == ReleaseFrame {
				return errPeerRelease
			}
			reply, err := transfer.Handle(local, in.frame)
			if err != nil {
				if PathFailureReason(err) == PathCloseBackendWrite {
					return err
				}
				if reply != nil && !errors.Is(err, ErrOffset) {
					return &pathFailure{reason: PathCloseBackendWrite, err: err}
				}
				return err
			}
			if reply != nil && (ack == nil || ack.Kind != FINAckFrame) {
				ack = reply
			}
		}
	}
}
