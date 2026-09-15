#!/bin/bash
# Sync the liki skill repo to the web serving directory.
# Single unified skill: liki (SKILL.md router + bazi/divination/fengshui/naming domains,
# each = ENTRY.md + app/ + domains/ + tools/; single root VERSION/feedback).
# Usage: scripts/sync-skills.sh [--dest web/skills]
set -euo pipefail

DEST="${DEST:-web/skills}"

while [ $# -gt 0 ]; do
  case "$1" in
    --dest)  DEST="$2"; shift 2 ;;
    *) echo "Usage: $0 [--dest DIR]" >&2; exit 1 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# skills 仓根（ml8s/liki 的 clone 目录，与仓库名一致）
SRC_ROOT="$PROJECT_DIR/../liki/skills"
DIST_DIR="$PROJECT_DIR/../liki/dist"   # 构建产物在 liki 仓库根
WEBAPP_SRC="$PROJECT_DIR/../liki/webapp"
WELL_KNOWN_DIR="$PROJECT_DIR/web/.well-known/agent-skills"

SKILL="liki"
SRC_DIR="$SRC_ROOT/$SKILL"
DEST_DIR="$PROJECT_DIR/$DEST/$SKILL"

mkdir -p "$WELL_KNOWN_DIR"

if [ ! -d "$SRC_DIR" ]; then
  echo "[sync-skills] ✗ 源目录不存在: $SRC_DIR" >&2
  exit 1
fi

# ── 清理旧的四 skill 副本 + 历史归档（合并为单一 skill 前的遗留）──
for OLD in liki-bazi liki-divination liki-fengshui liki-naming; do
  rm -rf "$PROJECT_DIR/$DEST/$OLD" "$PROJECT_DIR/$DEST/$OLD.tar.gz"
done

echo "==> $SKILL (from $SRC_DIR)"
mkdir -p "$DEST_DIR"

# ── 1. 同步 skill 源文件（随包分发内容，与 build-archive.sh 口径一致）──
rsync -a --no-group --delete \
  --exclude .git \
  --exclude .github \
  --exclude .githooks \
  --exclude './.gitignore' \
  --exclude .claude \
  --exclude .reasonix \
  --exclude .pytest_cache \
  --exclude __pycache__ \
  --exclude './reasonix.toml' \
  --exclude './README.md' \
  --exclude './README.en.md' \
  --exclude './CONTRIBUTING.md' \
  --exclude './CHANGELOG.md' \
  --exclude '*.tar.gz' \
  --exclude dist \
  --exclude './Makefile' \
  --exclude './AGENTS.md' \
  --exclude './LICENSE' \
  --exclude './pytest.ini' \
  --exclude webapp \
  --exclude tests \
  --exclude scripts \
  "$SRC_DIR/" "$DEST_DIR/"

# ── 1.5 Web 部署专用：报告生成提示词（mingshu/hepan）──
# webapp 不进安装包，但 liki-web 后端从 web/skills/liki/webapp/{mingshu,hepan} 加载。
# 桥接：报告提示词仍引用合并前的路径（liki-bazi/domains、liki-bazi/webapp），
# 同步时机械改写为新结构（幂等；引擎侧修好后本步自然空转）。
if [ -d "$WEBAPP_SRC" ]; then
  rsync -a --delete "$WEBAPP_SRC/" "$DEST_DIR/webapp/"
  find "$DEST_DIR/webapp" -type f -name '*.md' -print0 |
    xargs -0 -r sed -i \
      -e 's|liki-bazi/domains/|liki/bazi/domains/|g' \
      -e 's|liki-bazi/webapp/|liki/webapp/|g'
  echo "  ✓ webapp synced (mingshu/hepan prompts, paths bridged)"
else
  echo "  WARNING: webapp source not found: $WEBAPP_SRC" >&2
fi

# ── 2. 归档 ──
if [ -f "$DIST_DIR/$SKILL.tar.gz" ]; then
  cp "$DIST_DIR/$SKILL.tar.gz" "$PROJECT_DIR/$DEST/$SKILL.tar.gz"
  echo "  ✓ archive synced: $SKILL.tar.gz"
else
  echo "  WARNING: dist/$SKILL.tar.gz not found. Run: cd "$PROJECT_DIR/../liki" && make build-archive" >&2
fi

# ── 3. index.json（单一 skill 索引）──
if [ -f "$DIST_DIR/index.json" ]; then
  cp "$DIST_DIR/index.json" "$WELL_KNOWN_DIR/index.json"
  echo "  ✓ index.json: $WELL_KNOWN_DIR/index.json"
else
  echo "  WARNING: dist/index.json not found" >&2
  exit 1
fi
