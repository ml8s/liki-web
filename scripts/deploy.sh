#!/bin/bash
set -euo pipefail
# 部署到指定区域：us / cn

REGION="${1:-us}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR"

# 更新 skills 源码
echo "── pull skills ──"
cd ../liki && git pull --ff-only origin master && cd "$PROJECT_DIR"

# 构建镜像
echo "── build docker ──"
scripts/build-docker.sh --compose-file deploy/liki/docker-compose.yml \
  --export /tmp/liki_images.tar.gz --services web engine

# 读取 .env
if [ ! -f .env ]; then
  echo "❌ .env 不存在" >&2
  exit 1
fi

case "$REGION" in
  us)
    SERVER="${SERVER_US:-43.130.2.209}"; USER="ubuntu"
    DOMAIN=$(grep -oP '^DOMAIN_US=\K.*' .env 2>/dev/null || echo "liki.hk")
    ;;
  cn)
    SERVER="${SERVER_CN:-120.79.194.247}"; USER="root"
    DOMAIN=$(grep -oP '^DOMAIN_CN=\K.*' .env 2>/dev/null || echo "liki.hk")
    ;;
  *) echo "Usage: scripts/deploy.sh us|cn" >&2; exit 1 ;;
esac

echo "── deploy $REGION ($DOMAIN) ──"
DEPLOY_ENV_DIR=$(mktemp -d)
cp .env "$DEPLOY_ENV_DIR/.env"
if [ -f .env.production.alipay ]; then
  cat .env.production.alipay >> "$DEPLOY_ENV_DIR/.env"
  echo "✓ merged .env.production.alipay into deploy env"
fi
scripts/deploy-remote.sh --server "$SERVER" --user "$USER" --domain "$DOMAIN" \
  --image /tmp/liki_images.tar.gz --env "$DEPLOY_ENV_DIR/.env"
rm -rf "$DEPLOY_ENV_DIR"
rm -f /tmp/liki_images.tar.gz
echo "✅ deployed to $REGION"
