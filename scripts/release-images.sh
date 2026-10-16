#!/usr/bin/env bash
# Build, smoke, and push the liki-web release image for a CalVer release tag.
#
# Invoked by release.yml on a pushed vYYYY.MM.DD.N tag after `make gate`.
# Requires a logged-in GHCR session. Emits digest outputs for the workflow.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

RELEASE_VERSION="${RELEASE_VERSION:?set RELEASE_VERSION to the CalVer version without the v prefix (e.g. 2026.10.16.0)}"
if ! [[ "$RELEASE_VERSION" =~ ^[0-9]{4}\.[0-9]{2}\.[0-9]{2}\.[0-9]+$ ]]; then
  echo "❌ RELEASE_VERSION must be CalVer YYYY.MM.DD.N: $RELEASE_VERSION" >&2
  exit 1
fi
version_file="$(cat cmd/liki-web/VERSION)"
if [ "$version_file" != "$RELEASE_VERSION" ]; then
  echo "❌ RELEASE_VERSION=$RELEASE_VERSION does not match cmd/liki-web/VERSION=$version_file" >&2
  exit 1
fi
RELEASE_TAG="v$RELEASE_VERSION"
SOURCE_SHA="$(git rev-parse HEAD)"
TAG_SHA="$(git rev-parse --verify "refs/tags/$RELEASE_TAG^{commit}" 2>/dev/null || true)"
if [ "$TAG_SHA" != "$SOURCE_SHA" ]; then
  echo "❌ checked-out commit is not the $RELEASE_TAG tag ($SOURCE_SHA)" >&2
  exit 1
fi

BUILD_TIME="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
OUT_DIR="$ROOT/build/release"
rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

REGISTRY="ghcr.io/ml8s"

make build

docker buildx build \
  --file Dockerfile \
  --metadata-file "$OUT_DIR/web.json" \
  --provenance=mode=max \
  --sbom=true \
  --label "org.opencontainers.image.revision=$SOURCE_SHA" \
  --label "org.opencontainers.image.version=$RELEASE_TAG" \
  --build-arg "VERSION=$RELEASE_VERSION" \
  --build-arg "COMMIT=$SOURCE_SHA" \
  --build-arg "BUILD_TIME=$BUILD_TIME" \
  --tag "$REGISTRY/liki-web:$RELEASE_TAG" \
  --tag "$REGISTRY/liki-web:latest" \
  --push \
  .

DIGEST="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["containerimage.digest"])' "$OUT_DIR/web.json")"

echo "── release smoke ──"
IMAGE="$REGISTRY/liki-web@$DIGEST" scripts/smoke-image.sh

python3 scripts/build_candidate_manifest.py \
  --output "$OUT_DIR/candidate-digests.json" \
  --registry "ghcr.io" \
  --source-commit "$SOURCE_SHA" \
  --candidate-tag "$RELEASE_TAG" \
  --metadata-file "$OUT_DIR/web.json"

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  {
    echo "web_digest=$DIGEST"
    echo "manifest=$OUT_DIR/candidate-digests.json"
  } >> "$GITHUB_OUTPUT"
fi

echo "✅ release image pushed: $RELEASE_TAG"
