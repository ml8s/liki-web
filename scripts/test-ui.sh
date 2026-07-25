#!/bin/bash
# 浏览器 E2E 测试 — 全部 Playwright specs
set -eo pipefail
URL="${1:-http://localhost:8080}"
cd "$(dirname "$0")/../web"
BASE_URL="$URL" npx playwright test --config e2e/playwright.config.js
