#!/usr/bin/env bash
# Start Liki development environment WITHOUT Docker.
# Engine MCP on :8085, Counsel MCP on :8086 (optional), Web API on :8081, Caddy on :8080.
#
# Prerequisites: make build-go build-web (run by Makefile 'dev' target)
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

# Load .env if present
if [ -f .env ]; then
  set -a; source .env; set +a
fi

LIKI_ENGINE_MCP_URL="${LIKI_ENGINE_MCP_URL:-http://localhost:8085}"
export LIKI_WEB_HOST="${LIKI_WEB_HOST:-localhost}"

export DODO_API_KEY="${DODO_API_KEY:-}"
export DODO_WEBHOOK_KEY="${DODO_WEBHOOK_KEY:-}"
export DODO_TEST_MODE="${DODO_TEST_MODE:-true}"
export XUNHU_APPID="${XUNHU_APPID:-}"
export XUNHU_APPSECRET="${XUNHU_APPSECRET:-}"
export RESEND_API_KEY="${RESEND_API_KEY:-}"
export LIKI_EMAIL_FROM_ADDRESS="${LIKI_EMAIL_FROM_ADDRESS:-Liki <noreply@localhost>}"
export LIKI_PUBLIC_BASE_URL="${LIKI_PUBLIC_BASE_URL:-http://localhost:8080}"
export LIKI_DATA_DIR="${LIKI_DATA_DIR:-$(pwd)/data}"
mkdir -p "$LIKI_DATA_DIR"
export LIKI_HTTP_LISTEN_ADDRESS="${LIKI_HTTP_LISTEN_ADDRESS:-:8081}"
export LIKI_ENGINE_MCP_URL
export LIKI_EXPERTS_URL="${LIKI_EXPERTS_URL:-http://127.0.0.1:18083}"
export LIKI_EXPERTS_TOKEN="${LIKI_EXPERTS_TOKEN:-dev-agent-secret}"
export TRUSTED_PROXY_HOPS="${TRUSTED_PROXY_HOPS:-1}"
export JWT_SECRET="${JWT_SECRET:-dev-secret-do-not-use-in-production}"
export LIKI_REFERRAL_PAYOUT_ENCRYPTION_KEY="${LIKI_REFERRAL_PAYOUT_ENCRYPTION_KEY:-dev-referral-payout-encryption-key-32-bytes}"
export LIKI_DEVELOPMENT_MODE="${LIKI_DEVELOPMENT_MODE:-true}"

# Caddy binary
if [ -z "${CADDY_BIN:-}" ]; then
  if command -v caddy &>/dev/null; then
    CADDY_BIN="caddy"
  elif [ -f /tmp/caddy ]; then
    chmod +x /tmp/caddy
    CADDY_BIN="/tmp/caddy"
  else
    echo "==> Downloading Caddy..."
    arch=$(uname -m)
    case "$arch" in
      x86_64)  caddy_arch="amd64" ;;
      aarch64) caddy_arch="arm64" ;;
      *)       echo "ERROR: unsupported arch $arch"; exit 1 ;;
    esac
    curl -sL "https://caddyserver.com/api/download?os=linux&arch=${caddy_arch}" -o /tmp/caddy
    curl -sL "https://caddyserver.com/api/download?os=linux&arch=${caddy_arch}&format=checksum" -o /tmp/caddy.sha256
    expected=$(awk '{print $1}' /tmp/caddy.sha256)
    actual=$(sha256sum /tmp/caddy | awk '{print $1}')
    if [ "$expected" != "$actual" ]; then
      echo "ERROR: Caddy checksum verification failed"
      exit 1
    fi
    rm -f /tmp/caddy.sha256
    chmod +x /tmp/caddy
    CADDY_BIN="/tmp/caddy"
  fi
fi

ENGINE_MCP_BIN="${ENGINE_MCP_BIN:-}"
if [ -z "$ENGINE_MCP_BIN" ]; then
  echo "ERROR: set ENGINE_MCP_BIN to an extracted engine-mcp artifact" >&2
  exit 1
fi

# Counsel MCP (optional; needed by the naming free tool). Point COUNSEL_DIR at the
# liki/counsel checkout to enable; without it the naming island will fail locally.
COUNSEL_DIR="${COUNSEL_DIR:-}"
COUNSEL_PY="${COUNSEL_PY:-$COUNSEL_DIR/.venv/bin/python}"
if [ -n "$COUNSEL_DIR" ] && [ ! -x "$COUNSEL_PY" ]; then
  echo "==> Warning: COUNSEL_DIR set but $COUNSEL_PY not executable; skipping counsel"
  COUNSEL_DIR=""
fi
if [ -z "$COUNSEL_DIR" ]; then
  echo "==> Counsel MCP disabled (set COUNSEL_DIR to enable; naming tool needs it)"
fi

# Web binary (pre-built by 'make build-go')
WEB_BIN="${WEB_BIN:-bin/liki-web}"
if [ ! -x "$WEB_BIN" ]; then
  echo "ERROR: $WEB_BIN not found. Run: make build-go" >&2
  exit 1
fi

# Free ports
for port in 8080 8081 8085 8086; do
  if fuser "$port/tcp" 2>/dev/null; then
    echo "==> Warning: killing existing process on port $port"
    fuser -k "$port/tcp" 2>/dev/null || true
  fi
done
for i in $(seq 1 30); do
  if ! fuser 8080/tcp 8081/tcp 8085/tcp 8086/tcp 2>/dev/null; then break; fi
  sleep 0.5
done

ENGINE_MCP_PID=""
echo "==> Starting engine MCP on :8085"
"$ENGINE_MCP_BIN" --addr :8085 &
ENGINE_MCP_PID=$!

for i in $(seq 1 30); do
  curl -sf "$LIKI_ENGINE_MCP_URL/health" >/dev/null 2>&1 && break
  sleep 0.5
done

COUNSEL_PID=""
if [ -n "$COUNSEL_DIR" ]; then
  echo "==> Starting counsel MCP on :8086"
  (cd "$COUNSEL_DIR" && LIKI_ENGINE_MCP_URL=http://localhost:8085/mcp "$COUNSEL_PY" -m uvicorn app.counsel_mcp:app --host 127.0.0.1 --port 8086) &
  COUNSEL_PID=$!
  for i in $(seq 1 40); do
    curl -sf http://localhost:8086/readyz >/dev/null 2>&1 && break
    sleep 0.5
  done
fi

echo "==> Starting web API on $LIKI_HTTP_LISTEN_ADDRESS"
"$WEB_BIN" &
API_PID=$!
for i in $(seq 1 30); do
  curl -sf "http://${LIKI_WEB_HOST}:${LIKI_HTTP_LISTEN_ADDRESS#*:}/api/health" >/dev/null 2>&1 && break
  sleep 0.5
done

echo "==> Starting Caddy on :8080"
"$CADDY_BIN" run --config "$PROJECT_DIR/dev/Caddyfile.local" --adapter caddyfile &
CADDY_PID=$!

trap "kill $ENGINE_MCP_PID $COUNSEL_PID $API_PID $CADDY_PID 2>/dev/null; exit" INT TERM EXIT

echo "==> Ready: http://localhost:8080"
echo "    Engine MCP: http://localhost:8085/mcp"
if [ -n "$COUNSEL_DIR" ]; then
  echo "    Counsel MCP: http://localhost:8086/mcp"
fi
echo "    Press Ctrl+C to stop"
wait
