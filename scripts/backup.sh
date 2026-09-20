#!/bin/bash
# backup.sh — SQLite WAL-mode backups (VACUUM INTO) for all Liki databases.
# VACUUM INTO is an online, read-only backup operation and includes a
# checkpointed view of the database, so WAL files are not needed in the copy.
#
# Database names match cmd/liki/main.go:
#   platform, profiles, feedback, liki, naming, mingshu, hepan, aipay, referral
# Missing files are skipped (for example, referral.db may not exist yet).
#
# Usage:
#   scripts/backup.sh                    # keep 30 days, output to data/backups/
#   scripts/backup.sh /mnt/backups 7     # custom dir, 7-day retention
#
# Cron (daily at 03:07 UTC):
#   7 3 * * * cd /app && scripts/backup.sh >> /var/log/backup.log 2>&1

set -euo pipefail

DATA_DIR="${LIKI_DATA_DIR:-data}"
BACKUP_DIR="${1:-data/backups}"
RETENTION_DAYS="${2:-30}"

DATABASES=(
  platform
  profiles
  feedback
  liki
  naming
  mingshu
  hepan
  aipay
  referral
)

if ! command -v sqlite3 >/dev/null 2>&1; then
    echo "[backup] sqlite3 CLI not found — install with: apt install sqlite3"
    exit 1
fi

for path in "$DATA_DIR" "$BACKUP_DIR"; do
    case "$path" in *"'"*) echo "[backup] ERROR: path contains single-quote: $path"; exit 1 ;; esac
done

mkdir -p "$BACKUP_DIR"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
DONE=0
SKIPPED=0

for name in "${DATABASES[@]}"; do
    db_path="$DATA_DIR/$name.db"

    if [[ ! -f "$db_path" ]]; then
        echo "[backup] $(date -u +%Y-%m-%dT%H:%M:%SZ) skip $name — $db_path does not exist"
        SKIPPED=$((SKIPPED + 1))
        continue
    fi

    backup_file="$BACKUP_DIR/$name-$TIMESTAMP.db"
    echo "[backup] $(date -u +%Y-%m-%dT%H:%M:%SZ) $name ← $db_path"
    sqlite3 "$db_path" "VACUUM INTO '$backup_file'"

    if [[ "$(sqlite3 "$backup_file" 'PRAGMA integrity_check;')" != "ok" ]]; then
        echo "[backup] ERROR: integrity check failed for $backup_file"
        exit 1
    fi

    find "$BACKUP_DIR" -name "$name-*.db" -mtime "+${RETENTION_DAYS}" -delete
    DONE=$((DONE + 1))
done

echo "[backup] $(date -u +%Y-%m-%dT%H:%M:%SZ) done: $DONE backed up, $SKIPPED skipped → $BACKUP_DIR (retention ${RETENTION_DAYS}d)"
