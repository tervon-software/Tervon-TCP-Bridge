package tdp

import (
	"maps"
	"sync"
	"time"
)

// TransportStats aggregates negotiated attachments for one transport.
// Lifetime totals include only closed attachments.
type TransportStats struct {
	Opened         uint64
	Closed         uint64
	LifetimeTotal  time.Duration
	LifetimeMax    time.Duration
	ClosedByReason map[PathCloseReason]uint64
}

type PathStatsSnapshot struct {
	TCP       TransportStats
	QUICUDP   TransportStats
	QUICToTCP uint64
	TCPToQUIC uint64
}

// PathStats collects process-local counters. Its zero value is ready to use.
type PathStats struct {
	mu       sync.Mutex
	snapshot PathStatsSnapshot
}

// Observe can be passed directly to NewObservedRegistry.
func (s *PathStats) Observe(event PathEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var stats *TransportStats
	switch event.Transport {
	case TransportTCP:
		stats = &s.snapshot.TCP
	case TransportQUIC:
		stats = &s.snapshot.QUICUDP
	default:
		return
	}

	if event.Opened {
		stats.Opened++
		switch {
		case event.Transport == TransportTCP && event.PreviousTransport == TransportQUIC:
			s.snapshot.QUICToTCP++
		case event.Transport == TransportQUIC && event.PreviousTransport == TransportTCP:
			s.snapshot.TCPToQUIC++
		}
		return
	}

	stats.Closed++
	stats.LifetimeTotal += event.Lifetime
	stats.LifetimeMax = max(stats.LifetimeMax, event.Lifetime)
	if stats.ClosedByReason == nil {
		stats.ClosedByReason = make(map[PathCloseReason]uint64)
	}
	stats.ClosedByReason[boundedCloseReason(event.Reason)]++
}

func boundedCloseReason(reason PathCloseReason) PathCloseReason {
	switch reason {
	case PathCloseNormal, PathCloseHandoff, PathCloseACKTimeout,
		PathCloseTransportRead, PathCloseTransportWrite, PathCloseBackendWrite:
		return reason
	default:
		return PathCloseUnknown
	}
}

// Snapshot returns a detached copy, safe for the caller to retain or modify.
func (s *PathStats) Snapshot() PathStatsSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := s.snapshot
	snapshot.TCP.ClosedByReason = maps.Clone(snapshot.TCP.ClosedByReason)
	snapshot.QUICUDP.ClosedByReason = maps.Clone(snapshot.QUICUDP.ClosedByReason)
	return snapshot
}
