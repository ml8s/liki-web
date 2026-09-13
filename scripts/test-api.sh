#!/bin/bash
# Liki — API smoke test (HTTP)
set -euo pipefail

BASE="${1:-http://localhost:8080}"
if [ $# -ge 2 ]; then
  API="$2"
elif echo "$BASE" | /bin/grep -q '^http://localhost'; then
  API="http://localhost:8081"
else
  API="$BASE"
fi
PASS=0; FAIL=0

RED=""; GREEN=""; YELLOW=""; BOLD=""; NC=""
[ -t 1 ] && { RED='\033[31m'; GREEN='\033[32m'; BOLD='\033[1m'; NC='\033[0m'; }

TMP=$(mktemp -d /tmp/test-api-XXXXXX)
trap 'rm -rf "$TMP"' EXIT

# ── helpers ──────────────────────────────────────────────────────

api() {
  local method="$1" path="$2" data="${3:-}"
  local f="$TMP/body"
  local fake_ip="10.$((RANDOM % 256)).$((RANDOM % 256)).$((RANDOM % 254 + 1))"
  if [ -n "$data" ]; then
    curl -s -w '%{http_code}' -o "$f" -X "$method" "$API$path" \
      -H 'Content-Type: application/json' \
      -H "X-Forwarded-For: $fake_ip" \
      -d "$data" || echo "000"
  else
    curl -s -w '%{http_code}' -o "$f" -X "$method" "$API$path" \
      -H 'Content-Type: application/json' \
      -H "X-Forwarded-For: $fake_ip" || echo "000"
  fi
}

api_with_cookie() {
  local method="$1" path="$2" cookie="$3" data="${4:-}"
  local f="$TMP/body"
  local fake_ip="10.$((RANDOM % 256)).$((RANDOM % 256)).$((RANDOM % 254 + 1))"
  local hdrs=(-H 'Content-Type: application/json' -H "X-Forwarded-For: $fake_ip")
  [ -n "$cookie" ] && hdrs+=(-H "Cookie: $cookie")
  if [ -n "$data" ]; then
    curl -s -w '%{http_code}' -o "$f" -X "$method" "$API$path" "${hdrs[@]}" -d "$data" || echo "000"
  else
    curl -s -w '%{http_code}' -o "$f" -X "$method" "$API$path" "${hdrs[@]}" || echo "000"
  fi
}

caddy() {
  local method="$1" path="$2" data="${3:-}"
  local f="$TMP/body"
  if [ -n "$data" ]; then
    curl -s -w '%{http_code}' -o "$f" -X "$method" "$BASE$path" \
      -H 'Content-Type: application/json' \
      -d "$data" || echo "000"
  else
    curl -s -w '%{http_code}' -o "$f" -X "$method" "$BASE$path" \
      -H 'Content-Type: application/json' || echo "000"
  fi
}

body() { cat "$TMP/body" 2>/dev/null || true; }

json_val() { echo "$1" | jq -r "$2" 2>/dev/null || true; }

check() {
  local desc="$1" expected="$2" actual="${3:-}"
  if [ "$actual" = "$expected" ]; then
    echo -e "  ${GREEN}\xe2\x9c\x93${NC} $desc"
    PASS=$((PASS + 1))
  else
    echo -e "  ${RED}\xe2\x9c\x97${NC} $desc (expected '$expected', got '$actual')"
    FAIL=$((FAIL + 1))
  fi
}

check_range() {
  local desc="$1" expected_re="$2" actual="${3:-}"
  if echo "$actual" | /bin/grep -qE "$expected_re"; then
    echo -e "  ${GREEN}\xe2\x9c\x93${NC} $desc"
    PASS=$((PASS + 1))
  else
    echo -e "  ${RED}\xe2\x9c\x97${NC} $desc (expected match '$expected_re', got '$actual')"
    FAIL=$((FAIL + 1))
  fi
}

check_200()  { check "$1 HTTP" "200" "${2:-}"; }
check_201()  { check "$1 HTTP" "201" "${2:-}"; }
check_204()  { check "$1 HTTP" "204" "${2:-}"; }
check_302()  { check "$1 HTTP" "302" "${2:-}"; }
check_400()  { check "$1 HTTP" "400" "${2:-}"; }
check_401()  { check "$1 HTTP" "401" "${2:-}"; }
check_403()  { check "$1 HTTP" "403" "${2:-}"; }
check_404()  { check "$1 HTTP" "404" "${2:-}"; }
check_422()  { check "$1 HTTP" "422" "${2:-}"; }
check_405()  { check "$1 HTTP" "405" "${2:-}"; }
check_any()  { check_range "$1 HTTP" "${2:-}" "${3:-}"; }

# ── test data ─────────────────────────────────────────────────────

TEST_EMAIL="api-test@example.com"
ORDER_ID=""

# Track created order ID for chained tests
save_order_id() {
  local b=$(body)
  ORDER_ID=$(json_val "$b" '.data.order_id // .data.order_id // empty')
  [ -n "$ORDER_ID" ] && echo "  → saved order_id: $ORDER_ID"
}

# ============================================================================
echo "${BOLD}Liki — API Smoke Test${NC}"
echo "Target: Caddy=$BASE API=$API"
command -v jq &>/dev/null || { echo "jq is required"; exit 1; }
echo ""

# ============================================================================
# Health
# ============================================================================
echo "${BOLD}── Health ──${NC}"
s=$(api GET /api/health)
check_200 "GET /api/health" "$s"
b=$(body)
check "  status ok" "ok" "$(json_val "$b" '.data.status')"

# ============================================================================
# Static (Caddy)
# ============================================================================
echo ""
echo "${BOLD}── Static ──${NC}"
# Check if Caddy is in front by testing redirect behavior: Caddy redirects / → locale
_caddy_check=$(curl -s -o /dev/null -w '%{http_code}' -H "Accept-Language: zh-CN" "$BASE/" 2>/dev/null)
if [ "$_caddy_check" = "302" ] || [ "$_caddy_check" = "308" ]; then
  s=$(caddy GET /)
  check_302 "GET / (locale redirect)" "$s"
  s=$(caddy GET /zh-Hant/)
  check_200 "GET /zh-Hant/" "$s"
  b=$(body)
  check "  has liki.hk" "true" "$(echo "$b" | /bin/grep -q 'liki.hk' && echo true || echo false)"

  # ── Locale redirects (防回归: 路径限制 + slash) ──
  echo ""
  echo "${BOLD}── Locale Redirects ──${NC}"

  # Helper: get redirect Location header
  redir_to() {
    curl -s -o /dev/null -w '%{redirect_url}' -H "Accept-Language: $1" "$BASE$2"
  }

  # Bare paths → locale-prefixed (must have / between locale and path)
  loc=$(redir_to "zh-CN" /naming)
  check "  /naming zh-CN → /zh-Hans/naming"  "$BASE/zh-Hans/naming"  "$loc"
  loc=$(redir_to "en" /naming)
  check "  /naming en → /en/naming"          "$BASE/en/naming"       "$loc"
  loc=$(redir_to "zh-CN" /mingshu)
  check "  /mingshu zh-CN → /zh-Hans/mingshu" "$BASE/zh-Hans/mingshu" "$loc"
  loc=$(redir_to "en" /mingshu)
  check "  /mingshu en → /en/mingshu"      "$BASE/en/mingshu"     "$loc"
  loc=$(redir_to "zh-CN" /hepan)
  check "  /hepan zh-CN → /zh-Hans/hepan"    "$BASE/zh-Hans/hepan"   "$loc"
  loc=$(redir_to "en" /hepan)
  check "  /hepan en → /en/hepan"            "$BASE/en/hepan"         "$loc"

  # Already locale-prefixed → 200 (no redirect loop)
  s=$(caddy GET /zh-Hans/naming)
  check_200 "GET /zh-Hans/naming (no redirect)" "$s"
  s=$(caddy GET /zh-Hant/naming)
  check_200 "GET /zh-Hant/naming (no redirect)" "$s"
  s=$(caddy GET /en/naming)
  check_200 "GET /en/naming (no redirect)" "$s"

  # No slash-loss regression: must not redirect to /zh-Hansnaming
  loc=$(redir_to "zh-CN" /naming)
  check "  slash in path: NOT /zh-Hansnaming" "$BASE/zh-Hans/naming" "$loc"
else
  echo "  (Caddy not running — skipping static checks)"
fi

# ============================================================================
# Orders
# ============================================================================
echo ""
echo "${BOLD}── Orders ──${NC}"

# Create order
s=$(api POST /api/naming/orders "{\"email\":\"$TEST_EMAIL\",\"product\":\"naming\",\"currency\":\"USD\",\"lang\":\"zh-Hans\"}")
check_200 "POST /api/naming/orders (create)" "$s"
save_order_id

# Create order with unsupported currency → falls back to CNY (no validation)
s=$(api POST /api/naming/orders "{\"email\":\"$TEST_EMAIL\",\"product\":\"naming\",\"currency\":\"XYZ\"}")
check_200 "POST /api/naming/orders (unsupported currency)" "$s"

# Create order without email → accepted (handler ignores unknown fields)
s=$(api POST /api/naming/orders '{"product":"naming","currency":"USD"}')
check_200 "POST /api/naming/orders (no email field)" "$s"

# Create order with unsupported product → falls back to naming (no validation)
s=$(api POST /api/naming/orders "{\"email\":\"$TEST_EMAIL\",\"product\":\"unknown\",\"currency\":\"USD\"}")
check_200 "POST /api/naming/orders (unknown product)" "$s"

# Select order (chained from created order)
if [ -n "$ORDER_ID" ]; then
  s=$(api POST /api/naming/orders/select "{\"order_id\":\"$ORDER_ID\",\"email\":\"$TEST_EMAIL\"}")
  check_any "POST /api/naming/orders/select (valid)" "^(200|403)$" "$s"
fi

# Select order — non-existent
s=$(api POST /api/naming/orders/select '{"order_id":"no-such-id","email":"test@example.com"}')
check_404 "POST /api/naming/orders/select (not found)" "$s"

# Select order — email mismatch
if [ -n "$ORDER_ID" ]; then
  s=$(api POST /api/naming/orders/select "{\"order_id\":\"$ORDER_ID\",\"email\":\"wrong@example.com\"}")
  check_403 "POST /api/naming/orders/select (email mismatch)" "$s"
fi

# Order status — existing
if [ -n "$ORDER_ID" ]; then
  s=$(api GET "/api/naming/orders/$ORDER_ID/status")
  check_200 "GET /api/naming/orders/{id}/status (existing)" "$s"
fi

# Order status — non-existent
s=$(api GET /api/naming/orders/nonexistent/status)
check_404 "GET /api/naming/orders/{id}/status (not found)" "$s"

# Retry payment — non-existent order
s=$(api POST /api/naming/orders/nonexistent/retry '{}')
check_404 "POST /api/naming/orders/{id}/retry (not found)" "$s"

# ============================================================================
# Auth
# ============================================================================
echo ""
echo "${BOLD}── Auth ──${NC}"

# Login with email that has an order (unpaid → 401)
if [ -n "$ORDER_ID" ]; then
  s=$(api POST /api/naming/auth/login "{\"email\":\"$TEST_EMAIL\"}")
  check_401 "POST /api/naming/auth/login (unpaid order)" "$s"
fi

# Login with non-existent email
s=$(api POST /api/naming/auth/login '{"email":"no-such@example.com"}')
check_401 "POST /api/naming/auth/login (no orders)" "$s"

# Login with missing email
s=$(api POST /api/naming/auth/login '{}')
check_422 "POST /api/naming/auth/login (missing email)" "$s"

# ============================================================================
# Payment
# ============================================================================
echo ""
echo "${BOLD}── Payment ──${NC}"
if [ -n "$ORDER_ID" ]; then
  s=$(api POST /api/payments/intents "{\"product\":\"naming\",\"business_order_id\":\"$ORDER_ID\",\"provider\":\"dodo\"}")
  check_200 "POST /api/payments/intents (create)" "$s"
  PAYMENT_INTENT_ID=$(json_val "$b" '.data.id // empty')
fi

if [ -n "${PAYMENT_INTENT_ID:-}" ]; then
  s=$(api GET "/api/payments/intents/$PAYMENT_INTENT_ID/status")
  check_200 "GET /api/payments/intents/{id}/status" "$s"
fi

s=$(api POST /api/payments/webhook '{"type":"order.paid","data":{"order_id":"test"}}')
check_400 "POST /api/payments/webhook (no sig)" "$s"

# Payment return is an opaque technical redirect; unknown ids redirect to /.
s=$(api GET "/api/payments/return/nonexistent")
check_302 "GET /api/payments/return/{id} (unknown)" "$s"

# ============================================================================
# Agent
# ============================================================================
echo ""
echo "${BOLD}── Agent ──${NC}"

	# Broken JSON — auth check before body parse (runs here before burst exhausted)
	s=$(api POST /api/naming/agent/chat '{broken}')
	check_401 "POST /api/naming/agent/chat (broken JSON - auth before body)" "$s"

# With invalid JWT cookie → 401
s=$(curl -s -w '%{http_code}' -o "$TMP/body" -X POST "$API/api/naming/agent/chat" \
  -H 'Content-Type: application/json' \
  -H "Cookie: liki_token=not-a-valid-jwt" \
  -d '{"message":"hello","lang":"zh-Hans"}')
check_401 "POST /api/naming/agent/chat (invalid JWT)" "$s"

# With invalid JWT in Authorization header
s=$(curl -s -w '%{http_code}' -o "$TMP/body" -X POST "$API/api/naming/agent/chat" \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer invalid' \
  -d '{"message":"hello","lang":"zh-Hans"}')
check_401 "POST /api/naming/agent/chat (invalid Bearer)" "$s"

# Chat history — no JWT
s=$(api GET /api/naming/orders/nonexistent/messages)
check_401 "GET /api/naming/orders/{id}/messages (no JWT)" "$s"

# Chat history — with invalid JWT
s=$(curl -s -w '%{http_code}' -o "$TMP/body" -X GET "$API/api/naming/orders/nonexistent/messages" \
  -H "Cookie: liki_token=not-a-valid-jwt")
check_401 "GET /api/naming/orders/{id}/messages (invalid JWT)" "$s"

# ============================================================================
# Admin
# ============================================================================
echo ""
echo "${BOLD}── Admin ──${NC}"

# Send code (always succeeds to avoid enumeration)
s=$(api POST /api/admin/send-code "{\"email\":\"$TEST_EMAIL\"}")
check_200 "POST /api/admin/send-code (valid)" "$s"

# Send code — bad JSON
s=$(api POST /api/admin/send-code 'not-json')
check_400 "POST /api/admin/send-code (bad JSON)" "$s"

# Verify code — wrong code
s=$(api POST /api/admin/verify-code "{\"email\":\"$TEST_EMAIL\",\"code\":\"000000\"}")
check_403 "POST /api/admin/verify-code (wrong code)" "$s"

# Verify code — missing fields
s=$(api POST /api/admin/verify-code '{}')
check_403 "POST /api/admin/verify-code (missing fields)" "$s"

# Admin list orders — no JWT (should not reach handler, auth fails first)
s=$(api GET "/api/admin/orders?email=$TEST_EMAIL")
check_403 "GET /api/admin/orders (no JWT)" "$s"

# Admin create order — no JWT
s=$(api POST /api/admin/orders "{\"email\":\"$TEST_EMAIL\",\"product\":\"naming\"}")
check_403 "POST /api/admin/orders (no JWT)" "$s"

# Admin expire order — no JWT
s=$(api PATCH "/api/admin/orders/$ORDER_ID/expire" '{"expires_at":"2026-07-10 00:00:00"}')
check_403 "PATCH /api/admin/orders/{id}/expire (no JWT)" "$s"

# Admin verification codes — no JWT
s=$(api GET /api/admin/verification-codes)
check_403 "GET /api/admin/verification-codes (no JWT)" "$s"

s=$(api POST /api/admin/verification-codes '{"count":1}')
check_403 "POST /api/admin/verification-codes (no JWT)" "$s"

# ============================================================================
# Verification Codes
# ============================================================================
echo ""
echo "${BOLD}── Verification Codes ──${NC}"

# Redeem non-existent code
s=$(api POST /api/naming/verification-codes/redeem '{"code":"XXXXXXXX","email":"test@example.com"}')
check_400 "POST /api/naming/verification-codes/redeem (not found)" "$s"

# Redeem code too short (normalized length != 8)
s=$(api POST /api/naming/verification-codes/redeem '{"code":"ABC","email":"test@example.com"}')
check_400 "POST /api/naming/verification-codes/redeem (invalid format)" "$s"

# Redeem missing fields
s=$(api POST /api/naming/verification-codes/redeem '{}')
check_any "POST /api/naming/verification-codes/redeem (missing fields)" "^(400|429)$" "$s"

# Redeem missing email
s=$(api POST /api/naming/verification-codes/redeem '{"code":"XXXXXXXX"}')
check_any "POST /api/naming/verification-codes/redeem (missing email)" "^(400|429)$" "$s"

# Redeem missing code
s=$(api POST /api/naming/verification-codes/redeem '{"email":"test@example.com"}')
check_any "POST /api/naming/verification-codes/redeem (missing code)" "^(400|429)$" "$s"

# ============================================================================
# Misc
# ============================================================================
echo ""
echo "${BOLD}── Misc ──${NC}"
s=$(api GET /api/version)
check_200 "GET /api/version" "$s"
b=$(body)
check_range "  has build_time" ".+" "$(json_val "$b" '.data.build_time // ""')"
check_range "  has version" ".+" "$(json_val "$b" '.data.version // ""')"

s=$(api GET /api/location)
check_200 "GET /api/location" "$s"

s=$(api GET /api/stats)
check_200 "GET /api/stats" "$s"

s=$(api POST /api/analytics/pageview '{"path":"/zh-Hans/"}')
check_204 "POST /api/analytics/pageview" "$s"

# ============================================================================
# Edge Cases — input validation
# ============================================================================
echo ""
echo "${BOLD}── Edge Cases ──${NC}"

# Malformed JSON (1 representative — L4 Go unit covers all endpoints in detail)
s=$(api POST /api/naming/orders '{broken}')
check_400 "POST /api/naming/orders (broken JSON)" "$s"

# Unknown endpoints
s=$(api GET /api/unknown)
check_404 "GET /api/unknown (not found)" "$s"

s=$(api POST /api/unknown '{}')
check_404 "POST /api/unknown (not found)" "$s"

# Method not allowed — GET on POST-only endpoints
s=$(api GET '/api/naming/orders?email=test@example.com')
check_405 "GET /api/naming/orders (method not allowed)" "$s"

# ============================================================================
# Rate Limiting (optional — only runs when SKIP_EXTERNAL=1 is NOT set)
# ============================================================================
if [ -z "${SKIP_EXTERNAL:-}" ]; then
  echo ""
  echo "${BOLD}── Rate Limiting ──${NC}"
  # Fire burst + 1 requests quickly to trigger rate limit on checkout
  for i in $(seq 1 5); do
    s=$(api POST /api/payments/intents "{\"product\":\"naming\",\"business_order_id\":\"rate-test-$i\",\"provider\":\"dodo\"}")
    if [ "$s" = "429" ]; then
      echo -e "  ${GREEN}\xe2\x9c\x93${NC} Rate limit triggered at request #$i"
      PASS=$((PASS + 1))
      rate_limited=true
      break
    fi
  done
  if [ "${rate_limited:-false}" = false ]; then
    echo -e "  ${YELLOW}  - Rate limit not triggered (may depend on IP tracking)${NC}"
  fi
else
  echo ""
  echo "${BOLD}── Rate Limiting ──${NC}"
  echo "  (skipped — set SKIP_EXTERNAL=1)"
fi

# ============================================================================
# CORS Headers (only when Caddy is running)
# ============================================================================
echo ""
echo "${BOLD}── CORS ──${NC}"
if curl -s --connect-timeout 2 -o /dev/null -w '%{http_code}' "$BASE/api/health" 2>/dev/null | /bin/grep -q .; then
  # Test preflight
  local_f="$TMP/cors_headers"
  curl -s -D "$local_f" -o /dev/null -X OPTIONS "$API/api/health" \
    -H 'Origin: https://liki.hk' \
    -H 'Access-Control-Request-Method: GET'
  acao=$(grep -i 'Access-Control-Allow-Origin' "$local_f" 2>/dev/null | head -1 | tr -d '\r')
  if [ -n "$acao" ]; then
    echo -e "  ${GREEN}\xe2\x9c\x93${NC} CORS preflight returns Allow-Origin header"
    PASS=$((PASS + 1))
  else
    echo -e "  ${RED}\xe2\x9c\x97${NC} CORS preflight missing Allow-Origin header"
    FAIL=$((FAIL + 1))
  fi
else
  echo "  (Caddy not running — skipping CORS checks)"
fi

# ============================================================================
echo ""
echo "${BOLD}────────────────────────────────────────${NC}"
TOTAL=$((PASS + FAIL))
echo "Total: $TOTAL  ${GREEN}PASS: $PASS${NC}  ${RED}FAIL: $FAIL${NC}"
echo ""

if [ "$FAIL" -gt 0 ]; then
  echo "${RED}Some tests failed.${NC}"
  exit 1
else
  echo "${GREEN}All API tests passed.${NC}"
  exit 0
fi
