#!/bin/bash
# Sync the liki skill repo to the web serving directory.
# liki is a single self-contained skill (SKILL.md + app/ + domains/ + webapp/).
# Usage: scripts/sync-skills.sh [--dest web/skills]
set -euo pipefail

SKILL="${SKILL:-liki}"
DEST="${DEST:-web/skills}"

while [ $# -gt 0 ]; do
  case "$1" in
    --dest)  DEST="$2"; shift 2 ;;
    *) echo "Usage: $0 [--dest DIR]" >&2; exit 1 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

SRC_DIR="$PROJECT_DIR/../skills/liki"
DEST_DIR="$PROJECT_DIR/$DEST/$SKILL"
WELL_KNOWN_DIR="$PROJECT_DIR/web/.well-known/agent-skills"
DIST_DIR="$SRC_DIR/dist"

mkdir -p "$DEST_DIR" "$WELL_KNOWN_DIR"

echo "==> $SKILL (from $SRC_DIR)"

# ── 1. Sync source files for web access (SKILL.md/app/domains/webapp/VERSION) ──
if [ -d "$SRC_DIR" ]; then
  rsync -a --no-group --delete \
    --exclude .git \
    --exclude .git-old \
    --exclude .git-fresh \
    --exclude .claude \
    --exclude .reasonix \
    --exclude reasonix.toml \
    --exclude README.md \
    --exclude README.en.md \
    --exclude CONTRIBUTING.md \
    --exclude CHANGELOG.md \
    --exclude '*.tar.gz' \
    --exclude dist \
    --exclude Makefile \
    --exclude AGENTS.md \
    --exclude LICENSE \
    --exclude scripts \
    "$SRC_DIR/" "$DEST_DIR/"
else
  echo "  WARNING: source directory not found: $SRC_DIR" >&2
  exit 1
fi

# ── 2. Copy pre-built archive from dist/ ──
if [ -f "$DIST_DIR/$SKILL.tar.gz" ] && [ -f "$DIST_DIR/index.json" ]; then
  cp "$DIST_DIR/$SKILL.tar.gz" "$PROJECT_DIR/$DEST/$SKILL.tar.gz"
  cp "$DIST_DIR/index.json" "$WELL_KNOWN_DIR/index.json"
  echo "  ✓ archive synced: $SKILL.tar.gz"
  echo "  ✓ index.json: $WELL_KNOWN_DIR/index.json"
else
  echo "  WARNING: dist/$SKILL.tar.gz not found. Run: cd ../skills/liki && make build-archive" >&2
  exit 1
fi
