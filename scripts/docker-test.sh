#!/bin/bash
# Docker 测试环境启停
#   docker-test.sh up    构建镜像 + 启动容器 + 等待就绪
#   docker-test.sh down  停止容器 + 清理
set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ENGINE_DIR="${ENGINE_DIR:-$PROJECT_DIR/../liki/engine}"
DEV_DIR="$PROJECT_DIR/dev"
COMPOSE_FILE="$DEV_DIR/docker-compose.yml"
TEST_ENV_FILE="${LIKI_TEST_ENV_FILE:-/tmp/liki-web-e2e.env}"
export LIKI_TEST_ENV_FILE="$TEST_ENV_FILE"

CMD="${1:-up}"

do_up() {
  trap 'rm -f "$TEST_ENV_FILE"' EXIT

  # ── Build ──
  echo "==> 构建..."
  BUILD_TIME=$(date '+%Y-%m-%dT%H:%M:%S')
  export BUILD_TIME

  # Run independent builds in parallel
  (
    echo "  → skills"
    for repo in liki; do  # 现阶段只发布 liki
      (cd "$PROJECT_DIR/../liki" && make build-archive >/dev/null 2>&1)
    done
    "$SCRIPT_DIR/sync-skills.sh" >/dev/null
  ) &
  PID_SKILLS=$!

  (
    echo "  → engine"
    make -C "$ENGINE_DIR" build-linux 2>&1 | sed 's/^/    /'
  ) &
  PID_ENGINE=$!

  (
    echo "  → web (linux amd64)"
    BUILD_TIME="$BUILD_TIME" "$SCRIPT_DIR/build-web.sh"
    BUILD_TIME="$BUILD_TIME" "$SCRIPT_DIR/build-go.sh" --goos linux --goarch amd64 --output "$PROJECT_DIR/bin/liki-web"
  ) &
  PID_WEB=$!

  wait $PID_SKILLS $PID_ENGINE $PID_WEB

  echo "  → Docker images"
  docker compose -f "$COMPOSE_FILE" build engine web 2>&1 | sed 's/^/    /'

  # ── Start ──
  echo "==> 启动容器..."

  # E2E uses a self-contained environment. It must not inherit production
  # provider credentials or depend on optional services on the developer host.
  cat > "$TEST_ENV_FILE" << 'EOF'
JWT_SECRET=e2e-jwt-secret-0123456789abcdefghijklmnop
LIKI_LLM_API_KEY=e2e-dummy
LIKI_LLM_MODEL=e2e-dummy
XUNHU_APPID=e2e
XUNHU_APPSECRET=e2e
REFERRAL_PAYOUT_SECRET=e2e-referral-payout-secret-0123456789abcdef
DOMAIN=localhost
LISTEN_ADDR=:8081
EOF

  DOMAIN=localhost docker compose -f "$COMPOSE_FILE" up -d --wait 2>&1 | sed 's/^/    /'

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
  docker compose -f "$COMPOSE_FILE" logs --tail 20 2>&1 | sed 's/^/    /' || true
  exit 1
}

do_down() {
  echo "==> 停止并清理..."
  cd "$PROJECT_DIR"
  docker compose -f "$COMPOSE_FILE" down --volumes --remove-orphans --timeout 5 2>/dev/null || true
  rm -f "$TEST_ENV_FILE"
}

case "$CMD" in
  up)   do_up ;;
  down) do_down ;;
  *)    echo "Usage: $0 up|down"; exit 1 ;;
esac
