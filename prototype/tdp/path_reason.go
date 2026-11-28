package tdp

import "errors"

// PathCloseReason records the observation at the endpoint, without inferring
// an underlying network cause from a timeout.
type PathCloseReason string

const (
	PathCloseUnknown        PathCloseReason = "unknown"
	PathCloseNormal         PathCloseReason = "normal_close"
	PathCloseHandoff        PathCloseReason = "planned_handoff"
	PathCloseACKTimeout     PathCloseReason = "ack_timeout"
	PathCloseTransportRead  PathCloseReason = "transport_read_error"
	PathCloseTransportWrite PathCloseReason = "transport_write_error"
	PathCloseBackendWrite   PathCloseReason = "backend_write_error"
)

type pathFailure struct {
	reason PathCloseReason
	err    error
}

func (f *pathFailure) Error() string { return f.err.Error() }
func (f *pathFailure) Unwrap() error { return f.err }

// PathFailureReason extracts the reason observed by frame I/O. An adapter
// waiting specifically for an ACK can record PathCloseACKTimeout itself.
func PathFailureReason(err error) PathCloseReason {
	var failure *pathFailure
	if errors.As(err, &failure) {
		return failure.reason
	}
	return PathCloseUnknown
}
