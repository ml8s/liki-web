#!/bin/bash
set -euo pipefail
# Caddyfile 格式校验（caddy 缺失或格式错误 → exit 1）

if ! command -v caddy >/dev/null 2>&1; then
  echo "❌ caddy not found — install it or run in CI" >&2
  exit 1
fi

FAILED=0
for f in deploy/liki/Caddyfile deploy/liki/Caddyfile.local; do
  if ! caddy fmt "$f" --diff >/dev/null 2>&1; then
    echo "❌ $f: format check failed"
    caddy fmt "$f" --diff 2>&1 || true
    FAILED=1
  else
    echo "✓ $f"
  fi
done
exit $FAILED
