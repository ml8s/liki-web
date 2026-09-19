#!/bin/bash
set -euo pipefail
# 部署后冒烟验证：health + 首页 200

URL="${1:-http://localhost:8080}"
FAILED=0

check() {
  local name="$1" url="$2"
  if curl -sf -o /dev/null --max-time 10 "$url"; then
    echo "✓ $name ($url)"
  else
    echo "❌ $name ($url)"
    FAILED=1
  fi
}

check "health" "$URL/api/health"
check "homepage" "$URL/zh-Hans/"

if [ $FAILED -eq 0 ]; then
  echo "✅ smoke 通过"
else
  echo "❌ smoke 失败 — 考虑回滚或检查服务器配置"
fi
exit $FAILED
