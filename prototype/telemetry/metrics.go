// Package telemetry collects bounded aggregate bridge measurements without
// inspecting payloads or retaining client identifiers.
package telemetry

import (
	"crypto/tls"
	"encoding/json"
	"math"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"tervon-tcp-bridge/tdp"
)

// Reason is a fixed-cardinality error category. Do not add user-controlled labels.
type Reason string

const (
	ReasonBackend  Reason = "backend_dial"
	ReasonHTTP     Reason = "http_status"
	ReasonTimeout  Reason = "timeout"
	ReasonCanceled Reason = "cancellation"
	ReasonShutdown Reason = "shutdown"
	ReasonAuth     Reason = "authorization"
	ReasonTLS      Reason = "tls_handshake"
)

type Snapshot struct {
	TDP                   *TDPMetrics       `json:"tdp,omitempty"`
	BuildVersion          string            `json:"build_version"`
	BuildCommit           string            `json:"build_commit"`
	Ready                 bool              `json:"ready"`
	MaxSessions           int               `json:"max_sessions"`
	MemoryAllocBytes      uint64            `json:"memory_alloc_bytes"`
	BackendDialSuccess    uint64            `json:"backend_dial_success"`
	BackendDialFailure    uint64            `json:"backend_dial_failure"`
	ConnectLatencyP50Ms   float64           `json:"connect_latency_p50_ms"`
	ConnectLatencyP95Ms   float64           `json:"connect_latency_p95_ms"`
	ConnectLatencyP99Ms   float64           `json:"connect_latency_p99_ms"`
	ConnectLatencySamples uint64            `json:"connect_latency_samples"`
	TLS13H2MTLSRequests   uint64            `json:"tls13_h2_mtls_requests"`
	Schema                string            `json:"schema"`
	At                    time.Time         `json:"at"`
	UptimeSeconds         float64           `json:"uptime_seconds"`
	SessionsStarted       uint64            `json:"sessions_started"`
	SessionsCompleted     uint64            `json:"sessions_completed"`
	SessionsRejected      uint64            `json:"sessions_rejected"`
	SessionsActive        int64             `json:"sessions_active"`
	UploadBytes           uint64            `json:"upload_bytes"`
	DownloadBytes         uint64            `json:"download_bytes"`
	UploadBps             float64           `json:"upload_bytes_per_second"`
	DownloadBps           float64           `json:"download_bytes_per_second"`
	Errors                map[Reason]uint64 `json:"errors"`
	Goroutines            int               `json:"goroutines"`
	PaddingState          string            `json:"padding_state"`
	JitterState           string            `json:"jitter_state"`
}

// Registry is safe for concurrent data-plane updates and infrequent snapshots.
// Rates are calculated between snapshots; consumers should scrape at fixed intervals.
type Registry struct {
	tdpRegistry    atomic.Pointer[tdp.Registry]
	BuildVersion   string
	BuildCommit    string
	maxSessions    atomic.Int64
	ready          atomic.Bool
	backendSuccess atomic.Uint64
	backendFailure atomic.Uint64
	tls13h2mtls    atomic.Uint64
	latencyBuckets [12]atomic.Uint64
	started        time.Time
	active         atomic.Int64
	startedCount   atomic.Uint64
	completed      atomic.Uint64
	rejected       atomic.Uint64
	up             atomic.Uint64
	down           atomic.Uint64
	errors         [7]atomic.Uint64
	mu             sync.Mutex
	lastAt         time.Time
	lastUp         uint64
	lastDown       uint64
}

func New() *Registry                { return &Registry{started: time.Now()} }
func (r *Registry) SetReady(v bool) { r.ready.Store(v) }
func (r *Registry) SetMaxSessions(n int) {
	if n > 0 {
		r.maxSessions.Store(int64(n))
	}
}
func (r *Registry) BackendDial(ok bool) {
	if ok {
		r.backendSuccess.Add(1)
	} else {
		r.backendFailure.Add(1)
	}
}

// ObserveConnectLatency records only bounded bucket counts, never per-connection data.
func (r *Registry) ObserveConnectLatency(d time.Duration) {
	ms := d.Milliseconds()
	bounds := [...]int64{1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000}
	idx := len(bounds)
	for i, b := range bounds {
		if ms <= b {
			idx = i
			break
		}
	}
	r.latencyBuckets[idx].Add(1)
}

// ObserveTLS counts only negotiated and verified sessions, not configured settings.
func (r *Registry) ObserveTLS(state *tls.ConnectionState) {
	if state != nil && state.Version == tls.VersionTLS13 && state.NegotiatedProtocol == "h2" && len(state.VerifiedChains) > 0 && len(state.PeerCertificates) > 0 {
		r.tls13h2mtls.Add(1)
	}
}
func latencyPercentile(counts [12]uint64, q float64) float64 {
	var total uint64
	for _, n := range counts {
		total += n
	}
	if total == 0 {
		return 0
	}
	target := uint64(math.Ceil(float64(total) * q))
	bounds := [...]float64{1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 2000}
	var cumulative uint64
	for i, n := range counts {
		cumulative += n
		if cumulative >= target {
			return bounds[i]
		}
	}
	return bounds[len(bounds)-1]
}

func (r *Registry) Begin() func() {
	r.startedCount.Add(1)
	r.active.Add(1)
	var ended atomic.Bool
	return func() {
		if ended.CompareAndSwap(false, true) {
			r.completed.Add(1)
			r.active.Add(-1)
		}
	}
}
func (r *Registry) Reject() { r.rejected.Add(1) }

// AddUpload/AddDownload count only successfully copied application bytes.
// They must be called at io.Copy boundaries, not by wrapping *net.TCPConn,
// because the bridge relies on CloseWrite half-close semantics.
func (r *Registry) AddUpload(n int64) {
	if n > 0 {
		r.up.Add(uint64(n))
	}
}
func (r *Registry) AddDownload(n int64) {
	if n > 0 {
		r.down.Add(uint64(n))
	}
}
func (r *Registry) Error(reason Reason) {
	switch reason {
	case ReasonBackend:
		r.errors[0].Add(1)
	case ReasonHTTP:
		r.errors[1].Add(1)
	case ReasonTimeout:
		r.errors[2].Add(1)
	case ReasonCanceled:
		r.errors[3].Add(1)
	case ReasonShutdown:
		r.errors[4].Add(1)
	case ReasonAuth:
		r.errors[5].Add(1)
	case ReasonTLS:
		r.errors[6].Add(1)
	}
}
func (r *Registry) Snapshot() Snapshot {
	s := Snapshot{Schema: "tervon.bridge.telemetry.v1", At: time.Time{}, UptimeSeconds: 0, SessionsStarted: r.startedCount.Load(), SessionsCompleted: r.completed.Load(), SessionsRejected: r.rejected.Load(), SessionsActive: r.active.Load(), UploadBytes: 0, DownloadBytes: 0, Goroutines: runtime.NumGoroutine(), PaddingState: "NOT_IMPLEMENTED", JitterState: "NOT_IMPLEMENTED", Errors: map[Reason]uint64{}}
	for i, k := range []Reason{ReasonBackend, ReasonHTTP, ReasonTimeout, ReasonCanceled, ReasonShutdown, ReasonAuth, ReasonTLS} {
		s.Errors[k] = r.errors[i].Load()
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	s.MemoryAllocBytes = mem.Alloc
	s.BuildVersion = r.BuildVersion
	s.BuildCommit = r.BuildCommit
	s.Ready = r.ready.Load()
	s.MaxSessions = int(r.maxSessions.Load())
	s.BackendDialSuccess = r.backendSuccess.Load()
	s.BackendDialFailure = r.backendFailure.Load()
	s.TLS13H2MTLSRequests = r.tls13h2mtls.Load()
	var buckets [12]uint64
	for i := range buckets {
		buckets[i] = r.latencyBuckets[i].Load()
		s.ConnectLatencySamples += buckets[i]
	}
	s.ConnectLatencyP50Ms = latencyPercentile(buckets, 0.50)
	s.ConnectLatencyP95Ms = latencyPercentile(buckets, 0.95)
	s.ConnectLatencyP99Ms = latencyPercentile(buckets, 0.99)
	r.mu.Lock()
	// Take the timestamp while holding the snapshot lock. Capturing it
	// before lock acquisition can make concurrent snapshots go backwards.
	now := time.Now()
	s.At = now.UTC()
	s.UptimeSeconds = now.Sub(r.started).Seconds()
	up := r.up.Load()
	down := r.down.Load()
	s.UploadBytes = up
	s.DownloadBytes = down
	if !r.lastAt.IsZero() {
		elapsed := now.Sub(r.lastAt).Seconds()
		if elapsed > 0 {
			s.UploadBps = float64(up-r.lastUp) / elapsed
			s.DownloadBps = float64(down-r.lastDown) / elapsed
		}
	}
	r.lastAt = now
	r.lastUp = up
	r.lastDown = down
	r.mu.Unlock()
	if registry := r.tdpRegistry.Load(); registry != nil {
		s.TDP = newTDPMetrics(registry.PathStatsSnapshot())
	}
	return s
}

// Handler must only be mounted on a private, loopback-only listener. It does
// not itself provide authentication or enforce the listener's bind address.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/metrics/v1" {
			http.NotFound(w, req)
			return
		}
		if req.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(r.Snapshot())
	})
}
