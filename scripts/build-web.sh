#!/bin/bash
# Build web static assets with Astro.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# Use project Node version (defined in .nvmrc)
source "$SCRIPT_DIR/node-path.sh"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BUILD_TS="${BUILD_TS:-$(date '+%s')}"

cd "$PROJECT_DIR/web"

# Astro/vite 在某些环境（CI runner）会 stat web/public/skills；本地该目录由测试误创建。
# 确保存在（静态资源目录，build 时 Astro 会拷贝 public/）——无内容也无碍（产物无 /skills/ 资源引用）。
mkdir -p public/skills
[ -d public/skills ] || { echo "[build-web] ✗ public/skills 不可用"; exit 1; }
echo "[build-web] public/ contents: $(ls public/ | tr '\n' ' ')"

echo "[build-web] Astro..."
npx astro build

cd "$PROJECT_DIR"
echo "[build-web] sitemap..."
node web/scripts/build-sitemap.cjs

