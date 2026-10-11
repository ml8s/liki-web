#!/usr/bin/env bash
# Start Liki development environment WITHOUT Docker.
# Engine MCP on :8085, Web API on :8081, Caddy on :8080.
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
LIKI_ENGINE_MCP_TOKEN="${LIKI_ENGINE_MCP_TOKEN:-}"
export LIKI_WEB_HOST="${LIKI_WEB_HOST:-localhost}"
export LIKI_ENGINE_HOST="${LIKI_ENGINE_HOST:-localhost}"

export DODO_API_KEY="${DODO_API_KEY:-}"
export DODO_WEBHOOK_KEY="${DODO_WEBHOOK_KEY:-}"
export DODO_TEST_MODE="${DODO_TEST_MODE:-true}"
export XUNHU_APPID="${XUNHU_APPID:-}"
export XUNHU_APPSECRET="${XUNHU_APPSECRET:-}"
export RESEND_API_KEY="${RESEND_API_KEY:-}"
export RESEND_FROM="${RESEND_FROM:-noreply@localhost}"
export RETURN_URL="${RETURN_URL:-http://localhost:8080}"
export LIKI_DATA_DIR="${LIKI_DATA_DIR:-$(pwd)/data}"
mkdir -p "$LIKI_DATA_DIR"
export LISTEN_ADDR="${LISTEN_ADDR:-:8081}"
export LIKI_ENGINE_MCP_URL
export LIKI_ENGINE_MCP_TOKEN
export LIKI_AGENTS_URL="${LIKI_AGENTS_URL:-http://127.0.0.1:18083}"
export LIKI_AGENTS_INTERNAL_TOKEN="${LIKI_AGENTS_INTERNAL_TOKEN:-dev-agent-secret}"
export TRUSTED_PROXY_HOPS="${TRUSTED_PROXY_HOPS:-1}"
export JWT_SECRET="${JWT_SECRET:-dev-secret-do-not-use-in-production}"
export REFERRAL_PAYOUT_SECRET="${REFERRAL_PAYOUT_SECRET:-dev-referral-payout-secret-do-not-use-in-production}"
export DEV_MODE="${DEV_MODE:-true}"

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
ENGINE_RPC_BIN="${ENGINE_RPC_BIN:-}"

# Web binary (pre-built by 'make build-go')
WEB_BIN="${WEB_BIN:-bin/liki-web}"
if [ ! -x "$WEB_BIN" ]; then
  echo "ERROR: $WEB_BIN not found. Run: make build-go" >&2
  exit 1
fi

# Free ports
for port in 8080 8081 8082 8085; do
  if fuser "$port/tcp" 2>/dev/null; then
    echo "==> Warning: killing existing process on port $port"
    fuser -k "$port/tcp" 2>/dev/null || true
  fi
done
for i in $(seq 1 30); do
  if ! fuser 8080/tcp 8081/tcp 8082/tcp 8085/tcp 2>/dev/null; then break; fi
  sleep 0.5
done

ENGINE_MCP_PID=""
echo "==> Starting engine MCP on :8085"
"$ENGINE_MCP_BIN" --addr :8085 &
ENGINE_MCP_PID=$!

ENGINE_RPC_PID=""
if [ -n "$ENGINE_RPC_BIN" ]; then
  echo "==> Starting legacy engine JSON-RPC on :8082"
  LISTEN_ADDR=:8082 "$ENGINE_RPC_BIN" &
  ENGINE_RPC_PID=$!
fi
for i in $(seq 1 30); do
  curl -sf "$LIKI_ENGINE_MCP_URL/health" >/dev/null 2>&1 && break
  sleep 0.5
done

echo "==> Starting web API on $LISTEN_ADDR"
"$WEB_BIN" &
API_PID=$!
for i in $(seq 1 30); do
  curl -sf "http://${LIKI_WEB_HOST}:${LISTEN_ADDR#*:}/api/health" >/dev/null 2>&1 && break
  sleep 0.5
done

echo "==> Starting Caddy on :8080"
"$CADDY_BIN" run --config "$PROJECT_DIR/dev/Caddyfile.local" --adapter caddyfile &
CADDY_PID=$!

trap "kill $ENGINE_MCP_PID $ENGINE_RPC_PID $API_PID $CADDY_PID 2>/dev/null; exit" INT TERM EXIT

echo "==> Ready: http://localhost:8080"
echo "    Engine MCP: http://localhost:8085/mcp"
echo "    Legacy Engine JSON-RPC: http://localhost:8080/jsonrpc (requires ENGINE_RPC_BIN)"
echo "    Press Ctrl+C to stop"
wait
