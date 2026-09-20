#!/usr/bin/env bash

# Split the legacy monolithic naming database into bounded-context databases.
#
# Safety model:
#   1. The caller must explicitly confirm that liki-web is stopped.
#   2. The source database is never modified.
#   3. Target database files must not already exist, so every run starts from
#      a clean staging directory and can never merge partial data.
#   4. The source is copied with sqlite3 VACUUM INTO before it is read.
#   5. Every table's row count and ordered content digest are verified.
#
# Usage:
#   LIKI_MIGRATION_CONFIRM_STOPPED=1 \
#     scripts/migrate-db.sh /var/lib/liki/naming.db /var/lib/liki-split
#
# After a successful run, verify the application against TARGET and only then
# swap the staged directory into production manually. The script deliberately
# does not perform that production swap.

set -euo pipefail

SOURCE="${1:-}"
TARGET="${2:-}"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"

usage() {
  echo "Usage: LIKI_MIGRATION_CONFIRM_STOPPED=1 $0 <legacy-naming.db> <empty-target-dir>" >&2
  exit 2
}

[[ -n "$SOURCE" && -n "$TARGET" ]] || usage
[[ "${LIKI_MIGRATION_CONFIRM_STOPPED:-0}" == "1" ]] || {
  echo "❌ Refusing to run: set LIKI_MIGRATION_CONFIRM_STOPPED=1 after stopping liki-web" >&2
  exit 1
}

for path in "$SOURCE" "$TARGET"; do
  case "$path" in *"'"*) echo "❌ Paths must not contain single quotes: $path" >&2; exit 1 ;; esac
done

[[ -f "$SOURCE" ]] || { echo "❌ Source database not found: $SOURCE" >&2; exit 1; }
[[ ! -e "$TARGET" ]] || { echo "❌ Target must not exist: $TARGET" >&2; exit 1; }

if command -v pgrep >/dev/null 2>&1 && pgrep -x liki-web >/dev/null 2>&1; then
  echo "❌ liki-web is still running. Stop it and run again." >&2
  exit 1
fi

command -v sqlite3 >/dev/null 2>&1 || { echo "❌ sqlite3 is required" >&2; exit 1; }
command -v sha256sum >/dev/null 2>&1 || { echo "❌ sha256sum is required" >&2; exit 1; }
command -v realpath >/dev/null 2>&1 || { echo "❌ realpath is required" >&2; exit 1; }

SOURCE="$(realpath "$SOURCE")"
mkdir -p "$TARGET"
TARGET="$(realpath "$TARGET")"
BACKUP="${SOURCE}.backup.${TIMESTAMP}.db"

echo "=== Liki database split ==="
echo "Source:       $SOURCE"
echo "Staging:      $TARGET"
echo "Cold backup:  $BACKUP"

check() {
  local result
  result="$(sqlite3 "$1" "$2")"
  if [[ "$result" != "ok" ]]; then
    echo "❌ SQLite integrity check failed for $1: $result" >&2
    exit 1
  fi
}

check "$SOURCE" "PRAGMA integrity_check"
sqlite3 "$SOURCE" "PRAGMA wal_checkpoint(TRUNCATE);" >/dev/null
echo "🧾 Cold copy: $SOURCE → $BACKUP"
sqlite3 "$SOURCE" "VACUUM INTO '$BACKUP';"
check "$BACKUP" "PRAGMA integrity_check"

initialize_schema_migrations() {
  local db="$1"
  sqlite3 "$db" <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
    scope TEXT NOT NULL,
    id TEXT NOT NULL,
    applied_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (scope, id)
);
SQL
}

# Database file -> whitespace-separated tables, in dependency-safe order.
declare -A MIGRATIONS=(
  ["platform.db"]="users auth_identities auth_codes auth_sessions auth_audit_logs payment_intents payment_notifications fulfillment_tasks payment_events"
  ["profiles.db"]="profiles"
  ["feedback.db"]="feedback"
  ["aipay.db"]="aipay_orders"
  ["liki.db"]="products entitlements chat_orders unified_chat_sessions unified_chat_messages"
  ["naming.db"]="orders chat_messages verification_codes"
)

for dbfile in "${!MIGRATIONS[@]}"; do
  if [[ "$SOURCE" == "$(realpath -m "$TARGET/$dbfile")" ]]; then
    echo "❌ Target database equals source: $TARGET/$dbfile" >&2
    exit 1
  fi
  if [[ -e "$TARGET/$dbfile" ]]; then
    echo "❌ Target database already exists: $TARGET/$dbfile" >&2
    exit 1
  fi
done

table_exists() {
  [[ "$(sqlite3 "$SOURCE" "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='$1';")" -ne 0 ]]
}

content_hash() {
  sqlite3 "$1" <<SQL | LC_ALL=C sort | sha256sum | cut -d' ' -f1
.mode quote
SELECT * FROM "$2";
SQL
}

copy_table() {
  local dbfile="$1" table="$2" create_sql source_count target_count source_hash target_hash
  local target_db="$TARGET/$dbfile"

  if ! table_exists "$table"; then
    echo "  ⚠️  $table: absent in source; application migrations will create it"
    return
  fi

  create_sql="$(sqlite3 "$SOURCE" "SELECT sql FROM sqlite_master WHERE type='table' AND name='$table';")"
  if [[ "$table" == "entitlements" ]]; then
    # Cross-database foreign keys are forbidden. The Go migration owns the
    # products catalog, so product/user references become plain TEXT IDs.
    create_sql="$(sed -e 's/ REFERENCES users(id)[^,)]*//g' \
                     -e 's/ REFERENCES products(id)[^,)]*//g' <<<"$create_sql")"
  fi

  printf '%s;\n' "$create_sql" | sqlite3 "$target_db" >/dev/null
  source_count="$(sqlite3 "$SOURCE" "SELECT count(*) FROM \"$table\";")"

  sqlite3 "$SOURCE" <<SQL
ATTACH DATABASE '$target_db' AS target;
INSERT INTO target."$table" SELECT * FROM main."$table";
DETACH DATABASE target;
SQL

  target_count="$(sqlite3 "$target_db" "SELECT count(*) FROM \"$table\";")"
  if [[ "$source_count" != "$target_count" ]]; then
    echo "❌ $dbfile.$table row count: source=$source_count target=$target_count" >&2
    exit 1
  fi

  source_hash="$(content_hash "$SOURCE" "$table")"
  target_hash="$(content_hash "$target_db" "$table")"
  if [[ "$source_hash" != "$target_hash" ]]; then
    echo "❌ $dbfile.$table content digest mismatch" >&2
    exit 1
  fi

  while IFS= read -r index_sql; do
    if [[ -n "$index_sql" ]]; then
      printf '%s;\n' "$index_sql" | sqlite3 "$target_db" >/dev/null
    fi
  done < <(sqlite3 "$SOURCE" "SELECT sql FROM sqlite_master WHERE type='index' AND tbl_name='$table' AND sql IS NOT NULL;")

  while IFS= read -r trigger_sql; do
    if [[ -n "$trigger_sql" ]]; then
      printf '%s;\n' "$trigger_sql" | sqlite3 "$target_db" >/dev/null
    fi
  done < <(sqlite3 "$SOURCE" "SELECT sql FROM sqlite_master WHERE type='trigger' AND tbl_name='$table' AND sql IS NOT NULL;")

  echo "  ✅ $table: $source_count rows, digest $source_hash"
}

for dbfile in platform.db profiles.db feedback.db aipay.db liki.db naming.db; do
  target_db="$TARGET/$dbfile"
  echo ""
  echo "📂 Creating $dbfile"
  sqlite3 "$target_db" "VACUUM;"
  initialize_schema_migrations "$target_db"

  for table in ${MIGRATIONS[$dbfile]}; do
    copy_table "$dbfile" "$table"
  done

  check "$target_db" "PRAGMA integrity_check"
  if ! foreign_errors="$(sqlite3 "$target_db" "PRAGMA foreign_key_check;")"; then
    echo "❌ $dbfile foreign key check could not run" >&2
    exit 1
  fi
  if [[ -n "$foreign_errors" ]]; then
    echo "❌ $dbfile foreign key check failed:" >&2
    echo "$foreign_errors" >&2
    exit 1
  fi
done

echo ""
echo "🧾 Cross-database soft references"
missing_entitlement_users="$(sqlite3 "$TARGET/liki.db" <<SQL
ATTACH DATABASE '$TARGET/platform.db' AS platform;
SELECT count(*) FROM main.entitlements AS e
LEFT JOIN platform.users AS u ON u.id = e.user_id
WHERE u.id IS NULL;
SQL
)"
missing_chat_order_users="$(sqlite3 "$TARGET/liki.db" <<SQL
ATTACH DATABASE '$TARGET/platform.db' AS platform;
SELECT count(*) FROM main.chat_orders AS o
LEFT JOIN platform.users AS u ON u.id = o.user_id
WHERE u.id IS NULL;
SQL
)"

if [[ "$missing_entitlement_users" != "0" ]]; then
  echo "❌ entitlements.user_id has $missing_entitlement_users values missing from users" >&2
  exit 1
fi
if [[ "$missing_chat_order_users" != "0" ]]; then
  echo "❌ chat_orders.user_id has $missing_chat_order_users values missing from users" >&2
  exit 1
fi
echo "✅ entitlement and chat order user references match platform users"

echo ""
echo "✅ Staging split complete."
echo "Next steps:"
echo "  1. Run liki-web with LIKI_DATA_DIR=$TARGET and smoke-test old users/orders."
echo "  2. Keep $BACKUP and the original $SOURCE until sign-off."
echo "  3. Only after sign-off, swap the staged directory into production manually."
