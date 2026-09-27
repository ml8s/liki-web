#!/usr/bin/env bash
set -euo pipefail

# Verify the immutable liki skill artifact prepared by .github/actions/prepare-skills.
# liki-web never rebuilds or mutates the producer artifact.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

for path in \
  web/skills/liki/SKILL.md \
  web/skills/liki/VERSION.txt \
  web/skills/liki/natal/app/mingshu.md \
  web/skills/liki/webapp/mingshu/generate.md \
  web/skills/liki.tar.gz \
  web/.well-known/agent-skills/index.json; do
  test -f "$path" || { echo "❌ Missing prepared skill artifact: $path" >&2; exit 1; }
done

python3 - <<'PY'
import hashlib
import json
import pathlib

archive = pathlib.Path("web/skills/liki.tar.gz")
actual = "sha256:" + hashlib.sha256(archive.read_bytes()).hexdigest()
discovery = json.loads(pathlib.Path("web/.well-known/agent-skills/index.json").read_text(encoding="utf-8"))
skills = discovery.get("skills", [])
if not skills or any(item.get("digest") != actual for item in skills):
    raise SystemExit("❌ Prepared liki.tar.gz does not match discovery index digest")
manifest_path = pathlib.Path("web/.well-known/agent-skills/liki-manifest.json")
if manifest_path.is_file():
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    if manifest.get("archive_sha256") != actual:
        raise SystemExit("❌ Prepared liki.tar.gz does not match manifest digest")
print("✓ Prepared liki skill artifact is internally consistent")
PY
