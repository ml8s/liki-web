#!/usr/bin/env bash
set -euo pipefail
# Local full-stack browser E2E. External Engine inputs are immutable artifacts;
# this repository does not checkout or test the liki source tree.

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# The release matrix covers the Consultation-default experience. Production can
# still roll back with the same flag set to false; Docker E2E deliberately
# tests the forward surface instead of silently degrading into Classic specs.
export VITE_FEATURE_APP_FLOW_REDESIGN="${VITE_FEATURE_APP_FLOW_REDESIGN:-true}"

echo "── Docker up ──"
scripts/docker-test.sh up || { scripts/docker-test.sh down 2>/dev/null || true; echo "❌ Docker 启动失败"; exit 1; }

# Cleanup must not depend on the script's cwd (playwright runs from web/).
cleanup() { (cd "$ROOT" && scripts/docker-test.sh down) 2>/dev/null || true; }
trap cleanup EXIT

echo "── API smoke ──"
scripts/test-api.sh http://localhost:8080 http://localhost:8080

echo "── Playwright E2E ──"
cd web && BASE_URL=http://localhost:8080 npx playwright test --config e2e/playwright.config.ts --grep-invert "@system" ${E2E_GREP:+--grep "$E2E_GREP"}

echo ""
echo "✅ e2e 全部通过"
