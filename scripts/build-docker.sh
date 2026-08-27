#!/bin/bash
# Build Docker images.
# Usage: scripts/build-docker.sh --compose-file deploy/liki/docker-compose.yml [--export /tmp/liki_images.tar.gz] [--services web engine]
set -euo pipefail

COMPOSE_FILE=""
EXPORT=""
SERVICES=""

while [ $# -gt 0 ]; do
  case "$1" in
    --compose-file) COMPOSE_FILE="$2"; shift 2 ;;
    --export)       EXPORT="$2"; shift 2 ;;
    --services)     SERVICES="${@:2}"; break ;;
    *) echo "Usage: $0 --compose-file PATH [--export PATH] [--services SVC...]" >&2; exit 1 ;;
  esac
done

if [ -z "$COMPOSE_FILE" ]; then
  echo "ERROR: --compose-file is required" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BUILD_TIME="${BUILD_TIME:-$(date -u '+%Y-%m-%dT%H:%M:%SZ')}"

# 自动加载 .env（compose 插值需要 FEEDBACK_API_KEY 等变量；不加载 → 空串插值告警，
# 若带空变量 up 起容器，feedback list/update 认证将全 403）
if [ -f "$PROJECT_DIR/.env" ]; then
    set -a
    # shellcheck disable=SC1091
    . "$PROJECT_DIR/.env"
    set +a
fi

echo "[build-docker] build..."
cd "$PROJECT_DIR"
docker compose -f "$COMPOSE_FILE" build $SERVICES

if [ -n "$EXPORT" ]; then
  echo "[build-docker] export → $EXPORT"
  IMAGES="liki-web:latest liki-engine:latest"
  docker save $IMAGES | gzip > "$EXPORT"
fi
