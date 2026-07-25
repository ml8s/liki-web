#!/bin/bash
# Build the liki-web Go binary with ldflags injection.
# Usage: scripts/build-go.sh [--output bin/liki-web] [--goos linux] [--goarch amd64] [--version 0.3.16]
set -euo pipefail

OUTPUT="${OUTPUT:-bin/liki-web}"
GOOS="${GOOS:-}"
GOARCH="${GOARCH:-}"
VERSION="${VERSION:-}"
BUILD_TIME="${BUILD_TIME:-}"

while [ $# -gt 0 ]; do
  case "$1" in
    --output)     OUTPUT="$2"; shift 2 ;;
    --goos)       GOOS="$2"; shift 2 ;;
    --goarch)     GOARCH="$2"; shift 2 ;;
    --version)    VERSION="$2"; shift 2 ;;
    --build-time) BUILD_TIME="$2"; shift 2 ;;
    *) echo "Usage: $0 [--output PATH] [--goos OS] [--goarch ARCH] [--version VER] [--build-time TIME]" >&2; exit 1 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR"

# Defaults
if [ -z "$VERSION" ]; then
  VERSION=$(cat cmd/liki/VERSION 2>/dev/null || echo "dev")
fi
if [ -z "$BUILD_TIME" ]; then
  BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
fi

export CGO_ENABLED=0
[ -n "$GOOS" ]   && export GOOS
[ -n "$GOARCH" ] && export GOARCH

echo "[build-go] $OUTPUT (version=$VERSION, build=$BUILD_TIME)"
go build -ldflags="-s -w -X main.BuildTime=$BUILD_TIME -X main.Version=$VERSION" -o "$OUTPUT" ./cmd/liki/
