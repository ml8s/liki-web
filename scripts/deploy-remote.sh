#!/bin/bash
# Upload images + configs and deploy on remote server.
# Usage: scripts/deploy-remote.sh --server HOST --user USER --domain DOMAIN --image PATH [--env .env]
set -eo pipefail

SERVER=""; USER=""; DOMAIN=""; IMAGE_FILE=""; ENV_FILE=""

while [ $# -gt 0 ]; do
  case "$1" in
    --server)    SERVER="$2"; shift 2 ;;
    --user)      USER="$2"; shift 2 ;;
    --domain)    DOMAIN="$2"; shift 2 ;;
    --image)     IMAGE_FILE="$2"; shift 2 ;;
    --env)       ENV_FILE="$2"; shift 2 ;;
    *) echo "Usage: $0 --server HOST --user USER --domain DOMAIN --image PATH [--env .env]" >&2; exit 1 ;;
  esac
done

for var in SERVER USER DOMAIN IMAGE_FILE; do
  if [ -z "${!var}" ]; then
    echo "ERROR: --${var,,} is required" >&2; exit 1
  fi
done

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# SSH setup
if ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=accept-new "$USER@$SERVER" "echo ok" >/dev/null 2>&1; then
  SSH_CMD="ssh"
  SCP_CMD="scp"
else
  read -rsp "请输入 $DOMAIN 服务器密码: " SSHPASS
  echo ""
  SSH_CMD="sshpass -e ssh -o StrictHostKeyChecking=accept-new"
  SCP_CMD="sshpass -e scp -o StrictHostKeyChecking=accept-new"
  export SSHPASS
fi

# Upload configs
echo "[deploy-remote] 上传配置..."
if [ -n "$ENV_FILE" ] && [ -f "$ENV_FILE" ]; then
  ENV_DIR="$(cd "$(dirname "$ENV_FILE")" && pwd)"
  ENV_NAME="$(basename "$ENV_FILE")"
  tar czf /tmp/liki_configs.tar.gz \
    -C "$PROJECT_DIR/deploy/liki" docker-compose.yml Caddyfile \
    -C "$ENV_DIR" "$ENV_NAME"
else
  tar czf /tmp/liki_configs.tar.gz \
    -C "$PROJECT_DIR/deploy/liki" docker-compose.yml Caddyfile
fi
$SCP_CMD /tmp/liki_configs.tar.gz "$USER@$SERVER:/tmp/liki_configs.tar.gz"
rm -f /tmp/liki_configs.tar.gz

echo "[deploy-remote] 上传镜像 ($(du -h "$IMAGE_FILE" | cut -f1))..."
$SCP_CMD "$IMAGE_FILE" "$USER@$SERVER:/tmp/liki_images.tar.gz"

# Remote deploy
echo "[deploy-remote] 服务器部署..."
$SSH_CMD "$USER@$SERVER" "DOMAIN='$DOMAIN' bash -s" << 'ENDSSH'
set -e
docker ps >/dev/null 2>&1 && D="docker" || D="sudo -E docker"

sudo mkdir -p /opt/liki
sudo chown "$(whoami)" /opt/liki

# Stop old
if [ -f /opt/liki/docker-compose.yml ]; then
  cd /opt/liki && $D compose down --timeout 10 2>&1 || true
fi

# Release ports 80/443
for port in 80 443; do
  for i in $(seq 1 5); do
    cid=$($D ps -q --filter "publish=$port" 2>/dev/null || true)
    if [ -n "$cid" ]; then
      echo "强制停止占用端口 $port 的容器: $cid"
      $D stop "$cid" 2>/dev/null || true
      $D rm -f "$cid" 2>/dev/null || true
      sleep 1
      continue
    fi
    pid=$(sudo lsof -ti :$port 2>/dev/null || true)
    if [ -n "$pid" ]; then
      echo "清除占用端口 $port 的旧进程: $pid"
      sudo kill -9 $pid 2>/dev/null || true
      sleep 1
      continue
    fi
    break
  done
done

# Extract configs
tar xzf /tmp/liki_configs.tar.gz -C /tmp/
mv /tmp/docker-compose.yml /opt/liki/docker-compose.yml
mv /tmp/Caddyfile /opt/liki/Caddyfile
[ -f /tmp/.env ] && mv /tmp/.env /opt/liki/.env
rm -f /tmp/liki_configs.tar.gz

# DB volume
$D volume create liki_data 2>/dev/null || true
$D run --rm -v liki_data:/data alpine chown 1000:1000 /data

cd /opt/liki
$D load < /tmp/liki_images.tar.gz
$D compose up -d
$D image prune -f

echo "等待服务就绪..."
sleep 5
$D compose ps
rm -f /tmp/liki_images.tar.gz
ENDSSH

unset SSHPASS

# Health check — API
echo "[deploy-remote] 健康检查 (https://${DOMAIN}/api/health)..."
for i in $(seq 1 6); do
  st=$(curl -sf -o /dev/null -w "%{http_code}" --connect-timeout 5 "https://${DOMAIN}/api/health" 2>&1 || true)
  if [ "$st" = "200" ]; then
    echo "API 健康检查通过 ($DOMAIN)"
    break
  fi
  if [ "$i" -eq 6 ]; then
    echo "API 健康检查失败 ($DOMAIN), HTTP $st" >&2
    exit 1
  fi
  echo "  HTTP $st, 重试 ($i/6)..."
  sleep 5
done

# Health check — locale pages (catch missing static files after deploy)
echo "[deploy-remote] 各语言首页检查..."
for locale in en zh-Hans zh-Hant; do
  st=$(curl -sf -o /dev/null -w "%{http_code}" --connect-timeout 5 "https://${DOMAIN}/${locale}/" 2>&1 || true)
  if [ "$st" != "200" ]; then
    echo "  $locale 首页返回 $st" >&2
    exit 1
  fi
  echo "  /${locale}/ ✓ 200"
done

echo "部署完成 ($DOMAIN)。"
