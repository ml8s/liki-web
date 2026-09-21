#!/bin/bash
# Docker 测试环境启停
#   docker-test.sh up    构建镜像 + 启动容器 + 等待就绪
#   docker-test.sh down  停止容器 + 清理
set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ENGINE_DIR="${ENGINE_DIR:-$PROJECT_DIR/../liki/engine}"
COMPOSE_FILE="$PROJECT_DIR/deploy/liki/docker-compose.test.yml"
DEPLOY_DIR="$PROJECT_DIR/deploy/liki"

CMD="${1:-up}"

do_up() {
  trap 'rm -f "$DEPLOY_DIR/.env"' EXIT

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
  "$SCRIPT_DIR/build-docker.sh" --compose-file "$COMPOSE_FILE" 2>&1 | sed 's/^/    /'

  # ── Start ──
  echo "==> 启动容器..."

  # Populate web volume
  TMP_WEB=$(mktemp -d)
  cp -rL "$PROJECT_DIR/web/." "$TMP_WEB/"
  BUILD_TS=$(date '+%s')
  find "$TMP_WEB" -name '*.html' \
    -exec sed -Ei "s|(src=\"[/]?js/[^\"?]*)(\?v=[^\"]*)?|\1?v=$BUILD_TS|g; s|(href=\"[/]?css/[^\"?]*)(\?v=[^\"]*)?|\1?v=$BUILD_TS|g" {} + \
    -exec sed -i "s/BUILD_TIME_PLACEHOLDER/$BUILD_TIME/g" {} +
  (cd "$TMP_WEB" && tar czf /tmp/liki-web-test.tar.gz .)
  rm -rf "$TMP_WEB"

  docker volume create liki-web_data 2>/dev/null || true
  docker run --rm -i -v liki-web_data:/dest alpine sh -c \
    "find /dest -mindepth 1 -delete 2>/dev/null; tar xzf - -C /dest" < /tmp/liki-web-test.tar.gz
  rm -f /tmp/liki-web-test.tar.gz

  # Ensure DB volume writable
  docker volume create liki_data 2>/dev/null || true
  docker run --rm -v liki_data:/data alpine chown 1000:1000 /data 2>/dev/null || true

  # Write .env
  /bin/grep -E '^(JWT_SECRET|REFERRAL_PAYOUT_SECRET|DEEPSEEK|LLM_|DODO_|XUNHU_|RESEND_|TENCENT_|EMAIL_|RETURN_URL|PAYMENTS_WEBHOOK_URL|ADMIN_)' "$PROJECT_DIR/.env" > "$DEPLOY_DIR/.env"
  if ! grep -q '^REFERRAL_PAYOUT_SECRET=' "$DEPLOY_DIR/.env"; then
    printf '\nREFERRAL_PAYOUT_SECRET=e2e-referral-payout-secret-0123456789abcdef\n' >> "$DEPLOY_DIR/.env"
  fi
  cat >> "$DEPLOY_DIR/.env" << 'EOF'
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
  docker compose -f "$COMPOSE_FILE" down --volumes --timeout 5 2>/dev/null || true
  rm -f "$DEPLOY_DIR/.env"
}

case "$CMD" in
  up)   do_up ;;
  down) do_down ;;
  *)    echo "Usage: $0 up|down"; exit 1 ;;
esac
