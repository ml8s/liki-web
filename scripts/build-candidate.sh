#!/usr/bin/env bash
# Build and validate the liki-web candidate in the local registry.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

[ -z "$(git status --porcelain --untracked-files=normal)" ] || {
  echo "❌ Git worktree is not clean; candidate releases require a clean tree" >&2
  exit 1
}

RELEASE_VERSION="${RELEASE_VERSION:?set RELEASE_VERSION to the selected CalVer, e.g. 2026.10.13.0}"
if [[ ! "$RELEASE_VERSION" =~ ^[0-9]{4}\.[0-9]{2}\.[0-9]{2}\.[0-9]+$ ]]; then
  echo "❌ RELEASE_VERSION must use CalVer YYYY.MM.DD.N: $RELEASE_VERSION" >&2
  exit 1
fi
version_file="$(cat cmd/liki-web/VERSION)"
if [ "$version_file" != "$RELEASE_VERSION" ]; then
  echo "❌ RELEASE_VERSION=$RELEASE_VERSION does not match cmd/liki-web/VERSION=$version_file" >&2
  exit 1
fi


make gate

[ -z "$(git status --porcelain --untracked-files=normal)" ] || {
  echo "❌ Git worktree changed during gate; aborting candidate build" >&2
  git status --short >&2
  exit 1
}
SOURCE_COMMIT="$(git rev-parse HEAD)"
SHA7="${SOURCE_COMMIT:0:7}"
BUILD_TIME="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
CANDIDATE_TAG="candidate-${SHA7}-$(date -u '+%Y%m%d%H%M%S')"

CANDIDATE_REGISTRY="${CANDIDATE_REGISTRY:-127.0.0.1:5000}"
CANDIDATE_REGISTRY_NAME="${CANDIDATE_REGISTRY_NAME:-liki-candidate-registry}"
CANDIDATE_REGISTRY_IMAGE="${CANDIDATE_REGISTRY_IMAGE:-registry:2}"

ensure_candidate_registry() {
  if curl -fsS "http://${CANDIDATE_REGISTRY}/v2/" >/dev/null 2>&1; then
    return
  fi
  if docker container inspect "$CANDIDATE_REGISTRY_NAME" >/dev/null 2>&1; then
    docker start "$CANDIDATE_REGISTRY_NAME" >/dev/null
  else
    docker run -d --name "$CANDIDATE_REGISTRY_NAME" \
      --restart unless-stopped \
      -p 127.0.0.1:5000:5000 \
      "$CANDIDATE_REGISTRY_IMAGE" >/dev/null
  fi
  for _ in $(seq 1 30); do
    curl -fsS "http://${CANDIDATE_REGISTRY}/v2/" >/dev/null 2>&1 && return
    sleep 1
  done
  echo "❌ local candidate registry did not become ready" >&2
  exit 1
}

ensure_candidate_registry
IMAGE="$CANDIDATE_REGISTRY/ml8s/liki-web:${CANDIDATE_TAG}"
METADATA_FILE="$ROOT/build/candidate/buildx-metadata.json"
MANIFEST_FILE="$ROOT/build/candidate/candidate-digests.json"

rm -rf "$ROOT/build/candidate"
mkdir -p "$(dirname "$METADATA_FILE")"
make build
docker buildx build \
  --build-arg "VERSION=${RELEASE_VERSION}" \
  --build-arg "COMMIT=${SOURCE_COMMIT}" \
  --build-arg "BUILD_TIME=${BUILD_TIME}" \
  --provenance=mode=max \
  --sbom=true \
  --metadata-file "$METADATA_FILE" \
  --tag "$IMAGE" \
  .

python3 scripts/build_candidate_manifest.py \
  --output "$MANIFEST_FILE" \
  --registry "$CANDIDATE_REGISTRY" \
  --source-commit "$SOURCE_COMMIT" \
  --candidate-tag "$CANDIDATE_TAG" \
  --metadata-file "$METADATA_FILE"

DIGEST="$(python3 -c 'import json; print(json.load(open("'"$METADATA_FILE"'"))["containerimage.digest"])')"
echo "📦 local candidate digest: $DIGEST"
IMAGE="$CANDIDATE_REGISTRY/ml8s/liki-web@$DIGEST" scripts/smoke-image.sh
echo "✅ candidate validated locally: ${IMAGE}"
