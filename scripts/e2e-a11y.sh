#!/usr/bin/env bash
# Run only the accessibility contract during make gate. This starts the same
# local web/Caddy pair as CI's e2e job; product APIs are mocked by the specs.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

DATA_DIR="$(mktemp -d /tmp/liki-web-a11y.XXXXXX)"
export LIKI_DATA_DIR="$DATA_DIR"
WEB_LOG="$DATA_DIR/web.log"
CADDY_LOG="$DATA_DIR/caddy.log"
WEB_PID=""
CADDY_PID=""
cleanup() {
  [ -z "$CADDY_PID" ] || kill "$CADDY_PID" 2>/dev/null || true
  [ -z "$WEB_PID" ] || kill "$WEB_PID" 2>/dev/null || true
  wait 2>/dev/null || true
  rm -rf "$DATA_DIR"
}
trap cleanup EXIT

export JWT_SECRET="liki-a11y-secret-0123456789abcdef"
export LIKI_REFERRAL_PAYOUT_ENCRYPTION_KEY="liki-a11y-referral-payout-encryption-key-32-bytes"
export LIKI_HTTP_LISTEN_ADDRESS=":8081"
export LIKI_PUBLIC_BASE_URL="http://localhost:8080"
export LIKI_CHAT_ONE_TIME_PRICE_CNY_MINOR_UNITS="7900"
export LIKI_CHAT_ONE_TIME_PRICE_USD_MINOR_UNITS="2990"
export LIKI_CHAT_SUBSCRIPTION_PRICE_USD_MINOR_UNITS="1990"
export LIKI_CHAT_ONE_TIME_ACCESS_DAYS="30"
export LIKI_CHAT_KOL_REVENUE_SHARE_BASIS_POINTS="4000"
export LIKI_CHAT_SCOUT_REVENUE_SHARE_BASIS_POINTS="1000"
export DODO_PRODUCT_LIKI_CHAT_ONE_TIME="a11y-one-time-product"
export DODO_PRODUCT_LIKI_CHAT_SUBSCRIPTION="a11y-subscription-product"
export XUNHU_APPID="a11y-test-dummy"
export XUNHU_APPSECRET="a11y-test-dummy"
export LIKI_PAYMENT_WEBHOOK_URL="http://localhost:8081/api/payments/webhook"
export LIKI_AGENTS_URL="http://127.0.0.1:18083"
export LIKI_AGENTS_INTERNAL_TOKEN="liki-a11y-agent-secret"
export LIKI_DEVELOPMENT_MODE="true"
export LIKI_WEB_HOST="localhost"

./bin/liki-web >"$WEB_LOG" 2>&1 &
WEB_PID=$!
for _ in $(seq 1 10); do
  curl -sf http://localhost:8081/api/health 2>/dev/null | grep -q '"ok"' && break
  sleep 1
done
curl -sf http://localhost:8081/api/health | grep -q '"ok"' || {
  echo "liki-web did not become healthy" >&2
  exit 1
}

caddy run --config dev/Caddyfile.local >"$CADDY_LOG" 2>&1 &
CADDY_PID=$!
for _ in $(seq 1 30); do
  if curl -sf http://localhost:8080/zh-Hans/ 2>/dev/null | grep -q '<html' && \
    curl -sf http://localhost:8080/api/health 2>/dev/null | grep -q '"ok"'; then
    break
  fi
  sleep 1
done
curl -sf http://localhost:8080/api/health | grep -q '"ok"' || {
  echo "Caddy did not become healthy" >&2
  exit 1
}

# The Consultation/Mine contract is flag-gated at build time. Build the SPA
# once with the preview experience enabled so the accessibility gate covers the
# surface that will ship, while production builds remain flag-off by default.
echo "[e2e-a11y] Liki App (Consultation preview)..."
(
  cd web/app
  VITE_FEATURE_APP_FLOW_REDESIGN=true npm run build
)

cd web
set +e
E2E_SPECS="${E2E_A11Y_SPECS:-e2e/journeys/app-a11y.spec.ts e2e/journeys/admin-a11y.spec.ts e2e/journeys/domain-flow-a11y.spec.ts}"
BASE_URL="http://localhost:8080" npx playwright test \
  --config e2e/playwright.config.ts \
  $E2E_SPECS
test_rc=$?
set -e
if [ "$test_rc" -ne 0 ]; then
  echo "=== liki-web log ===" >&2
  tail -100 "$WEB_LOG" >&2 || true
  echo "=== caddy log ===" >&2
  tail -100 "$CADDY_LOG" >&2 || true
fi
exit "$test_rc"
