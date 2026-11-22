# Implementation status — 2026-11-15

**DONE:** Public repository established; Go HTTP/2 relay library and isolated tests imported.

**TESTED locally:** Go 1.23.2 on Linux amd64: `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...` passed in a local extracted working tree.

**NOT TESTED:** CI for GitHub HEAD, half-close, early EOF, resource bounds, identity-to-service authorization, real VPN protocols, real Homeserver to server 6 transport, DPI behavior and throughput.

**BLOCKED for release:** License selection, hardening, executable entry points, configuration and real route tests.

**Production impact:** None. No server settings changed. This is a research baseline only.

**Next:** Add failing regression tests for half-close and backend early EOF, implement bounded cancellation and enforce authenticated identity-to-service authorization. Every verification must answer a release-gating question.

## 2026-11-15 — identity authorization increment

- Added SHA-256 client leaf certificate fingerprint to allowed service mapping; default deny.
- Added regression test for valid mTLS identity requesting an unauthorized service (HTTP 403).
- Fixed lazy round-robin map initialization under mutex.
- Local Go 1.23.2: unit tests, race detector, vet passed after changes.
- GitHub Actions result for latest commit not yet confirmed.
- Remaining lifecycle issues: half-close, early EOF, cancellation and resource limits. Do not deploy.

## Lifecycle regression increment

- Published `prototype/lifecycle_test.go` with real TCP half-close and backend early-EOF regression scenarios.
- Equivalent local tests passed on Go 1.23.2 with `go test`, `go test -race` and `go vet`.
- These tests do not establish complete cancellation safety, bounded goroutine lifetimes or resource limits. GitHub CI status for the published commit has not been verified.
- No server deployment or production change.

## Source integrity correction

On 2026-11-15 a review of GitHub HEAD found prototype/bridge.go malformed: a prior patch had appended a second fragment after the file's end, so the published source could not compile. Commit 3a71e43 replaced the file with one coherent Go source file and preserved the certificate-fingerprint service authorization. This is a **source repair**, not evidence that GitHub CI passed. The previous PASS statements for the published revision are superseded until CI or an exact checkout is tested. Full lifecycle cancellation and resource bounding remain unimplemented. Do not deploy.

## 2026-11-15 — backend EOF bounded-handler patch

- Reviewed the current GitHub source before editing; confirmed the HTTP handler awaited the upload goroutine indefinitely after backend EOF.
- Commit ec1c9f1 closes request body and backend socket after response EOF, and bounds handler waiting to 500 ms.
- An equivalent patch to the local Go 1.23.2 working copy passed unit tests, race detector, and vet. **The exact published commit has not been executed locally or confirmed by CI.**
- IMPORTANT: bounded handler waiting is **not** proof of bounded goroutine lifetime. Request-body read cancellation and shutdown behavior require further adversarial tests. Do not deploy.

## 2026-11-15 — concurrent admission gate

- Commit 1d60bcf adds optional `Server.MaxConcurrent` semaphore for admitted HTTP/2 streams; excess requests receive HTTP 503 before backend dialing.
- Commit eb28c39 adds `TestStreamCapacityRejectsBeforeBackendDial`.
- IMPORTANT: admission occurs after mTLS-chain validation but before fingerprint service authorization; this must be reconsidered to prevent unauthorized clients consuming stream slots.
- IMPORTANT: zero means unlimited, and the semaphore is initialized once: changing MaxConcurrent at runtime is unsupported. No per-identity quotas or total HTTP connection cap exist.
- **Neither GitHub CI nor exact-commit local execution is confirmed for these changes.** They are unverified source changes, not a completed security milestone.
- Request-body cancellation and active-session shutdown remain unresolved. No production deployment.

## 2026-11-15 — cancellation-aware backend dialing

- Commit f7bfd62 changes backend selection to `pick(ctx, service)`, uses standard-library `net.Dialer.DialContext`, and checks cancellation before each fallback attempt.
- Commit 41c58cd adds tests for a pre-cancelled context and cancelled fallback.
- This only covers **dial and fallback cancellation**, not stream upload/download goroutine cleanup, shutdown, idle timeouts, or bounded total HTTP connections.
- Exact published commit CI status **not verified**. The GitHub connector's workflow-run lookup only covers PR-triggered runs; empty results cannot establish whether push CI ran. Container git access to github.com is unavailable due to DNS.
- No production deployment.

## 2026-11-15 — grouped lifecycle review

- Updated client response-EOF path to close local TCP, request pipe, and context, then join the upload goroutine (commit 4eb7f43). Previously the handler could return with the upload still reading the local connection.
- Added regression `TestEarlyBackendEOFWithOpenClientUpload` (commit 0f9b687).
- Equivalent client EOF change and regression passed Go 1.23.2 unit tests, race detector, and vet on the local working copy. The working copy is **not a byte-for-byte checkout of the current GitHub tree**, so these are not CI results.
- Reviewed formatting gate: repository has previously unformatted test files; formatting remains diagnostic, not a passing strict gate.
- Remaining release blockers: exact-commit CI verification; server-side request-body goroutine cancellation proof; bounded shutdown when sessions refuse to stop; per-client limits; robust config/CLI and operational TLS setup. **Do not deploy.**

## 2026-11-15 — bounded listener shutdown

- Encapsulator `Client.ListenAndServe` now uses `ShutdownTimeout` when waiting for active sessions: default 5 seconds, negative means unbounded. An exceeded deadline returns an explicit error rather than claiming a clean shutdown (commit 2e910b7).
- Added tests for timeout and successful drain in `prototype/shutdown_timeout_test.go` (commit 321accf).
- Locally verified the same wait helper and tests with Go 1.23.2: `go test`, `go test -race`, and `go vet` passed. This does **not** establish PASS for the complete published commit.
- **Important:** the deadline bounds how long the listener waits; it does not forcibly kill a blocked HTTP transport or prove all goroutines exited. Timeout must be treated as an incomplete shutdown. Exact-commit CI and end-to-end shutdown tests remain release blockers.

## 2026-11-15 — local TCP admission

- Added `Client.MaxLocalConnections` to cap local TCP sessions before spawning HTTP forwarding goroutines. Excess accepted sockets are immediately closed (commit e9af50c).
- Added `TestLocalConnectionAdmissionCap` to verify rejection and shutdown (commit 9b49df2).
- Zero still means unlimited, for experimental compatibility; production must explicitly configure a positive cap. This is a total local connection cap, **not** per-IP, per-identity, or rate limiting.
- The full published revision has not yet been run through CI; do not interpret source publication as a passing test. The implementation still lacks a production executable/configuration and complete end-to-end validation.

## 2026-11-15 — CLI configuration hardening

- Experimental `cmd/bridge` now rejects malformed/empty certificate SHA-256 fingerprints, authorization to services without routes, and empty backend addresses at configuration load time. Client URL query parameters are disallowed. Invalid configuration must fail before opening the listener.
- Server shutdown timeout now produces nonzero process exit instead of logging a warning and returning success.
- Added negative authorization configuration regression tests.
- **Verification gap:** GitHub Actions result for this exact tree remains unverified. The runtime CLI has not been tested with two real VPS or a production PKI. Do not deploy.

## 2026-11-15 — Homeserver independent test evidence (user-reported)

- Host: Ubuntu 26.04.1 x86_64, Go 1.25.5. Verified source commit `d06bf68a3aae17c0e598191ed15742d2c62367a4`.
- `go build ./...`, `go vet ./...`, `go test -count=1 -timeout=120s ./...` and `go test -race -count=1 -timeout=180s ./...`: PASS per host console output. Earlier commit `6fb7193` had duplicate `strings` import in CLI tests, fixed in `d06bf68`.
- Loopback integration with ephemeral test CA and separate server/client leaf certificates, Encapsulator/Decapsulator and TCP echo backend: 262144 bytes sent and received, SHA-256 matched; PASS.
- Eight concurrent loopback sessions, 131072 bytes each (1048576 bytes total): all eight byte-exact PASS. systemd transient unit `tervon-bridge-parallel-test.service` ended `Result=success`, `ExecMainStatus=0`.
- An earlier SSH console closed unexpectedly; test subprocesses were orphaned and manually terminated. No evidence of host OOM or sshd crash in inspected logs. Cause of console closure not established.
- These results are user-provided console evidence, not independently retrieved CI artifacts. The loopback test does not prove multi-host connectivity, fault recovery, throughput, or production readiness.

## 2026-11-15 — Backend outage/recovery integration evidence (user-reported)

- On isolated Homeserver loopback test, healthy backend echoed 65536 bytes byte-exact.
- After stopping the backend, a new connection returned no backend data and completed in 0.008 s (test-defined success; exact HTTP/2 error classification not asserted).
- After restarting the backend, another new connection echoed 65536 bytes byte-exact without restarting the Encapsulator or Decapsulator.
- Transient systemd test unit `tervon-bridge-recovery-test.service` exited successfully: `ActiveState=inactive`, `Result=success`, `ExecMainStatus=0`.
- This verifies new-connection recovery only; interrupted established TCP sessions are not automatically resumed, and inter-host connectivity remains untested.

## 2026-11-15 — Active backend process kill (user-reported)

- Controlled loopback backend acknowledged a live TCP tunnel, then its process was terminated with SIGKILL.
- Client observed EOF (zero bytes), completing 1.966 seconds after entering the blocking receive; both Encapsulator and Decapsulator processes remained alive.
- `tervon-bridge-active-drop-test.service`: `ActiveState=inactive`, `Result=success`, `ExecMainStatus=0`.
- Caveat: 1.966 seconds includes the test's deliberate approximately 2-second delay before backend kill; this is **not** measured bridge failover latency. EOF is expected session termination, not restoration of an interrupted TCP stream. Resource leaks and post-failure new-session success were not measured in this particular test.
