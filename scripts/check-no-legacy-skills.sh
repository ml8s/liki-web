#!/usr/bin/env bash
# check-no-legacy-skills.sh — 防止旧四 skill（liki-bazi / liki-naming / liki-divination / liki-fengshui）
# 定义或引用回潮。统一为单一 liki skill 后，这些标识不应出现在任何已跟踪文件里。
# 用 git grep（仅已跟踪文件），构建产物（web/skills、web/dist、node_modules 等）天然被排除。
set -eo pipefail

cd "$(git rev-parse --show-toplevel)"

PATTERNS='liki-bazi|liki-naming|liki-divination|liki-fengshui|SkillNaming|SkillBazi|SkillDivination|SkillFengshui|four skills|四个 skill|四个内部 skill|4 sub-skills'

# CHANGELOG.md 为历史版本记录，旧条目保留原状（改写会歪曲历史），故豁免。
# 本脚本自身需要点名旧标识以描述防的是什么，故豁免自身。
MATCHES="$(git grep -nE "$PATTERNS" -- \
  ':!node_modules' ':!web/dist' ':!CHANGELOG.md' \
  ':!scripts/check-no-legacy-skills.sh' 2>/dev/null || true)"
if [ -n "$MATCHES" ]; then
  echo "❌ 检测到旧四 skill 引用（统一 liki skill 后不应存在）："
  echo "$MATCHES"
  echo ""
  echo "  清理方式：git grep -nE '$PATTERNS' 定位后逐一改写为统一 liki；"
  echo "  若确有历史语义需保留（如 gitignore/备份），加进本脚本排除清单并注明原因。"
  exit 1
fi

# 已同步的 web/skills 也不允许残留旧 skill 目录/归档（防御性，构建产物但易被旧 sync 遗留）。
for OLD in liki-bazi liki-divination liki-fengshui liki-naming; do
  if [ -e "web/skills/$OLD" ] || [ -e "web/skills/$OLD.tar.gz" ]; then
    echo "❌ web/skills/ 残留旧 skill 产物：$OLD"
    exit 1
  fi
done

echo "✓ 无旧四 skill 残留引用"