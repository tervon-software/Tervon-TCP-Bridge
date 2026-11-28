# Tervon TCP Bridge 1.0.2-TDP

Branch: work/tdp. Base: aa7c2cbd2a17a64fb6c423bd7abd9ac8050fb799.

Approved first milestone: retain the active backend connection when the QUIC/UDP path fails, resume over TCP, and verify the complete payload by checksum. A short pause is acceptable.

This commit adds a bounded replay buffer for one stream direction and tests cumulative acknowledgments and resumption from a later offset. The two directions will need separate state. The current HTTP/2 handler ties the backend connection to the request context, so the TDP implementation will need to keep the backend alive across path changes and bind resumption to the authenticated client.

Package check: go test -race ./tdp and go vet ./tdp on Linux amd64, Go 1.23.12, passed on the staged sources. QUIC, TCP transport integration and the network acceptance test are pending. Packet manipulation is outside this milestone.

## Experimental scope and receive-side replay

Approved experiments: TCP/QUIC session continuity, optional cover traffic, and optional packet desynchronization. Test the hybrid alone first. Once the two options exist, compare the same workload with neither option, each option alone, and both together. Record transfer completion and checksum, failover pause, successful connections, and overhead. Decide which variants to retain after measurement.

Added a receive cursor for one stream direction. It skips already delivered bytes, rejects gaps, and acknowledges the exact count written to the destination after a partial write. Package checks on Linux amd64, Go 1.23.12: go test -race -count=1 ./tdp and go vet ./tdp passed. Network failover, cover traffic and packet desynchronization are pending.

## Versioned stream frames

Added prototype/tdp/frame.go and frame_test.go. Frame version 1 uses a 12-byte header: version (1 byte), kind (1 byte), big-endian payload length (2 bytes), and big-endian logical stream offset (8 bytes). DATA carries up to 32 KiB. ACK carries the next committed offset and has no payload. Reject unsupported versions and malformed headers before allocating a payload. A partial transport write invalidates the current path.

Package checks on Linux amd64, Go 1.23.12: go test -race -count=1 ./tdp and go vet ./tdp passed. The frame version check does not negotiate session capabilities. Authenticated session attachment, TCP/QUIC integration and failover acceptance remain pending.

## Session ownership and backend lifetime

TDP stands for Tervon Dual-path Protocol. Added prototype/tdp/session.go and session_test.go on work/tdp, based on dffc8164fc761badc83325203b2b2e409e545c0c. A session owns one backend TCP connection and admits one attachment for the fingerprint provided by the verified mTLS peer. Detach preserves the backend, a stale detach cannot release a newer attachment, and explicit session close closes the backend. The loopback test uses synthetic fingerprints and exercises the same accepted TCP connection before and after detach.

Homeserver, Go 1.25.5 linux/amd64: go test -race -count=1 ./... and go vet ./... PASS. SHA-256 session.go: 505e58cc217cedba015b4c34f56e64bacd04ae2e7ed0bc7e9aef4f4515d16019. SHA-256 session_test.go: c8241838dff305c02a3c7277d7f29c4f3b2e297ad78af130814bb1214055eaf8. Both hashes were matched before committing.

The package requires adapters to authenticate the peer and finish backend I/O before detaching. Session discovery, expiry, transport integration and real QUIC-to-TCP acceptance remain pending.

## Bounded session registry

Added prototype/tdp/registry.go and registry_test.go to work/tdp, based on 73c6a414d43ba19b53367eea04736dd297ab4f4d. The registry allocates random session IDs, limits registered sessions, resumes only for the matching peer fingerprint, and releases expired detached sessions. Adapters must call Reap periodically and finish backend I/O before detaching.

Homeserver, Go 1.25.5 linux/amd64: go test -race -count=1 ./... and go vet ./... PASS. Verified SHA-256 of registry.go: 04d03b3ae25d008bc90b9032e430ce2136cbd190e2b7b7c127575ea973a9a885. Verified SHA-256 of registry_test.go: a04ad562522c6370e089e06daf3708d8ca45fd0b3afb12e579f849da321deced. Next: integrate the TCP and QUIC paths with verified mTLS identity and test real failover.

## Verified mTLS identity for session open and resume

Added prototype/tdp/mtls.go and prototype/tdp_mtls_test.go on work/tdp, based on 662b9412b5b3dc8ea88fd5b7efcde62c7dacec44. Registry adapters derive the client fingerprint from the verified TLS leaf. A loopback HTTPS test exercises real TLS 1.3 handshakes, rejects a different trusted client certificate, and verifies that the original identity resumes the existing backend. The first run failed when the test set a read deadline after closing net.Pipe. Moving the deadline before End fixed the test.

Homeserver: go test -race -count=1 ./... and go vet ./... PASS. Recovered archive SHA-256 verified for mtls.go (48c1d76708c945778e8f5e303936694c377bc0eb1445e4b39a4a2c1beff2f09c) and tdp_mtls_test.go (81808fbe97149fdb5fa2f9418c71f48ad65ceec194e411830ab152847d2bb64a). TCP/QUIC adapter integration and end-to-end failover remain pending.

## Transfer state across path attachments

Branch work/tdp, base e51d31d4a986c609bfdb3d4cd2d771a83fa47eae. Added transfer.go and transfer_test.go and linked the transfer state to session.go. DATA and ACK processing now share a send window and receive cursor across attachments. The test drops an ACK, reattaches, replays the pending frame and checks the destination receives the payload exactly once.

Linux amd64, Go 1.23.12: go test -race -count=1 ./tdp and go vet ./tdp passed. This exercises the stream state without real TCP/QUIC path switching. Transport adapters and network acceptance remain pending.

## Experimental TCP path negotiation

Branch work/tdp, base bc19ea0ac83b4380d127a65777adb20f5ee760e1. Added tcp_path.go and tcp_path_test.go. Client and server negotiate TDP v1 via TLS ALPN. The server requires a verified peer certificate, checks authorization on open and resume, uses a service-scoped registry, and bounds negotiation with a context deadline. OPEN dials the configured backend. RESUME submits the session ID and reattaches to the original backend. No CLI integration or production listener is enabled by this change.

Linux amd64, Go 1.23.12: go test -race -count=1 ./tdp and go vet ./tdp passed. A TLS 1.3 test over net.Pipe verifies one backend dial across OPEN and RESUME. Frame pumping, QUIC and real network failover acceptance remain pending.

## DATA and ACK exchange on an attached path

Branch work/tdp, base 679c2e5893e5e90a209a9a722976a6a7b34cd3cb. Added framed_path.go and extended tcp_path_test.go. FramedPath queues outbound DATA before writing, reads and applies DATA/ACK, and can resend the pending window on another negotiated path. Its caller serializes operations and manages deadlines and attachment cleanup.

Linux amd64, Go 1.23.12: go test -race -count=1 ./tdp and go vet ./tdp passed. The test uses two TLS 1.3 exchanges over net.Pipe, drops the first ACK, verifies byte-exact single delivery to an in-memory sink upon replay, and checks the ACK clears the send window. The original backend is retained. Production pumping, QUIC and actual network failover remain pending.

## Shared session negotiation for transport adapters

Branch work/tdp, base 1a098e5c8c4d17542345e360da23c7431db6ed66. Added prototype/tdp/path.go and adapted tcp_path.go. The shared protocol takes a bidirectional stream plus verified TLS state and applies the same ALPN, authorization, OPEN/RESUME and rollback logic. The TCP adapter owns its TLS handshake and negotiation deadline. This establishes the integration point for a QUIC stream.

Linux amd64, Go 1.23.12: go test -race -count=1 ./tdp and go vet ./tdp passed, including the existing mTLS resume and DATA replay test. No QUIC dependency was added. QUIC adapter, continuous pumping and network failover acceptance remain pending.

## QUIC transport adapter and loopback handoff

Branch work/tdp, base 661741188081e016b836fe41103da1df69fff136. Pinned quic-go v0.54.1 to retain the existing Go 1.23 baseline. Added quic_path.go, go.sum and dependency metadata in go.mod, and extended tcp_path_test.go. The QUIC adapter uses the shared TDP negotiation over a bidirectional stream with verified TLS state and rejects early data. The loopback test opens TDP over real UDP/QUIC, delivers DATA, closes the QUIC connection before ACK, and explicitly reattaches over real TCP/mTLS. Replay delivers the bytes once, ACK clears the sender window, and the backend dial count stays one.

Linux amd64, Go 1.23.12: go test -race -count=10 ./tdp, go vet ./tdp and go mod verify passed. The backend fixture is net.Pipe. The test performs a deliberate connection close and explicit resume. Automatic detection of silent UDP loss, continuous duplex pumping, production integration and network acceptance remain pending.

## Silent UDP loss in loopback handoff test

Branch work/tdp, base 04c689a689093f9e73d419bcb306f4c56353d263. Updated tcp_path_test.go. After DATA delivery, the QUIC case drops datagrams in both directions at the server socket wrapper. The server queues an ACK and the client observes a read timeout while the transport is still open. The test then closes the failed path, explicitly detaches and resumes over TCP, and checks single delivery, release of the replay window and one backend dial.

Linux amd64, Go 1.23.12: go test -race -count=10 ./tdp and go vet ./tdp passed. The test controls the deadline and recovery. Continuous duplex forwarding, automatic recovery in the Bridge runtime and production acceptance remain pending.

## Client fallback runner

Branch work/tdp, base 22d4fd3c9278de89be0a89edf85634ce4ed01253. Added fallback.go and fallback_test.go, and annotated transport read/write failures in framed_path.go. After a QUIC path I/O failure, the runner releases the first path, calls authenticated TCP resume with the same ID, and replays buffered data before continuing. Protocol violations and caller cancellation do not initiate fallback. The caller supplies the path runner, deadlines, and transport setup. Server-side detachment and handling the race with TCP resume are still the responsibility of the runtime adapter.

Linux amd64, Go 1.23.12: go test -race -count=5 ./tdp and go vet ./tdp passed. Runner tests use an injected wire failure. The existing real-socket packet-loss test still performs explicit handoff. Runtime wiring and continuous duplex failover acceptance remain pending.

## Client fallback runner on real sockets

Branch work/tdp, base ab3023ec9c135c3a0e986828f2c0de67461704f5. Updated tcp_path_test.go to exercise RunWithTCPFallback with real QUIC/UDP and TCP/mTLS sockets. The QUIC case blocks UDP after DATA delivery, observes an ACK read timeout and lets the runner initiate TCP RESUME and replay. The test verifies the same session ID, exactly-once delivery to the sink, acknowledgment of replayed data and a single backend dial. The test still supplies the read deadline and explicitly detaches server state on first path cleanup. The backend fixture is net.Pipe.

Linux amd64, Go 1.23.12: go test -race -count=10 ./tdp and go vet ./tdp passed. Continuous bidirectional forwarding and independent server-side path failure handling are still pending.

## Authenticated QUIC round-trip probe

Branch work/tdp, base 0ab4e9b476cc921ae8595c976f4f3070c0899d77. Added PROBE to the QUIC path handshake, using a dedicated stream, existing mTLS route authorization and a bounded deadline. The probe does not claim a session or dial a backend. The real-socket test probes immediately before opening the QUIC session and verifies only one backend dial across the test.

Homeserver: go test -race -count=1 ./..., go test -race -count=10 ./tdp, go vet ./..., go mod verify and git diff --check passed. All four source SHA-256 hashes matched before committing.
- path.go: 65184619a0b13f9ab85e08530626fbb7459b7b61cfcf9ed6ae04542a101b4cf1
- tcp_path.go: 5ce758733a28747b38d19dacb5f825a3be371d2973914e926caa03fffbbc2d01
- tcp_path_test.go: 0e5267b93617149424caa5d212d0f9886b46cb199bae2e1755773654810316e4
- quic_path.go: fa07155d08134a43c2e9efced323ecc5013b6c1203d422fe7616022454f86f74

The test exercises a healthy QUIC probe before OPEN. Probing after recovery while TCP remains attached and coordinated TCP-to-QUIC migration are pending.

## QUIC recovery probe while TCP remains attached

Branch work/tdp, base f77597ef83e09e35b7a0ea9d191d8c21471a655e. Extended tcp_path_test.go to restore UDP after TCP fallback, perform a real authenticated QUIC PROBE on a new connection, verify the original TCP attachment remains active, and send and acknowledge another DATA frame over TCP. The complete received payload is checked and the backend dial count remains one. An initial build failure due to an undefined test identifier was fixed by using the fingerprint of the test certificate.

Homeserver: git diff --check, go test -race -count=1 ./..., go test -race -count=10 ./tdp and go vet ./... passed. Verified source SHA-256: 8e4a4b471efe94a3029ea9fa84b3bc68080babdd3aa1b0f48a2e83e241d4c529. Coordinated migration of the active session back to QUIC remains pending.

## Coordinated release before resuming over another transport

Branch work/tdp, base d60347772da1730251f834570bd234e31f470fc9. Added RELEASE and RELEASED control frames, RequestRelease and Registry.ConfirmRelease. The server releases the current attachment before writing confirmation, and rejects a stale attachment without releasing the newer one. Detachment preserves the backend and transfer state. Both sides must stop concurrent path I/O before the handshake, and the client must set a deadline.

Linux amd64, Go 1.23.12: go test -race -count=10 ./tdp and go vet ./tdp passed. The release test exchanges control frames over net.Pipe and verifies that resume keeps the original backend. The TCP-to-QUIC return still needs runtime dispatch, quiescing, and the real-socket round-trip test.

## Full transport handoff exercised on loopback sockets

Branch work/tdp, base 217c861d11d3379fabbcd22a606a7763ad55d5f0. Extended tcp_path_test.go. The test drops UDP, continues on TCP, restores UDP and probes a new QUIC connection while TCP remains attached. After another TCP DATA/ACK exchange the test drives RELEASE/RELEASED over TCP, resumes the same ID on the probed QUIC connection, and transfers fresh DATA/ACK there. The sink receives the expected concatenated payload exactly once, and backend identity and one dial are verified. Data goes to a test buffer; the backend fixture is net.Pipe.

Linux amd64, Go 1.23.12: go test -race -count=10 ./tdp and go vet ./tdp passed. The test orchestrates return-to-QUIC and includes only upload payload with ACKs in the reverse direction. Continuous bidirectional forwarding and automatic return in the runtime remain pending.

## Transport monitoring hooks

Branch work/tdp, base 49d38060ccdce696d242e8732037a60c86c6aa5e. Added optional Registry path observer. Successful negotiated attachments emit an opening event tagged tcp or quic_udp. A previous transport on an opening event marks a successful switch and its direction. Detach and End emit the individual attachment lifetime. Same-transport resume is an opening without a switch. Probes and failed negotiation do not emit successful openings. Events omit peer fingerprints, session IDs, addresses and payload. Lifetimes measure TDP attachments, excluding TLS setup and the detached grace period.

Linux amd64, Go 1.23.12: go test -race -count=10 ./tdp and go vet ./tdp passed. The real-socket handoff test verifies both switch directions and one lifetime event per attachment. Metrics aggregation/export and AI Processor 1.0.3 consumption and control of padding/jitter are separate, pending integrations.

## Observed reasons for transport closure

Branch work/tdp, base 70ba6f3172da83612ebc71d8c24058420b71dd2c. Added PathEvent.Reason, reason-aware registry detach/end, and framed I/O failure classification. Frame reads, frame writes, and writes to the receiving sink are distinguished, preserving the original error for errors.Is/errors.As. An adapter explicitly waiting for an ACK can report ack_timeout; a generic read timeout does not infer UDP blocking. Planned RELEASE and ordinary End have separate reasons. Opening events carry no reason. No raw errors or peer identifiers are added to telemetry events.

Linux amd64, Go 1.23.12: go test -race -count=10 ./tdp and go vet ./tdp passed. Tests cover I/O classification and closure events across the loopback transport cycle, where the test supplies the ACK-timeout observation. Runtime reporting, ACK-wait duration and pending-byte measurements, exporter, and AI Processor integration remain pending.

## Process-local transport statistics snapshot

Branch work/tdp, base 4b36fcfc1a15cc4e7d391c4119104ffcc450eb89. Added path_stats.go and path_stats_test.go. PathStats.Observe consumes registry path events. Snapshot provides separate TCP and QUIC/UDP opens and closes, total and maximum lifetime of closed paths, bounded reason counts and directional switch totals. Unknown reason values are folded into unknown. Snapshots copy the reason maps. Individual lifetimes are still available through close events.

Linux amd64, Go 1.23.12: go test -race -count=10 ./tdp and go vet ./tdp passed. Unit test validates both switch directions, excludes same-transport reopen from switch counts, and verifies reasons, durations and snapshot isolation. This package is ready to be wired to the registry observer. CLI/runtime wiring, metrics export and AI Processor remain pending.

## Registry-backed transport counters

Branch work/tdp, base 89db832e1c2852978c2e3a27f0864169001dada9. Wired PathStats into the registry and exposed Registry.PathStatsSnapshot(). Successful path negotiation and path closure update process-local statistics with or without a custom observer. The optional observer still receives individual lifetime and reason events outside the registry lock. PROBE and failed negotiation do not increment opens.

Linux amd64, Go 1.23.12: go test -race -count=10 ./tdp and go vet ./tdp passed. The real-socket loopback test checks separate open/close totals and lifetimes, both switch directions and closure reasons for QUIC-to-TCP-to-QUIC. TCP-only reattachment does not count as a switch. The CLI, exporter and AI Processor do not yet consume the snapshot.

## Optional TDP metrics in the local Bridge snapshot

Branch work/tdp, base 4d20393b2f11334284211cc5d82a5e53fb258a90. Added an optional tdp object to /metrics/v1, configured via telemetry.Registry.SetTDPRegistry. It exports separate TCP and QUIC/UDP open and close totals, closure reasons, lifetime totals and maxima in seconds, and both switch directions. Without a configured TDP registry the object is omitted and the existing schema identifier stays unchanged.

Homeserver: git diff --check, go test -race -count=1 ./..., go vet ./... and go mod verify passed. Tests verify the optional JSON object and conversion of nonzero lifetimes and counters. All three source SHA-256 hashes matched before committing:
- metrics.go: e7dfa80fda0a5200e025d24bf2581881341959b0d5005c08230caffa97c3043d
- tdp_metrics.go: 84ba3715c271ba3888d4b07302a8305063920f23cbacb77cf42993d6e7909b19
- tdp_metrics_test.go: 4ecbb08a564da28fd02a822b9adfa4f14c320b9cc86c5037439449199e6b4f24

Bridge runtime wiring and AI Processor consumption remain pending.

## Inbound frame loop for an attached TDP path

Branch work/tdp, base f05ab9ad8413a2a5be33e6c7f65bd42840aa8cff. Added inbound_path.go and inbound_path_test.go. ServeInboundPath forwards DATA to the backend, confirms delivery, applies incoming ACKs and handles RELEASE. It detaches on transport failure and ends a session on terminal backend write failure. The test checks backend delivery and ACK, retention of the same connection through RELEASE, and the planned-handoff metric.

Homeserver: git diff --check, go test -race -count=1 ./..., go test -race -count=10 ./tdp, go vet ./... and go mod verify passed. Verified source SHA-256:
- inbound_path.go: c4ac2b61c4deeb568f995dc3ec6ca407ca00ce03d05fc2983c783d339bee59e3
- inbound_path_test.go: 47424df8b423732586331dfe7b909f7480cc1511720a55aa2d17cd4f85690d99

Backend-to-client forwarding, continuous duplex operation and CLI wiring remain pending.
