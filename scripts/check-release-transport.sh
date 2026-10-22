#!/usr/bin/env bash
# Fail the source gate if the release chain can publish a relay-only image.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

workflow=".github/workflows/release.yml"
release_script="scripts/release-images.sh"

grep -Fq 'VITE_CHAT_TRANSPORT: direct' "$workflow" ||
  { echo "❌ release workflow does not set VITE_CHAT_TRANSPORT=direct" >&2; exit 1; }

if env -u VITE_CHAT_TRANSPORT "$release_script" --check-transport >/dev/null 2>&1; then
  echo "❌ release script accepts an unset transport" >&2
  exit 1
fi

if VITE_CHAT_TRANSPORT=relay "$release_script" --check-transport >/dev/null 2>&1; then
  echo "❌ release script accepts VITE_CHAT_TRANSPORT=relay" >&2
  exit 1
fi

VITE_CHAT_TRANSPORT=direct "$release_script" --check-transport
