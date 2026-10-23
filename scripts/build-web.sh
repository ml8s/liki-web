#!/bin/bash
# Build web static assets with Astro.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# Use project Node version (defined in .nvmrc)
source "$SCRIPT_DIR/node-path.sh"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR/web"

echo "[build-web] public/ contents: $(ls public/ | tr '\n' ' ')"

echo "[build-web] shared chart messages..."
node scripts/check-chart-messages.js

echo "[build-web] Astro..."
npx astro build

echo "[build-web] Liki App (Vue 3 SPA)..."
cd app
if [ ! -d node_modules ]; then
  npm ci --ignore-scripts
fi
npm run build
cd "$PROJECT_DIR/web"
node scripts/check-inline-scripts.js --dist dist
cd "$PROJECT_DIR"

echo "[build-web] sitemap..."
node web/scripts/build-sitemap.js
