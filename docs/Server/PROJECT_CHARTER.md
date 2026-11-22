# Tervon TCP Bridge — Project Charter

Approved direction: 2026-11-15.

## Mission

Build a high-throughput secure TCP-over-HTTPS transport designed to minimize distinctive observable traffic features in filtered networks. Claims of traffic indistinguishability must be supported by route-specific experiments, not assumptions.

## Architecture

Client -> local Encapsulator -> authenticated HTTP/2 over TLS 1.3 -> remote Decapsulator -> server-allowlisted TCP backend.

The bridge transports opaque bidirectional bytes and does not parse inner VPN protocols. Backend selection is pinned for each connection; a new connection may select another healthy backend. Half-close, cancellation, timeouts and failures must be handled explicitly.

## Security

Use maintained standard libraries for HTTP, TLS and authentication. Bind authenticated client identities to permitted service identifiers. Reject arbitrary destinations and unsafe redirects. Apply connection and resource limits, avoid logging user payloads, and protect secrets. Production services remain untouched by laboratory tests.

## v0.1 exclusions

Native UDP, WireGuard, Hysteria2, CONNECT-UDP, CDN, uTLS, Cloak and Meek. No seamless migration of live TCP sessions.

## Acceptance gates

1. Byte-for-byte full duplex fidelity, parallel sessions, half-close.
2. Authentication, service authorization and backend allowlisting.
3. Failure handling for both pre-connect and active sessions.
4. Measured connection setup, throughput, CPU and memory.
5. Separate tests for each real TCP VPN configuration.
6. Route-specific filtering observations with explicit methodology and limitations.

## Public development

Document reproducible evidence and limitations. Review code for secrets before publication. Add a verified compatible license before advertising the code as open-source licensed. Never publish live infrastructure configuration or private credentials.
