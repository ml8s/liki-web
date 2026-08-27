#!/bin/bash
# Sync the liki skill repo to the web serving directory.
# 4 independent skills: liki-bazi / liki-divination / liki-fengshui / liki-naming
#   (each = SKILL.md + app/ + domains/ + tools/ + VERSION + content.sha256)
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
WELL_KNOWN_DIR="$PROJECT_DIR/web/.well-known/agent-skills"

SKILLS=(liki-bazi liki-divination liki-fengshui liki-naming)

mkdir -p "$WELL_KNOWN_DIR"

# ── 清理旧的单 skill 副本（历史遗留 liki/）──
rm -rf "$PROJECT_DIR/$DEST/liki"

for SKILL in "${SKILLS[@]}"; do
  SRC_DIR="$SRC_ROOT/$SKILL"
  DEST_DIR="$PROJECT_DIR/$DEST/$SKILL"

  if [ ! -d "$SRC_DIR" ]; then
    echo "  WARNING: source directory not found: $SRC_DIR" >&2
    continue
  fi
  mkdir -p "$DEST_DIR"
  echo "==> $SKILL (from $SRC_DIR)"

  # ── 1. Sync source files for web access (SKILL.md/app/domains/tools/VERSION) ──
  # 排除工程文件/测试/缓存——web/skills 目录 = 随包分发内容，与 build-archive.sh 口径一致
  rsync -a --no-group --delete \
    --exclude .git \
    --exclude .git-old \
    --exclude .git-fresh \
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

  # ── 1.5 Web 部署专用：webapp 提示词（仅 liki-bazi 的 mingshu/hepan 生成流水线）──
  # webapp 不进 GitHub 安装包，但 liki-web 后端从 web/skills/liki-bazi/webapp/{mingshu,hepan}
  # 加载生成提示词——部署形态必须补上。
  if [ "$SKILL" = "liki-bazi" ]; then
    WEBAPP_SRC="$PROJECT_DIR/../liki/webapp"
    if [ -d "$WEBAPP_SRC" ]; then
      rsync -a --delete "$WEBAPP_SRC/" "$DEST_DIR/webapp/"
      echo "  ✓ webapp synced (mingshu/hepan prompts)"
    else
      echo "  WARNING: webapp source not found: $WEBAPP_SRC" >&2
    fi
  fi

  # ── 2. Copy pre-built archive from dist/ ──
  if [ -f "$DIST_DIR/$SKILL.tar.gz" ]; then
    cp "$DIST_DIR/$SKILL.tar.gz" "$PROJECT_DIR/$DEST/$SKILL.tar.gz"
    echo "  ✓ archive synced: $SKILL.tar.gz"
  else
    echo "  WARNING: dist/$SKILL.tar.gz not found. Run: cd "$PROJECT_DIR/../liki" && make build-archive" >&2
  fi
done

# ── 3. index.json（4 skill 索引）──
if [ -f "$DIST_DIR/index.json" ]; then
  cp "$DIST_DIR/index.json" "$WELL_KNOWN_DIR/index.json"
  echo "  ✓ index.json: $WELL_KNOWN_DIR/index.json"
else
  echo "  WARNING: dist/index.json not found" >&2
  exit 1
fi
