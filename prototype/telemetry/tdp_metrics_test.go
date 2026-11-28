package telemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tervon-tcp-bridge/tdp"
)

func TestMetricsHandlerTDPOptional(t *testing.T) {
	metrics := New()
	read := func() map[string]json.RawMessage {
		t.Helper()
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/metrics/v1", nil)
		metrics.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status: %d", response.Code)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
			t.Fatal(err)
		}
		return fields
	}

	if _, ok := read()["tdp"]; ok {
		t.Fatal("tdp present without a registry")
	}
	registry, err := tdp.NewRegistry(1, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	metrics.SetTDPRegistry(registry)
	fields := read()
	if string(fields["schema"]) != `"tervon.bridge.telemetry.v1"` {
		t.Fatalf("unexpected schema: %s", fields["schema"])
	}
	want := `{"tcp":{"opened_total":0,"closed_total":0,"lifetime_total_seconds":0,"lifetime_max_seconds":0,"closed_by_reason":{}},"quic_udp":{"opened_total":0,"closed_total":0,"lifetime_total_seconds":0,"lifetime_max_seconds":0,"closed_by_reason":{}},"quic_to_tcp_total":0,"tcp_to_quic_total":0}`
	if got := string(fields["tdp"]); got != want {
		t.Fatalf("tdp JSON: %s", got)
	}
	metrics.SetTDPRegistry(nil)
	if _, ok := read()["tdp"]; ok {
		t.Fatal("tdp present after registry removal")
	}
}

func TestTDPMetricsConvertsLifetimesToSeconds(t *testing.T) {
	snapshot := tdp.PathStatsSnapshot{
		QUICToTCP: 2,
		TCPToQUIC: 1,
		QUICUDP: tdp.TransportStats{
			Opened: 3, Closed: 2,
			LifetimeTotal: 2500 * time.Millisecond,
			LifetimeMax:   1500 * time.Millisecond,
			ClosedByReason: map[tdp.PathCloseReason]uint64{
				tdp.PathCloseACKTimeout: 2,
			},
		},
	}
	got := newTDPMetrics(snapshot)
	if got.QUICToTCP != 2 || got.TCPToQUIC != 1 ||
		got.QUICUDP.Opened != 3 || got.QUICUDP.Closed != 2 ||
		got.QUICUDP.LifetimeTotalSeconds != 2.5 ||
		got.QUICUDP.LifetimeMaxSeconds != 1.5 ||
		got.QUICUDP.ClosedByReason[tdp.PathCloseACKTimeout] != 2 {
		t.Fatalf("exported snapshot: %+v", got)
	}
}
