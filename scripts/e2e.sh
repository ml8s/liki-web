#!/bin/bash
set -euo pipefail
# Docker 全栈 E2E：up → api + ui + pytest → down

SKILLS_DIR="$(cd "$(dirname "$0")/.." && pwd)/../liki"

echo "── Docker up ──"
scripts/docker-test.sh up || { scripts/docker-test.sh down 2>/dev/null; echo "❌ Docker 启动失败"; exit 1; }

cleanup() { scripts/docker-test.sh down 2>/dev/null || true; }
trap cleanup EXIT

echo "── API smoke ──"
scripts/test-api.sh http://localhost:8080 http://localhost:8080

echo "── Playwright E2E ──"
cd web && BASE_URL=http://localhost:8080 npx playwright test --config e2e/playwright.config.js
cd ..

echo "── Skill pytest ──"
cd "$SKILLS_DIR" && LIKI_RPC_URL=http://localhost:8080/jsonrpc python3 -m pytest tests/ -q

echo ""
echo "✅ e2e 全部通过"
