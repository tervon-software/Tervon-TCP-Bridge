# Tervon TCP Bridge 1.0.2-TDP

Branch: work/tdp. Base: aa7c2cbd2a17a64fb6c423bd7abd9ac8050fb799.

Approved first milestone: retain the active backend connection when the QUIC/UDP path fails, resume over TCP, and verify the complete payload by checksum. A short pause is acceptable.

This commit adds a bounded replay buffer for one stream direction and tests cumulative acknowledgments and resumption from a later offset. The two directions will need separate state. The current HTTP/2 handler ties the backend connection to the request context, so the TDP implementation will need to keep the backend alive across path changes and bind resumption to the authenticated client.

Package check: go test -race ./tdp and go vet ./tdp on Linux amd64, Go 1.23.12, passed on the staged sources. QUIC, TCP transport integration and the network acceptance test are pending. Packet manipulation is outside this milestone.
