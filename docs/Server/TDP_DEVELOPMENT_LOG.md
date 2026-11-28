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
