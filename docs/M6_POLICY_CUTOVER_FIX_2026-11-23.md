# M6 Policy Cutover — ACK / Registrar Fix

Date: 2026-11-23

Repository: tervon-software/Tervon-TCP-Bridge
Branch: work/m6-android-ingress

## Scope

Correction of the existing M6 policy cutover procedure.

- Docker logs: read stdout and stderr.
- ACK: require exact post-signal policy generation digest.
- Registrar: tolerate an absent transient systemd unit.
- Preserve atomic policy update and rollback.
- No changes to Bridge core, Android, TLS or mTLS.

## Files

- integrations/android-ingress/stage4/m6_policy_cutover.py
- integrations/android-ingress/stage4/test_m6_policy_cutover.py

## Validation

13 isolated Python tests covering:

- Docker ACK parsing
- stale and incorrect ACK rejection
- registrar unit handling
- generation digest
- successful simulated cutover
- failed ACK with rollback
- rollback failure reporting

Tests use temporary policy files and mocked external calls.

## Status

ISOLATED_TESTS: PASS (13/13; 0 failures; 0 errors)
LIVE_CUTOVER: NOT_RUN
ANDROID_E2E: NOT_TESTED
RUNNING_SERVICES: UNCHANGED

## Limitations

Real Docker SIGHUP timing and systemd transient-unit
recreation have not been validated.

The registrar binary path and Docker image identity
must be revalidated before live use.

The script requires explicit --apply.
Do not run --apply without separate approval.

## Next step

Complete isolated validation, record the resulting commit,
then perform read-only Homeserver preflight.

## Homeserver validation — 2026-11-23

Environment: Homeserver, Python 3, isolated unittest.

Results:
- Syntax: PASS
- Isolated tests: 13/13 PASS
- Failures: 0
- Errors: 0
- Simulated policy update: PASS
- Simulated ACK failure and rollback: PASS
- Simulated rollback failure detection: PASS
- Registrar present/missing handling: PASS

Limitations:
- Docker and systemd calls were mocked.
- Live policy update was not executed.
- Registrar recovery was not tested against systemd.
- Android E2E remains NOT_TESTED.
- Live preflight requires active Registrar; previously
  observed systemd LoadState was not-found.

Working services and live configuration were unchanged.
