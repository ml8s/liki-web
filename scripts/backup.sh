#!/bin/bash
# backup.sh — SQLite WAL-mode backups (VACUUM INTO) for ALL product databases.
# Safe to run while the server is live (VACUUM INTO is a read-only operation
# that copies the database into a new file with WAL checkpointed in).
#
# 库清单与 cmd/liki/main.go 的默认值一一对齐（env 可覆盖）：
#   DB_PATH（naming）/ MINGSHU_DB_PATH / HEPAN_DB_PATH / FEEDBACK_DB_PATH
# 库文件不存在时跳过（开发机常只有部分库）——不视为错误。
#
# Usage:
#   scripts/backup.sh                    # keep 30 days, output to data/backups/
#   scripts/backup.sh /mnt/backups 7     # custom dir, 7-day retention
#
# Cron (daily at 03:07 UTC):
#   7 3 * * * cd /app && scripts/backup.sh >> /var/log/backup.log 2>&1

set -euo pipefail

BACKUP_DIR="${1:-data/backups}"
RETENTION_DAYS="${2:-30}"

# name:env 顺序表（name = 备份文件名前缀 + prune 匹配模式）
DBS=(
  "naming:DB_PATH"
  "mingshu:MINGSHU_DB_PATH"
  "hepan:HEPAN_DB_PATH"
  "feedback:FEEDBACK_DB_PATH"
)
# 与 main.go 默认一致（容器内为 /var/lib/liki/*，本地为 data/*）
declare -A DEFAULTS=(
  [DB_PATH]="data/liki.db"
  [MINGSHU_DB_PATH]="data/mingshu.db"
  [HEPAN_DB_PATH]="data/hepan.db"
  [FEEDBACK_DB_PATH]="data/feedback.db"
)

if ! command -v sqlite3 &>/dev/null; then
    echo "[backup] sqlite3 CLI not found — install with: apt install sqlite3"
    exit 1
fi

mkdir -p "$BACKUP_DIR"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
DONE=0; SKIPPED=0

for entry in "${DBS[@]}"; do
    name="${entry%%:*}"; env_key="${entry#*:}"
    db_path="${!env_key:-${DEFAULTS[$env_key]}}"

    if [ ! -f "$db_path" ]; then
        echo "[backup] $(date -u +%Y-%m-%dT%H:%M:%SZ) skip $name — $db_path 不存在"
        SKIPPED=$((SKIPPED+1))
        continue
    fi

    BACKUP_FILE="$BACKUP_DIR/${name}-${TIMESTAMP}.db"
    # Sanity-check: path must not contain single-quotes (would break VACUUM INTO).
    case "$BACKUP_FILE" in *"'"*) echo "[backup] ERROR: path contains single-quote"; exit 1 ;; esac

    echo "[backup] $(date -u +%Y-%m-%dT%H:%M:%SZ) $name ← $db_path (VACUUM INTO $BACKUP_FILE)"
    sqlite3 "$db_path" "VACUUM INTO '$BACKUP_FILE'"

    # Prune old backups（按本库前缀，互不误删）
    find "$BACKUP_DIR" -name "${name}-*.db" -mtime "+${RETENTION_DAYS}" -delete
    DONE=$((DONE+1))
done

echo "[backup] $(date -u +%Y-%m-%dT%H:%M:%SZ) 完成：$DONE 个库备份，$SKIPPED 个跳过 → $BACKUP_DIR（保留 ${RETENTION_DAYS}d）"
