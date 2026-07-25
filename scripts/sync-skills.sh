#!/bin/bash
# Sync skill files from source repository to web serving directory.
# Usage: scripts/sync-skills.sh [--src ../liki] [--dest web/skills]
set -euo pipefail

SRC="${SRC:-../liki}"
DEST="${DEST:-web/skills}"

while [ $# -gt 0 ]; do
  case "$1" in
    --src)  SRC="$2"; shift 2 ;;
    --dest) DEST="$2"; shift 2 ;;
    *) echo "Usage: $0 [--src DIR] [--dest DIR]" >&2; exit 1 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

SRC_DIR="$PROJECT_DIR/$SRC"
DEST_DIR="$PROJECT_DIR/$DEST"
WELL_KNOWN_DIR="$PROJECT_DIR/web/.well-known/agent-skills"

# ── 1. Sync source files for web access ──
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
    --exclude VERSION \
    --exclude AGENTS.md \
    --exclude LICENSE \
    "$SRC_DIR/" "$DEST_DIR/"
else
  echo "WARNING: source directory not found: $SRC_DIR" >&2
fi

# ── 2. Copy pre-built archive from source ──
if [ -f "$SRC_DIR/dist/liki.tar.gz" ] && [ -f "$SRC_DIR/dist/index.json" ]; then
  cp "$SRC_DIR/dist/liki.tar.gz" "$DEST_DIR/liki.tar.gz"
  mkdir -p "$WELL_KNOWN_DIR"
  cp "$SRC_DIR/dist/index.json" "$WELL_KNOWN_DIR/index.json"
  echo "  ✓ skills archive synced from $SRC/dist/"
else
  echo "WARNING: pre-built archive not found in $SRC/dist/" >&2
  echo "  Run: cd $SRC && make build-archive" >&2
fi
