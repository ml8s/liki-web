#!/bin/bash
# Docker 测试环境启停
#   docker-test.sh up    构建镜像 + 启动容器 + 等待就绪
#   docker-test.sh down  停止容器 + 清理
set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DEV_DIR="$PROJECT_DIR/dev"
COMPOSE_FILE="$DEV_DIR/docker-compose.yml"
TEST_ENV_FILE="${LIKI_TEST_ENV_FILE:-/tmp/liki-web-e2e.env}"
export LIKI_TEST_ENV_FILE="$TEST_ENV_FILE"
# Compose interpolation must also succeed for `down` when the env file is gone.
export LIKI_AGENTS_INTERNAL_TOKEN="${LIKI_AGENTS_INTERNAL_TOKEN:-e2e-agent-secret}"

CMD="${1:-up}"

do_up() {
  if [ -z "${LIKI_ENGINE_IMAGE:-}" ]; then
    cat >&2 <<'EOF'
LIKI_ENGINE_IMAGE is required for Docker E2E.

Set it to an immutable, verified engine release digest, for example:

  LIKI_ENGINE_IMAGE=ghcr.io/ml8s/liki-engine@sha256:<digest> make e2e

This repository intentionally does not keep a second default engine matrix.
The production combination remains owned by liki-deploy.
EOF
    exit 1
  fi
  if ! printf '%s' "$LIKI_ENGINE_IMAGE" | grep -Eq '@sha256:[0-9a-f]{64}$'; then
    echo "LIKI_ENGINE_IMAGE must be an immutable sha256 digest reference: $LIKI_ENGINE_IMAGE" >&2
    exit 1
  fi

  trap 'rm -f "$TEST_ENV_FILE"' EXIT

  # ── Build ──
  echo "==> 构建..."
  BUILD_TIME=$(date '+%Y-%m-%dT%H:%M:%S')
  export BUILD_TIME
  BUILD_TIME="$BUILD_TIME" "$SCRIPT_DIR/build-web.sh"
  BUILD_TIME="$BUILD_TIME" "$SCRIPT_DIR/build-go.sh" --goos linux --goarch amd64 --output "$PROJECT_DIR/bin/liki-web"

  # ── Start ──
  echo "==> 启动容器..."

  # E2E uses a self-contained environment. It must not inherit production
  # provider credentials or depend on optional services on the developer host.
  cat > "$TEST_ENV_FILE" << 'EOF'
JWT_SECRET=e2e-jwt-secret-0123456789abcdefghijklmnop
XUNHU_APPID=e2e
XUNHU_APPSECRET=e2e
REFERRAL_PAYOUT_SECRET=e2e-referral-payout-secret-0123456789abcdef
DOMAIN=localhost
LISTEN_ADDR=:8081
RETURN_URL=http://localhost:8080/api/payments/return
PAYMENTS_WEBHOOK_URL=http://localhost:8080/api/payments/webhook
LIKI_AGENTS_INTERNAL_TOKEN=e2e-agent-secret
EOF

  DOMAIN=localhost docker compose --env-file "$TEST_ENV_FILE" -f "$COMPOSE_FILE" up -d --build --wait 2>&1 | sed 's/^/    /'

  # ── Wait ──
  echo -n "==> 等待就绪"
  for i in $(seq 1 30); do
    if curl -sf -o /dev/null http://localhost:8080/api/health 2>/dev/null; then
      echo " ✓"
      return 0
    fi
    echo -n .
    sleep 1
  done
  echo " ✗ 超时"
  docker compose --env-file "$TEST_ENV_FILE" -f "$COMPOSE_FILE" logs --tail 20 2>&1 | sed 's/^/    /' || true
  exit 1
}

do_down() {
  echo "==> 停止并清理..."
  cd "$PROJECT_DIR"
  # Compose requires the env file to exist; recreate an empty one if `up` already cleaned it up.
  [ -f "$TEST_ENV_FILE" ] || : > "$TEST_ENV_FILE"
  # Image references are irrelevant while tearing down, but Compose still
  # interpolates them. Use a non-pullable placeholder rather than a usable
  # hidden engine default.
  export LIKI_ENGINE_IMAGE="${LIKI_ENGINE_IMAGE:-liki-engine/e2e-down-placeholder}"
  docker compose --env-file "$TEST_ENV_FILE" -f "$COMPOSE_FILE" down --volumes --remove-orphans --timeout 5 2>/dev/null || true
  rm -f "$TEST_ENV_FILE"
}

case "$CMD" in
  up)   do_up ;;
  down) do_down ;;
  *)    echo "Usage: $0 up|down"; exit 1 ;;
esac
