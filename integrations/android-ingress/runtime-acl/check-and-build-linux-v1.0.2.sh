#!/usr/bin/env bash
# Isolated v1.0.2 Linux kernel candidate: tests are mandatory build gates.
# Never installs, signals, restarts or edits a running Bridge service.
set -euo pipefail

REPO=/opt/tervon-tcp-bridge-m6-android-ingress
BASE=96c136702d10b6fb9a16aa2893f6b978e8b14e86
STAGE=/opt/tervon/Developed/v1.0.2-runtime-policy-validation
OVERLAY=integrations/android-ingress/runtime-acl/apply-runtime-policy-to-release-source.py
OUTPUT="$STAGE/tervon-bridge-linux-amd64-v1.0.2-candidate"

echo "===== SOURCE GATE ====="
test "$(git -C "$REPO" branch --show-current)" = "work/m6-android-ingress"
test "$(git -C "$REPO" remote get-url origin)" = "git@github.com:tervon-software/Tervon-TCP-Bridge.git"
test -z "$(git -C "$REPO" status --porcelain --untracked-files=no)"
SOURCE=$(git -C "$REPO" rev-parse HEAD)
test "$SOURCE" = "$(git -C "$REPO" rev-parse origin/work/m6-android-ingress)"
test -n "$SOURCE"
git -C "$REPO" cat-file -e "$BASE^{commit}"
test ! -e "$STAGE"
test "$(go env GOHOSTOS)" = "linux"
test "$(go env GOHOSTARCH)" = "amd64"

echo "SOURCE_HEAD=$SOURCE"
echo "RELEASE_BASE=$BASE"
echo "===== STAGE VERIFIED RELEASE SOURCE ====="
git -C "$REPO" worktree add --detach "$STAGE" "$BASE"
test "$(git -C "$STAGE" rev-parse HEAD)" = "$BASE"

echo "===== GUARDED OVERLAY PREFLIGHT ====="
python3 "$REPO/$OVERLAY" --release-root "$STAGE" --check

echo "===== APPLY CANDIDATE SOURCE ONLY ====="
python3 "$REPO/$OVERLAY" --release-root "$STAGE" --apply

cd "$STAGE/prototype"
gofmt -w bridge.go cmd/bridge/main.go runtime_policy.go runtime_policy_test.go runtime_policy_stream_test.go
echo "===== GO TEST ====="
go test -count=1 -timeout=240s ./...
echo "GO_TEST=PASS"
echo "===== GO RACE ====="
go test -race -count=1 -timeout=300s ./...
echo "GO_RACE=PASS"
echo "===== GO VET ====="
go vet ./...
echo "GO_VET=PASS"

echo "===== COMPILE CANDIDATE ONLY AFTER ALL PASS ====="
go build -trimpath -ldflags "-X main.buildVersion=1.0.2 -X main.buildCommit=$SOURCE" -o "$OUTPUT" ./cmd/bridge
test -s "$OUTPUT"
test "$(file -b "$OUTPUT" | grep -c 'ELF 64-bit')" -eq 1
sha256sum "$OUTPUT"
echo "CANDIDATE_PATH=$OUTPUT"
echo "LINUX_V1_0_2_BUILD=PASS_CANDIDATE_ONLY"
echo "PRODUCTION=UNCHANGED"
echo "RELEASE=NO"
