package tdp

import (
	"context"
	"errors"
	"fmt"
	"io"
)

var ErrPathIO = errors.New("tdp: transport I/O failed")

func pathIOError(err error) error {
	if err == nil || errors.Is(err, ErrFrame) || errors.Is(err, ErrFrameVersion) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrPathIO, err)
}

// RunWithTCPFallback runs an already opened QUIC path and, on transport
// failure, resumes the same session over TCP once. run must return only after
// all its I/O has stopped. releaseQUIC must release the old transport; the
// server detaches independently. resumeTCP must finish authenticated RESUME
// and may wait for the old server-side attachment to be released.
// run and resumeTCP must respect ctx and set appropriate I/O deadlines.
func RunWithTCPFallback(
	ctx context.Context,
	id SessionID,
	transfer *Transfer,
	quicWire io.ReadWriter,
	releaseQUIC func(),
	resumeTCP func(context.Context, SessionID) (io.ReadWriteCloser, error),
	run func(context.Context, *FramedPath) error,
) error {
	err := run(ctx, NewFramedPath(quicWire, transfer))
	releaseQUIC()
	if !errors.Is(err, ErrPathIO) {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	tcpWire, err := resumeTCP(ctx, id)
	if err != nil {
		return err
	}
	defer tcpWire.Close()
	path := NewFramedPath(tcpWire, transfer)
	if err := path.Replay(); err != nil {
		return err
	}
	return run(ctx, path)
}
