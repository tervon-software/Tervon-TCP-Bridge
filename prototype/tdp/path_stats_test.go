package tdp

import (
	"testing"
	"time"
)

func TestPathStatsSnapshot(t *testing.T) {
	var stats PathStats
	for _, event := range []PathEvent{
		{Transport: TransportQUIC, Opened: true},
		{Transport: TransportQUIC, Reason: PathCloseACKTimeout, Lifetime: time.Second},
		{Transport: TransportTCP, PreviousTransport: TransportQUIC, Opened: true},
		{Transport: TransportTCP, Reason: PathCloseHandoff, Lifetime: 2 * time.Second},
		{Transport: TransportQUIC, PreviousTransport: TransportTCP, Opened: true},
		{Transport: TransportQUIC, Reason: "unexpected detail", Lifetime: 3 * time.Second},
		{Transport: TransportTCP, PreviousTransport: TransportTCP, Opened: true},
	} {
		stats.Observe(event)
	}

	got := stats.Snapshot()
	if got.QUICToTCP != 1 || got.TCPToQUIC != 1 {
		t.Fatalf("switch counters: %+v", got)
	}
	if got.TCP.Opened != 2 || got.TCP.Closed != 1 || got.TCP.LifetimeTotal != 2*time.Second ||
		got.TCP.LifetimeMax != 2*time.Second || got.TCP.ClosedByReason[PathCloseHandoff] != 1 {
		t.Fatalf("TCP: %+v", got.TCP)
	}
	if got.QUICUDP.Opened != 2 || got.QUICUDP.Closed != 2 || got.QUICUDP.LifetimeTotal != 4*time.Second ||
		got.QUICUDP.LifetimeMax != 3*time.Second || got.QUICUDP.ClosedByReason[PathCloseACKTimeout] != 1 ||
		got.QUICUDP.ClosedByReason[PathCloseUnknown] != 1 || len(got.QUICUDP.ClosedByReason) != 2 {
		t.Fatalf("QUIC: %+v", got.QUICUDP)
	}
	got.QUICUDP.ClosedByReason[PathCloseACKTimeout] = 99
	if stats.Snapshot().QUICUDP.ClosedByReason[PathCloseACKTimeout] != 1 {
		t.Fatal("mutating the snapshot changed the collected counters")
	}
}
