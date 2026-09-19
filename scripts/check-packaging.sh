#!/bin/bash
set -euo pipefail
# 模拟部署打包：cp web/ → tmp，验证关键文件存在

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

cp -rL web/. "$TMP/"
echo "  files: $(find "$TMP" -type f | wc -l)"

for f in .well-known/agent-skills/index.json skills/liki.tar.gz skills/liki/SKILL.md; do
  if [ ! -f "$TMP/$f" ]; then
    echo "❌ packaging missing: $f"
    exit 1
  fi
done
echo "✓ packaging OK"
