#!/bin/bash
# Build web static assets with Astro.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# Use project Node version (defined in .nvmrc)
source "$SCRIPT_DIR/node-path.sh"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR/web"

# Astro/vite 在某些环境（CI runner）会 stat web/public/skills；本地该目录由测试误创建。
# 确保存在（静态资源目录，build 时 Astro 会拷贝 public/）——无内容也无碍（产物无 /skills/ 资源引用）。
mkdir -p public/skills
[ -d public/skills ] || { echo "[build-web] ✗ public/skills 不可用"; exit 1; }
echo "[build-web] public/ contents: $(ls public/ | tr '\n' ' ')"

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
