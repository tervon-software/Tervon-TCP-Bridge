package telemetry

import "tervon-tcp-bridge/tdp"

// TDPMetrics contains process-local counters from one TDP registry.
type TDPMetrics struct {
	TCP       TDPTransportMetrics `json:"tcp"`
	QUICUDP   TDPTransportMetrics `json:"quic_udp"`
	QUICToTCP uint64              `json:"quic_to_tcp_total"`
	TCPToQUIC uint64              `json:"tcp_to_quic_total"`
}

// TDPTransportMetrics summarizes negotiated attachments. Lifetimes include
// only closed attachments and are expressed in seconds.
type TDPTransportMetrics struct {
	Opened               uint64                         `json:"opened_total"`
	Closed               uint64                         `json:"closed_total"`
	LifetimeTotalSeconds float64                        `json:"lifetime_total_seconds"`
	LifetimeMaxSeconds   float64                        `json:"lifetime_max_seconds"`
	ClosedByReason       map[tdp.PathCloseReason]uint64 `json:"closed_by_reason"`
}

// SetTDPRegistry enables TDP metrics for registry. A nil registry omits them.
func (r *Registry) SetTDPRegistry(registry *tdp.Registry) {
	r.tdpRegistry.Store(registry)
}

func newTDPMetrics(snapshot tdp.PathStatsSnapshot) *TDPMetrics {
	return &TDPMetrics{
		TCP:       exportTDPTransport(snapshot.TCP),
		QUICUDP:   exportTDPTransport(snapshot.QUICUDP),
		QUICToTCP: snapshot.QUICToTCP,
		TCPToQUIC: snapshot.TCPToQUIC,
	}
}

func exportTDPTransport(stats tdp.TransportStats) TDPTransportMetrics {
	reasons := stats.ClosedByReason
	if reasons == nil {
		reasons = map[tdp.PathCloseReason]uint64{}
	}
	return TDPTransportMetrics{
		Opened:               stats.Opened,
		Closed:               stats.Closed,
		LifetimeTotalSeconds: stats.LifetimeTotal.Seconds(),
		LifetimeMaxSeconds:   stats.LifetimeMax.Seconds(),
		ClosedByReason:       reasons,
	}
}
