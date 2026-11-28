package tdp

import "time"

type Transport string

const (
	TransportTCP  Transport = "tcp"
	TransportQUIC Transport = "quic_udp"
)

// PathEvent describes an authenticated session attachment. A nonempty
// PreviousTransport on an opening event identifies a completed switch.
// A closing event carries the lifetime of that individual attachment.
type PathEvent struct {
	Transport         Transport
	PreviousTransport Transport
	Opened            bool
	Lifetime          time.Duration
	Reason            PathCloseReason
}

func (r *Registry) observeOpen(id SessionID, path *Attachment, transport Transport) {
	r.mu.Lock()
	entry := r.sessions[id]
	if entry == nil || entry.path != path || !entry.openedAt.IsZero() {
		r.mu.Unlock()
		return
	}
	previous := entry.transport
	entry.transport = transport
	entry.openedAt = r.now()

	event := PathEvent{Transport: transport, Opened: true}
	if previous != transport {
		event.PreviousTransport = previous
	}
	r.stats.Observe(event)
	r.mu.Unlock()
	if r.observe != nil {
		r.observe(event)
	}
}

// Called under the registry lock, before removing or detaching an entry.
func (r *Registry) closedEvent(entry *registeredSession, now time.Time, reason PathCloseReason) PathEvent {
	if entry.openedAt.IsZero() {
		return PathEvent{}
	}
	event := PathEvent{Transport: entry.transport, Lifetime: now.Sub(entry.openedAt), Reason: reason}
	entry.openedAt = time.Time{}
	r.stats.Observe(event)
	return event
}

func (r *Registry) observeClose(event PathEvent) {
	if r.observe != nil && event.Transport != "" {
		r.observe(event)
	}
}

// PathStatsSnapshot returns the current process-local transport counters for
// this registry, regardless of whether an external observer was configured.
func (r *Registry) PathStatsSnapshot() PathStatsSnapshot {
	return r.stats.Snapshot()
}
