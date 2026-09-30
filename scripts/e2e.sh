#!/usr/bin/env bash
set -euo pipefail
# Local full-stack browser E2E. External Engine and skill inputs are immutable
# artifacts; this repository does not checkout or test the liki source tree.

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "── Docker up ──"
scripts/docker-test.sh up || { scripts/docker-test.sh down 2>/dev/null || true; echo "❌ Docker 启动失败"; exit 1; }

# Cleanup must not depend on the script's cwd (playwright runs from web/).
cleanup() { (cd "$ROOT" && scripts/docker-test.sh down) 2>/dev/null || true; }
trap cleanup EXIT

echo "── Verify prepared skill artifact ──"
scripts/verify-prepared-skills.sh

echo "── API smoke ──"
scripts/test-api.sh http://localhost:8080 http://localhost:8080

echo "── Playwright E2E ──"
cd web && BASE_URL=http://localhost:8080 npx playwright test --config e2e/playwright.config.js ${E2E_GREP:+--grep "$E2E_GREP"}

echo ""
echo "✅ e2e 全部通过"
