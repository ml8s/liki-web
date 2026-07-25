#!/bin/bash
# Build web static assets with Astro.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
export PATH="$HOME/.nvm/versions/node/v24.14.1/bin:$PATH"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BUILD_TS="${BUILD_TS:-$(date '+%s')}"

cd "$PROJECT_DIR/web"

echo "[build-web] Astro..."
npx astro build

cd "$PROJECT_DIR"
echo "[build-web] sitemap..."
node web/scripts/build-sitemap.cjs

