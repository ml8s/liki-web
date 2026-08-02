#!/bin/bash
# Sync skill repos to web serving directory.
# Sync sub-skills from the aggregate liki repo (liki-master / liki-ask / liki-fengshui / liki-naming).
# Usage: scripts/sync-skills.sh [--repos "liki-master liki-ask liki-fengshui liki-naming"] [--dest web/skills]  # 源在 ../skills/liki/<sub>
set -euo pipefail

REPOS="${REPOS:-liki-master liki-ask liki-fengshui liki-naming}"  # 聚合库 liki 下的四子 skill
DEST="${DEST:-web/skills}"

while [ $# -gt 0 ]; do
  case "$1" in
    --repos) REPOS="$2"; shift 2 ;;
    --dest)  DEST="$2"; shift 2 ;;
    *) echo "Usage: $0 [--repos \"liki-master liki-ask ...\"] [--dest DIR]" >&2; exit 1 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

DEST_DIR="$PROJECT_DIR/$DEST"
WELL_KNOWN_DIR="$PROJECT_DIR/web/.well-known/agent-skills"
mkdir -p "$DEST_DIR" "$WELL_KNOWN_DIR"

# Merge index.json across all repos
: > "$WELL_KNOWN_DIR/index.json"

for repo in $REPOS; do
  SRC_DIR="$PROJECT_DIR/../skills/liki/$repo"  # 聚合库 liki 下子目录
  echo "==> $repo"

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
      "$SRC_DIR/" "$DEST_DIR/$repo/"
  else
    echo "  WARNING: source directory not found: $SRC_DIR" >&2
    continue
  fi

  # ── 2. Copy pre-built archive from aggregate dist/ (../skills/liki/dist/) ──
  AGG_DIST="$PROJECT_DIR/../skills/liki/dist"
  if [ -f "$AGG_DIST/$repo.tar.gz" ] && [ -f "$AGG_DIST/index.json" ]; then
    cp "$AGG_DIST/$repo.tar.gz" "$DEST_DIR/$repo.tar.gz"
    # merge this repo's index entry into the combined index
    python3 - "$WELL_KNOWN_DIR/index.json" "$AGG_DIST/index.json" << 'PYEOF'
import json, sys
combined_path, single_path = sys.argv[1], sys.argv[2]
try:
    with open(combined_path) as f:
        combined = json.load(f)
except (FileNotFoundError, json.JSONDecodeError):
    combined = {"$schema": "https://schemas.agentskills.io/discovery/0.2.0/schema.json", "skills": []}
with open(single_path) as f:
    single = json.load(f)
known = {s["name"] for s in combined["skills"]}
for s in single["skills"]:
    if s["name"] not in known:
        combined["skills"].append(s)
with open(combined_path, "w") as f:
    json.dump(combined, f, ensure_ascii=False, indent=2)
PYEOF
    echo "  ✓ archive synced: $repo.tar.gz"
  else
    echo "  WARNING: dist/$repo.tar.gz not found in $SRC_DIR" >&2
    echo "  Run: cd $repo && make build-archive" >&2
  fi
done

echo "  ✓ combined index: $WELL_KNOWN_DIR/index.json"
