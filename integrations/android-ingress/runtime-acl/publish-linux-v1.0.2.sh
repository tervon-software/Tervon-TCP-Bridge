#!/usr/bin/env bash
# Publish the already-tested Linux v1.0.2 binary. No builds, no systemd, no Docker.
set -euo pipefail

REPO=/opt/tervon-tcp-bridge-m6-android-ingress
CANDIDATE=/opt/tervon/Developed/v1.0.2-runtime-policy-validation/tervon-bridge-linux-amd64-v1.0.2-candidate
ARTIFACT_DIR=/opt/tervon/Developed/v1.0.2-github-release
ASSET=tervon-bridge-linux-amd64-v1.0.2
SHA=aea59080e0a112e9d5c079d45667ed8cb9b06e126326a398ea33dd94fb0c1371
BASE=96c136702d10b6fb9a16aa2893f6b978e8b14e86
POLICY_SOURCE=40dc00a24fb43ea05e1adf4a62845cb5546ca1bd
TAG=v1.0.2
GH_REPO=tervon-software/Tervon-TCP-Bridge

echo "===== RELEASE PREFLIGHT ====="
command -v gh >/dev/null || { echo "BLOCKED=GITHUB_CLI_MISSING"; exit 1; }
command -v sha256sum >/dev/null || { echo "BLOCKED=SHA256SUM_MISSING"; exit 1; }
command -v git >/dev/null || { echo "BLOCKED=GIT_MISSING"; exit 1; }
test -d "$REPO"
test "$(git -C "$REPO" branch --show-current)" = "work/m6-android-ingress"
test "$(git -C "$REPO" remote get-url origin)" = "git@github.com:tervon-software/Tervon-TCP-Bridge.git"
test -z "$(git -C "$REPO" status --porcelain --untracked-files=no)"
test -f "$CANDIDATE"
echo "$SHA  $CANDIDATE" | sha256sum --check --status || { echo "BLOCKED=BINARY_HASH_MISMATCH"; exit 1; }
test "$(git -C /opt/tervon/Developed/v1.0.2-runtime-policy-validation rev-parse HEAD)" = "$BASE"
test -f /opt/tervon/Developed/v1.0.2-runtime-policy-validation/prototype/runtime_policy.go
test ! -e "$ARTIFACT_DIR" || { echo "BLOCKED=STAGING_DIR_EXISTS"; exit 1; }

# Use the same configured GitHub CLI auth; do not print credentials.
gh auth status >/dev/null 2>&1 || { echo "BLOCKED=GH_AUTH_REQUIRED"; exit 1; }
git -C "$REPO" fetch origin work/m6-android-ingress --tags
test "$(git -C "$REPO" rev-parse HEAD)" = "$(git -C "$REPO" rev-parse origin/work/m6-android-ingress)" || {
  echo "BLOCKED=LOCAL_SOURCE_NOT_LATEST"; exit 1;
}
if git -C "$REPO" ls-remote --tags origin "refs/tags/$TAG" | grep -q .; then
  echo "BLOCKED=RELEASE_TAG_ALREADY_EXISTS"; exit 1
fi
if gh release view "$TAG" -R "$GH_REPO" >/dev/null 2>&1; then
  echo "BLOCKED=RELEASE_ALREADY_EXISTS"; exit 1
fi
SOURCE_HEAD=$(git -C "$REPO" rev-parse HEAD)
test -f "$REPO/docs/RELEASE_NOTES_v1.0.2.ru.md"

echo "===== PACKAGE VERIFIED BINARY ====="
mkdir -p -m 700 "$ARTIFACT_DIR"
cp -- "$CANDIDATE" "$ARTIFACT_DIR/$ASSET"
chmod 755 "$ARTIFACT_DIR/$ASSET"
printf '%s  %s\n' "$SHA" "$ASSET" > "$ARTIFACT_DIR/SHA256SUMS"
cat > "$ARTIFACT_DIR/BUILD_INFO" <<EOF
Name: Tervon TCP Bridge Linux AMD64
Version: v1.0.2
Artifact: $ASSET
SHA256: $SHA
Base release source commit: $BASE
RuntimePolicy overlay commit: $POLICY_SOURCE
Git tag target commit: $SOURCE_HEAD
Build: Go, Linux amd64, -trimpath
Validation: go test, go test -race, go vet, isolated policy SIGHUP/ACK PASS
Production deployment: NO
EOF
(cd "$ARTIFACT_DIR" && sha256sum -c SHA256SUMS)
echo "===== CREATE GITHUB RELEASE ====="
gh release create "$TAG" \
  "$ARTIFACT_DIR/$ASSET" \
  "$ARTIFACT_DIR/SHA256SUMS" \
  "$ARTIFACT_DIR/BUILD_INFO" \
  -R "$GH_REPO" \
  --target "$SOURCE_HEAD" \
  --title "Tervon TCP Bridge v1.0.2 — Linux amd64" \
  --notes-file "$REPO/docs/RELEASE_NOTES_v1.0.2.ru.md" \
  --latest

echo "===== VERIFY PUBLISHED ASSET ====="
RELEASE_URL=$(gh release view "$TAG" -R "$GH_REPO" --json url --jq .url)
ASSET_DIR=$(mktemp -d -p /opt/tervon/Developed v102-published-XXXXXX)
trap 'rm -rf -- "$ASSET_DIR"' EXIT
gh release download "$TAG" -R "$GH_REPO" -p "$ASSET" -D "$ASSET_DIR"
echo "$SHA  $ASSET_DIR/$ASSET" | sha256sum -c
echo "RELEASE_URL=$RELEASE_URL"
echo "RELEASE_ASSET_SHA256=PASS"
echo "LINUX_V1_0_2_GITHUB_RELEASE=PASS_PUBLISHED"
echo "PRODUCTION=UNCHANGED"
