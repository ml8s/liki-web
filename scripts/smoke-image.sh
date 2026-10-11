#!/usr/bin/env bash
# Start the locally built web image with deterministic non-secret fixture env
# and verify its health plus a rendered application page.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IMAGE_TAG="${IMAGE_TAG:-dev}"
IMAGE="${IMAGE:-ghcr.io/ml8s/liki-web:$IMAGE_TAG}"
PORT="${WEB_SMOKE_PORT:-18081}"
NAME="liki-web-smoke-$$"

cleanup() {
  docker rm -f "$NAME" >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker rm -f "$NAME" >/dev/null 2>&1 || true
docker run -d --name "$NAME" \
  -p "$PORT:8081" \
  --tmpfs /var/lib/liki:uid=1000,gid=1000 \
  -e LISTEN_ADDR=:8081 \
  -e JWT_SECRET=web-image-smoke-jwt-secret-0123456789abcdef \
  -e REFERRAL_PAYOUT_SECRET=web-image-smoke-referral-payout-secret-32-bytes \
  -e RETURN_URL=http://127.0.0.1:$PORT \
  -e PAYMENTS_WEBHOOK_URL=http://127.0.0.1:$PORT/api/payments/webhook \
  -e XUNHU_APPID=web-image-smoke \
  -e XUNHU_APPSECRET=web-image-smoke-secret \
  -e LIKI_AGENTS_URL=http://127.0.0.1:9 \
  -e LIKI_AGENTS_INTERNAL_TOKEN=web-image-smoke-agent-token \
  -e LIKI_DATA_DIR=/var/lib/liki \
  "$IMAGE" >/dev/null

ready=false
for _ in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:$PORT/api/health" >/dev/null 2>&1; then
    ready=true
    break
  fi
  if [ "$(docker inspect -f '{{.State.Running}}' "$NAME" 2>/dev/null || true)" != true ]; then
    break
  fi
  sleep 1
done

if [ "$ready" != true ]; then
  echo "❌ liki-web image did not become ready" >&2
  docker logs "$NAME" >&2 || true
  exit 1
fi

curl -fsS "http://127.0.0.1:$PORT/api/health" | grep -q '"ok"'
curl -fsS "http://127.0.0.1:$PORT/zh-Hans/" | grep -q '<html'
echo "✅ liki-web image smoke passed"
