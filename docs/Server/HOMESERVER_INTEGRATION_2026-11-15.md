# Homeserver integration evidence and reproducibility

The integration scripts executed on Homeserver on 2026-11-15 were created locally under `/opt/tervon-tcp-bridge-lab/`. **Their exact contents have not been committed**. This file documents the observed results; it is not a substitute for version-controlled test scripts.

## Verified output (user-provided console logs)

- Go 1.25.5, commit `d06bf68a3aae17c0e598191ed15742d2c62367a4`: build, vet, unit tests and race tests PASS.
- Single loopback TCP echo session over HTTP/2 + mTLS: 262144 bytes, SHA-256 identical.
- 8 concurrent echo sessions: 131072 bytes each, 1048576 bytes total; all PASS.
- Backend outage: initial 65536-byte echo PASS, failed backend connection completed in 0.008s without data, restarted backend then 65536-byte echo PASS.
- Established-session backend SIGKILL: client observed EOF; both bridge processes survived. Reported 1.966s includes deliberate approximately 2s wait before killing backend, so **not** failure detection latency.
- Resource check: 10 warm-up sessions and 100 additional byte-exact 32768-byte sessions PASS. File descriptors before/after: server 7/7, client 7/7. RSS KiB: server 11224 to 14920; client 12240 to 14784. RSS growth alone is not a leak finding.

## Local-only artifacts (not uploaded)

- `parallel-test.sh`
- `recovery-test.sh`
- `active-drop-test.sh`
- `resource-test.sh`
- ephemeral CA and leaf certificates in the local `e2e-20261115-175152` folder

These scripts should be generalized to create temporary certificates, use dynamically assigned loopback ports, enforce cleanup and bounded timeouts, and then committed with their own CI coverage. Do not commit local private keys, production addresses, or secrets.

## Remaining verification

Reproducible checked-in end-to-end harness, post-failure resource/goroutine observation, sustained load, real two-VPS test, security review and exact-commit CI evidence. Do not treat the loopback results as production approval.

## Reproducible harness — 2026-11-15 (user-reported PASS)

- Source revision: `924f2833888aa6c7e6a21a6f44a5665053ad64ff`; `tests/homeserver/run-all.sh` builds the Go binary, generates temporary CA and separate mTLS leaf certificates, writes configs, and invokes the four checked-in tests.
- `tervon-bridge-suite-v2.service` completed successfully in 19.619s wall time (1min 7.393s aggregate CPU time); peak systemd unit memory 465.7M includes test build and subprocesses, not Bridge RSS alone.
- Parallel: eight 131072-byte streams, 1048576 bytes total, PASS.
- Recovery: 65536-byte echo before and after backend restart PASS; unavailable backend request finished in 0.011s, PASS.
- Active backend kill: established session received EOF; both Bridge processes survived, PASS. Reported 1.966s includes the intentional 2s pre-kill delay.
- Resources: 10 warmup + 100 byte-exact 32768-byte sessions PASS. Server FD 7 -> 7, client FD 7 -> 7. Server RSS 12084 -> 15836 KiB, client RSS 12900 -> 16808 KiB. This short run does not establish absence of memory/goroutine leaks.
- The scripts still use fixed loopback ports 18443, 19000, 19001; harness checks that they are unoccupied. Dynamic ports and prolonged-load tests remain open.

## Stress test — 500 sessions (user-reported PASS)

- Harness commit `93d056782909a20ca14beaef624dff178499468c`; transient unit `tervon-bridge-stress-v1.service` completed with `Result=success`, `ExecMainStatus=0`.
- Ten batches of 50 TCP echo sessions, eight concurrent workers, 65536 bytes per session: 500/500 byte-exact PASS, 32768000 bytes (31.25 MiB) total, 7.48s for the transfer phase.
- Server (Decapsulator): FD 7 -> 7; RSS 12148 -> 16308 KiB (+4160 KiB). Client (Encapsulator): FD 7 -> 7; RSS 13044 -> 26092 KiB (+13048 KiB). Client RSS grew through the ten batches; investigate heap retention with longer repeated cycles and runtime metrics before claiming absence or presence of leaks.
- End-to-end unit duration 18.962s wall, 1min 7.884s CPU, 422.1M peak unit memory (includes build and test processes).
- This is a bounded stress test, not an extended soak test; no real two-VPS network testing yet.

## Repeated session cancellation / goroutine lifecycle (user-reported PASS)

- Source test `prototype/goroutine_lifecycle_test.go`, commit `3e529728397019e0f3b8b9ff8b0483e74e3b52c4`.
- Warm-up baseline 7 goroutines. Five cycles of 20 sessions: ordinary test 8,7,7,7,8; race-enabled test 7,7,7,7,8.
- `go test -run '^TestRepeatedCancellationGoroutines$' -count=1 -v -timeout=120s .` PASS (0.442s).
- `go test -race -run '^TestRepeatedCancellationGoroutines$' -count=1 -v -timeout=180s .` PASS (1.655s).
- No sustained goroutine accumulation or race report observed in this bounded scenario; not proof of leak-free long-term operation.

## Stress test — 1000 sessions (user-reported PASS)

- Stress parameterization commit `15554070f4c0647ae14a517acbc3061908441301`; `BRIDGE_STRESS_CONNECTIONS=1000` and `BRIDGE_TEST_SUITE=stress`.
- Transient unit `tervon-bridge-stress-1000.service` completed: `ActiveState=inactive`, `Result=success`, `ExecMainStatus=0`.
- 20 batches of 50 sessions with 8 workers; 65536000 byte-exact bytes (62.5 MiB) in 12.80s transfer phase. Both bridge processes survived. Whole unit: 24.655s wall, 1min 12.156s CPU, 426.4M unit peak memory (includes build/test subprocesses).
- Decapsulator FD 7 -> 7; RSS 12576 -> 16408 KiB (+3832 KiB). Encapsulator FD 7 -> 7; RSS 13304 -> 22716 KiB (+9412 KiB). Encapsulator peak sampled RSS 25308 KiB at batch 17, then decreased; no monotonic RSS accumulation observed in this bounded run.
- Result: PASS for this loopback stress scenario, not a long-term memory-leak proof or a two-VPS result.

## Long-lived HTTP/2 tunnel test — 8 sessions, 60 seconds (user-reported PASS)

- Harness commits `588f1b0bafac5f89c1356bf79aa46dc2c2094589` and `8433054e3e2cf371b79c42f7d9153778ade77a76`; unit `tervon-bridge-hold-v1.service` completed with `ActiveState=inactive`, `Result=success`, `ExecMainStatus=0`.
- Eight concurrent bidirectional TCP echo sessions remained open for the 60-second target, transferring 9797632 byte-exact bytes (approximately 9.34 MiB); both bridge processes survived.
- Active baseline server FD 15, RSS 14180 KiB; client FD 15, RSS 15948 KiB. After close server FD 7, RSS 14480 KiB (+300 KiB); client FD 7, RSS 16068 KiB (+120 KiB). All eight extra FDs released per process.
- Full unit 1min 12.820s wall, 1min 7.654s CPU, 418.3M unit peak memory (includes build/test subprocesses).
- This validates the 60-second loopback hold scenario, not multi-hour endurance or remote VPS behavior.

## Decapsulator forced termination and restart (user-reported PASS)

- Isolated `remote-failure` suite completed on Homeserver in `tervon-bridge-remote-failure-v1.service`: `ActiveState=inactive`, `Result=success`, `ExecMainStatus=0`.
- Healthy echo PASS; established active HTTP/2 tunnel; test Decapsulator killed with SIGKILL; active stream received EOF after approximately 0.001027s from the blocking receive start (not a network-failure detection SLA).
- New session while Decapsulator unavailable failed as expected; restarted test Decapsulator; fresh session recovered on attempt 1 with Encapsulator left running; both processes alive.
- Unit consumed 11.879s wall, 1min 5.980s CPU, 419.5M peak memory including build/test processes. Shell `Killed` notification is expected for intentional SIGKILL.
- Scope: process crash and restart over loopback, not a packet blackhole, route loss, or remote-VPS test.

## Loopback TCP blackhole and restoration (user-reported PASS)

- Isolated `blackhole` suite completed in `tervon-bridge-blackhole-v1.service`: `ActiveState=inactive`, `Result=success`, `ExecMainStatus=0`.
- Healthy byte-exact echo via loopback TCP proxy PASS. Proxy discarded both directions while processes remained alive. Active stream stalled as expected; no unexpected EOF or payload. After restoring forwarding, a fresh TCP stream succeeded on first attempt; both bridge processes survived.
- Unit wall 13.595s, CPU 1min 5.561s, memory peak 419.7M including build/test processes.
- Limitation: short fault-injection interval, not proof of automatic recovery/timeout of existing stalled TCP streams, long-duration outages, or real network packet loss on remote VPS.

## Full parallel batch — user-reported PASS (2026-11-15)

- `tervon-bridge-batch-v1.service`: `ActiveState=inactive`, `Result=success`, `ExecMainStatus=0`; `FULL PARALLEL BATCH PASSED`. Two lanes completed independently: integration exit=0, Go checks exit=0.
- Integration suite PASS: parallel 8 × 131072 bytes; backend recovery (failure handling 0.008s); active backend drop; resource test 100 sessions (FD 7→7 on both); stress 500 × 65536 bytes (32768000 byte-exact bytes, 7.39s transfer, FD delta 0 both); hold 8 sessions × 60s (9797632 byte-exact bytes, active FD 15→7 after close); remote Decapsulator SIGKILL/restart and first-attempt fresh connection recovery; proxy blackhole and first-attempt fresh connection recovery.
- Go lane PASS: `go build ./...`, `go vet ./...`, `go test -count=1 ./...`, `go test -race -count=1 ./...`. Go package test durations: 2.771s ordinary, 3.723s race (main package).
- Whole unit wall 1min 36.645s; CPU 4min 9.741s; peak memory 755.7M (includes build/test processes).
- Caveats: integration scenarios are sequential within their lane because they share loopback ports; Go checks run concurrently with integration. No long-term stall cleanup or remote multi-VPS validation yet.

## Pre-live heavier batch — user-reported PASS (2026-11-15)

- `tervon-bridge-prelive-v1.service`: `ActiveState=inactive`, `Result=success`, `ExecMainStatus=0`; both integration and Go lanes exit=0, `FULL PARALLEL BATCH PASSED`.
- 8 parallel streams: 1048576 byte-exact bytes. Backend failure handling 0.008s, active-drop 1.965s. Resource 100 sessions: FD 7→7.
- Stress 1000 sessions: 65536000 byte-exact bytes, elapsed 12.76s; server/client FD delta 0; RSS delta 4160/9876 KiB.
- Hold 16 connections for 120s: 39124992 byte-exact bytes; FD 23→7 after close on both processes.
- Decapsulator failure/restart and loopback TCP proxy blackhole/recovery passed; new stream recovered on first attempt in both scenarios.
- Go build, vet, unit tests and race tests PASS. Unit wall 2min 42.250s; CPU 4min 23.991s; peak memory 786.5M includes build/test processes.
- Scope: isolated loopback only. Real network packet loss, multi-host interoperability, long-term blackhole liveness and production routing not yet verified.
