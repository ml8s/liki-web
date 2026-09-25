#!/bin/bash
set -euo pipefail
# skill 结构验证（统一检查本项目 web/skills/liki/）

FAILED=0
check() {
  if [ ! -e "$1" ]; then
    echo "❌ Missing $1"
    FAILED=1
  fi
}

check "web/skills/liki/SKILL.md"
check "web/skills/liki/VERSION.txt"
check "web/skills/liki/natal/TOOLS.md"
check "web/skills/liki/divination/TOOLS.md"
check "web/skills/liki/webapp/mingshu/generate.md"

if [ $FAILED -eq 0 ]; then
  echo "✓ skill structure OK"
fi
exit $FAILED
