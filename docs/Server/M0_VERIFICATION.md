# M0 — Verification and integration status

## Status

IN_PROGRESS — integrated local candidate, not a release.

Base branch: main
Base commit: 97a34b50e188adbd43c56470aeff3faa327a2e9a

Environment: Homeserver, isolated directory
Go: 1.25.5 linux/amd64

## Implemented in local candidate

- Local read-only HTTP telemetry endpoint
- Configurable loopback-only metrics listener
- Session and application-byte counters
- Telemetry integration in client and server
- Telemetry concurrency and HTTP endpoint tests
- Existing TLS 1.3 / mTLS / HTTP/2 TCP bridge

These changes are not yet committed to GitHub.

## Verified

- go vet ./... — PASS
- go test -race -count=1 ./... — PASS
- go build -trimpath ./cmd/bridge — PASS
- Selected TCP/HTTP2/mTLS regression tests, 5 repetitions — PASS
- Selected telemetry HTTP tests, 5 repetitions — PASS
- git diff --check — PASS

Candidate binary:
SHA256 db8b2c3580dad5859feb989568eb3431c9222df66780a94fd10ec3b2bf336b93

## Remaining before M0 PASS

- Full review of integrated changes
- Separate-process TLS 1.3/mTLS/HTTP2 TCP E2E
- Verify real upload/download metrics on both sides
- Verify unauthorized connection rejection
- Verify process shutdown and cleanup
- Update documentation with actual E2E evidence
- Commit via GitHub SSH
- Verify committed revision

## Safety

No production service installation, restart, firewall
modification or public metrics exposure authorized.

M1/M2/M3 must not begin before confirmed M0 PASS.

## Separate-process loopback E2E — 2026-11-17

Status: PASS (local isolated environment)

Environment:
- Homeserver, linux/amd64
- Two independent Bridge processes
- Local TCP backend
- Ephemeral test CA and mTLS certificates
- Loopback-only listeners

Verified:
- Bidirectional byte-exact TCP transfer: 1,048,576 bytes each way
- TCP half-close: PASS
- Client upload/download metrics: 1,048,576 / 1,048,576
- Server upload/download metrics: 1,048,576 / 1,048,576
- Active sessions after transfer: 0 on both sides
- TLS 1.3 negotiation: PASS
- ALPN h2 negotiation: PASS
- Unauthenticated TLS connection rejected: PASS
- Test process cleanup: PASS
- Final test exit code: 0

Test failures resolved:
1. Repeated port binding failed with Errno 98.
   The test harness was corrected.
2. Python/OpenSSL rejected the original test CA
   because its key usage extensions were incomplete.
   The test CA was regenerated with CA:TRUE and
   keyCertSign/cRLSign.
3. After certificate regeneration, the server's
   authorized fingerprint was stale.
   Updating the SHA-256 fingerprint restored access.

Limitations:
- This PASS covers local separate-process E2E only.
- Real two-host connectivity has not been verified.
- No production deployment was performed.
- M0 remains IN_PROGRESS until all plan criteria pass.

## Candidate telemetry extension — isolated source audit

- Added: build fields, readiness, configured limits, runtime allocated memory,
  backend dial success/failure, bounded dial latency distribution, and verified
  server-side TLS1.3/h2/mTLS request counter.
- Local extracted-source checks: `go test -race ./...`, `go vet ./...` PASS.
- Important: the extracted archive lacks the repository's original Go test files.
  Therefore these checks do NOT constitute full repository regression PASS.
- Pending: apply to the existing working tree, full repository race tests,
  separate-process E2E rerun, TLS failure accounting, HTTP/2 active session
  metric, fixed-window throughput, backend identity-safe availability,
  and real two-host trial. M0 remains IN_PROGRESS.


## Extended telemetry contract E2E — 2026-11-17

Status: PASS — isolated loopback test, not production deployment.

Candidate SHA-256:
`fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c`

Verified:
- Two independent Bridge processes
- Byte-exact TCP transfer: 1,048,576 bytes in each direction
- TCP half-close
- TLS 1.3 and ALPN h2
- Rejection of unauthenticated connection
- Extended `/metrics/v1` JSON contract on both processes
- Readiness true and max_sessions 16
- Positive process memory measurements
- Server backend_dial_success 2
- Server connect_latency_samples 2
- Server connect_latency p50/p95/p99: 1/1/1 ms (bucket estimates)
- Server tls13_h2_mtls_requests 2
- E2E exit code 0
- Test binary restoration exit code 0

Build metadata currently reports dev/unknown.
The candidate must be rebuilt with actual version/commit metadata.

Remaining:
- Complete outstanding M0 monitoring and failure-path checks
- Review and reconcile README with actual implementation
- Commit verified changes through GitHub SSH
- Perform separately approved two-host operational verification

Overall M0 status: IN_PROGRESS.

## Production deployment — 2026-11-17

- Finserver: оба Decapsulator процесса — DEPLOYMENT PASS.
- Главный сервер каскада: Encapsulator — DEPLOYMENT PASS.
- SHA-256 кандидата:
  `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c`.
- Реальный пользовательский трафик: доступность подтверждена.
- Оценка отзывчивости положительная, количественный benchmark
  не проводился.
- Production telemetry endpoint отдельно не подтверждён.
- Резервные копии исходных бинарников сохранены.
- M0: эксплуатационный PASS с указанными ограничениями.
