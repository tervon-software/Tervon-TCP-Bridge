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
