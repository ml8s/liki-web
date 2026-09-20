#!/usr/bin/env python3
"""Split legacy Liki SQLite data into bounded-context databases.

The tool is intentionally staging-first: it never changes the source directory.
It backs up every participating source database, creates a fresh target
directory, validates row counts/content, and writes a migration manifest.

Recommended invocation:
  scripts/migrate-db.py \
    --legacy-main /var/lib/liki/liki.db \
    --target-dir /var/lib/liki-split \
    --confirm-stopped
"""

from __future__ import annotations

import argparse
import base64
import hashlib
import json
import os
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
import re
import shutil
import sqlite3
import sys

SCHEMA_MIGRATIONS_SQL = """
CREATE TABLE schema_migrations (
    scope TEXT NOT NULL,
    id TEXT NOT NULL,
    applied_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (scope, id)
)
"""

PLATFORM_TABLES = (
    "users", "auth_identities", "auth_codes", "auth_sessions",
    "auth_audit_logs", "payment_intents", "payment_notifications",
    "fulfillment_tasks", "payment_events",
)
PROFILES_TABLES = ("profiles",)
FEEDBACK_TABLES = ("feedback",)
AIPLAY_TABLES = ("aipay_orders",)
LIKI_TABLES = (
    "products", "entitlements", "chat_orders",
    "unified_chat_sessions", "unified_chat_messages",
)
NAMING_TABLES = ("orders", "chat_messages", "verification_codes")

TABLE_TARGET = {}
for table in PLATFORM_TABLES:
    TABLE_TARGET[table] = "platform.db"
for table in PROFILES_TABLES:
    TABLE_TARGET[table] = "profiles.db"
for table in FEEDBACK_TABLES:
    TABLE_TARGET[table] = "feedback.db"
for table in AIPLAY_TABLES:
    TABLE_TARGET[table] = "aipay.db"
for table in LIKI_TABLES:
    TABLE_TARGET[table] = "liki.db"
for table in NAMING_TABLES:
    TABLE_TARGET[table] = "naming.db"

FULL_COPY_DB_NAMES = ("mingshu.db", "hepan.db")
BASE_SIDE_CAR_DB_NAMES = ("naming.db", "liki.db", "feedback.db", "aipay.db")
SIDE_CAR_TABLES = {
    "naming.db": NAMING_TABLES,
    "liki.db": LIKI_TABLES,
    "feedback.db": FEEDBACK_TABLES,
    "aipay.db": AIPLAY_TABLES,
}


class MigrationError(Exception):
    pass


@dataclass
class SourceDB:
    path: Path
    conn: sqlite3.Connection


def qident(name: str) -> str:
    return '"' + name.replace('"', '""') + '"'


def qliteral(path: Path) -> str:
    return "'" + str(path).replace("'", "''") + "'"


def open_source(path: Path) -> sqlite3.Connection:
    # Read-only connections make accidental writes to the legacy directory
    # impossible. SQLite's backup API works normally from a read-only source.
    uri = path.resolve().as_uri() + "?mode=ro"
    conn = sqlite3.connect(uri, uri=True)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA query_only=ON")
    conn.execute("PRAGMA foreign_keys=OFF")
    return conn


def user_tables(conn: sqlite3.Connection) -> list[str]:
    rows = conn.execute(
        "SELECT name FROM sqlite_master "
        "WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
    ).fetchall()
    return [row[0] for row in rows]


def table_exists(conn: sqlite3.Connection, table: str) -> bool:
    row = conn.execute(
        "SELECT 1 FROM sqlite_master WHERE type='table' AND name=?", (table,)
    ).fetchone()
    return row is not None


def columns(conn: sqlite3.Connection, table: str) -> list[str]:
    return [row[1] for row in conn.execute(f"PRAGMA table_info({qident(table)})")]


def primary_key(conn: sqlite3.Connection, table: str) -> list[str]:
    rows = conn.execute(f"PRAGMA table_info({qident(table)})").fetchall()
    return [
        row[1] for row in sorted((r for r in rows if r[5] > 0), key=lambda r: r[5])
    ]


def has_rowid(conn: sqlite3.Connection, table: str) -> bool:
    try:
        conn.execute(f"SELECT rowid FROM {qident(table)} LIMIT 1").fetchone()
        return True
    except sqlite3.Error:
        return False


def check_integrity(conn: sqlite3.Connection, label: str) -> None:
    results = [row[0] for row in conn.execute("PRAGMA integrity_check")]
    if results != ["ok"]:
        raise MigrationError(f"integrity check failed for {label}: {results}")


def backup_database(source: sqlite3.Connection, destination: Path) -> None:
    destination.parent.mkdir(parents=True, exist_ok=True)
    target = sqlite3.connect(destination)
    try:
        source.backup(target)
        target.commit()
    finally:
        target.close()


def fresh_target(path: Path) -> sqlite3.Connection:
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists():
        raise MigrationError(f"target database already exists: {path}")
    conn = sqlite3.connect(path)
    conn.execute("PRAGMA foreign_keys=OFF")
    reset_schema_migrations(conn)
    return conn


def reset_schema_migrations(conn: sqlite3.Connection) -> None:
    # The target runs the current application migrations after the split.
    # Do not import a legacy migration history that might suppress them.
    conn.execute("DROP TABLE IF EXISTS schema_migrations")
    conn.execute(SCHEMA_MIGRATIONS_SQL)
    conn.commit()


def strip_cross_database_fk(sql: str) -> str:
    action = r"(?:\s+ON\s+(?:DELETE|UPDATE)\s+(?:CASCADE|RESTRICT|NO\s+ACTION|SET\s+NULL|SET\s+DEFAULT))*"
    pattern = re.compile(
        r"\s+REFERENCES\s+(?:users|products)\s*\(\s*id\s*\)" + action,
        flags=re.IGNORECASE,
    )
    return pattern.sub("", sql)


def create_source_schema(
    target: sqlite3.Connection,
    source: sqlite3.Connection,
    table: str,
    strip_fk: bool = False,
) -> None:
    if table_exists(target, table):
        return
    row = source.execute(
        "SELECT sql FROM sqlite_master WHERE type='table' AND name=?", (table,)
    ).fetchone()
    if row is None or not row[0]:
        raise MigrationError(f"cannot read CREATE SQL for {table}")
    sql = row[0]
    if strip_fk:
        sql = strip_cross_database_fk(sql)
    target.execute(sql)


def copy_indexes_triggers(
    target: sqlite3.Connection,
    source: sqlite3.Connection,
    table: str,
) -> None:
    rows = source.execute(
        "SELECT type,name,sql FROM sqlite_master "
        "WHERE tbl_name=? AND sql IS NOT NULL AND type IN ('index','trigger')",
        (table,),
    ).fetchall()
    for row in rows:
        target.execute(row[2])


def assert_schema_compatible(
    target: sqlite3.Connection,
    source: sqlite3.Connection,
    table: str,
) -> None:
    target_cols = columns(target, table)
    source_cols = columns(source, table)
    if target_cols != source_cols:
        raise MigrationError(
            f"schema mismatch for {table}: target={target_cols!r}, source={source_cols!r}"
        )


def attach_source(target: sqlite3.Connection, source_path: Path) -> None:
    target.execute("ATTACH DATABASE ? AS migration_source", (str(source_path),))


def detach_source(target: sqlite3.Connection) -> None:
    target.execute("DETACH DATABASE migration_source")


def assert_no_merge_conflicts(
    target: sqlite3.Connection,
    source: sqlite3.Connection,
    table: str,
) -> None:
    cols = columns(source, table)
    keys = primary_key(source, table) or cols
    target_cols = set(columns(target, table))
    if not set(cols).issubset(target_cols):
        raise MigrationError(f"source table {table} has columns missing in target")

    join = " AND ".join(f"t.{qident(k)} IS s.{qident(k)}" for k in keys)
    where = " OR ".join(f"t.{qident(c)} IS NOT s.{qident(c)}" for c in cols if c not in keys)
    sql = (
        f"SELECT count(*) FROM main.{qident(table)} AS t "
        f"JOIN migration_source.{qident(table)} AS s ON {join}"
    )
    if where:
        sql += f" WHERE {where}"
    conflicts = target.execute(sql).fetchone()[0]
    if conflicts:
        raise MigrationError(
            f"{table} has {conflicts} rows with the same primary key but different data; "
            "resolve the conflicting source databases before migration"
        )


def merge_table(
    target: sqlite3.Connection,
    source: SourceDB,
    table: str,
) -> int:
    cols = columns(source.conn, table)
    col_list = ",".join(qident(c) for c in cols)
    attach_source(target, source.path)
    try:
        assert_schema_compatible(target, source.conn, table)
        assert_no_merge_conflicts(target, source.conn, table)
        target.execute(
            f"INSERT OR IGNORE INTO main.{qident(table)} ({col_list}) "
            f"SELECT {col_list} FROM migration_source.{qident(table)}"
        )
        target.commit()
    finally:
        detach_source(target)
    return source.conn.execute(f"SELECT count(*) FROM {qident(table)}").fetchone()[0]


def copy_table(
    target: sqlite3.Connection,
    source: SourceDB,
    table: str,
) -> None:
    create_source_schema(target, source.conn, table, strip_fk=(table == "entitlements"))
    attach_source(target, source.path)
    try:
        target.execute(
            f"INSERT INTO main.{qident(table)} "
            f"SELECT * FROM migration_source.{qident(table)}"
        )
        target.commit()
    finally:
        detach_source(target)
    copy_indexes_triggers(target, source.conn, table)


def sync_sequences(
    target: sqlite3.Connection,
    source: sqlite3.Connection,
    tables: list[str],
) -> None:
    if not table_exists(source, "sqlite_sequence"):
        return
    if not table_exists(target, "sqlite_sequence"):
        return
    for table in tables:
        if table not in tables or not table_exists(target, table):
            continue
        row = source.execute(
            "SELECT seq FROM sqlite_sequence WHERE name=?", (table,)
        ).fetchone()
        if row is None:
            continue
        source_seq = int(row[0])
        target_row = target.execute(
            "SELECT seq FROM sqlite_sequence WHERE name=?", (table,)
        ).fetchone()
        target_seq = int(target_row[0]) if target_row else 0
        final_seq = max(source_seq, target_seq)
        target.execute(
            "INSERT OR REPLACE INTO sqlite_sequence(name,seq) VALUES(?,?)",
            (table, final_seq),
        )


def json_value(value):
    if isinstance(value, bytes):
        return {"$base64": base64.b64encode(value).decode("ascii")}
    if isinstance(value, str):
        return value
    return value


def table_inventory(conn: sqlite3.Connection, label: str) -> dict:
    result = {}
    for table in user_tables(conn):
        count = conn.execute(f"SELECT count(*) FROM {qident(table)}").fetchone()[0]
        keys = primary_key(conn, table)
        if keys:
            order = " ORDER BY " + ",".join(qident(k) for k in keys)
        elif has_rowid(conn, table):
            order = " ORDER BY rowid"
        else:
            order = ""
        digest = hashlib.sha256()
        cursor = conn.execute(f"SELECT * FROM {qident(table)}{order}")
        names = [d[0] for d in cursor.description]
        for row in cursor:
            record = json.dumps(
                dict(zip(names, (json_value(v) for v in row))),
                ensure_ascii=False,
                sort_keys=True,
                separators=(",", ":"),
                default=json_value,
            )
            digest.update(record.encode("utf-8"))
            digest.update(b"\n")
        result[table] = {"count": count, "sha256": digest.hexdigest()}
    return {label: result}


def ensure_no_liki_process() -> None:
    for entry in Path("/proc").iterdir():
        if not entry.name.isdigit():
            continue
        try:
            exe = os.readlink(entry / "exe")
        except OSError:
            continue
        if Path(exe).name == "liki-web":
            raise MigrationError(f"liki-web is still running (pid {entry.name})")


def assert_not_nested(a: Path, b: Path, message: str) -> None:
    try:
        a.relative_to(b)
    except ValueError:
        return
    raise MigrationError(message)


def create_table_source_schema_if_needed(target, source, table):
    if not table_exists(target, table):
        create_source_schema(target, source, table, strip_fk=(table == "entitlements"))
        copy_indexes_triggers(target, source, table)


def validate_main_schema(main: SourceDB) -> None:
    if "users" not in user_tables(main.conn):
        raise MigrationError(
            f"{main.path} does not look like the legacy main database: users table is missing"
        )


def find_unknown_main_tables(main: SourceDB) -> list[str]:
    known = set(TABLE_TARGET)
    return [
        table for table in user_tables(main.conn)
        if table not in known and table != "schema_migrations"
    ]


def validate_paths(args) -> tuple[Path, Path, Path, Path]:
    main_path = args.legacy_main.resolve()
    if not main_path.is_file():
        raise MigrationError(f"legacy main database not found: {main_path}")
    source_dir = (
        args.source_dir.resolve() if args.source_dir else main_path.parent.resolve()
    )
    target_dir = args.target_dir.resolve()
    if target_dir.exists():
        raise MigrationError(f"target directory must not exist: {target_dir}")
    assert_not_nested(target_dir, source_dir, "target directory must not be inside source directory")
    assert_not_nested(source_dir, target_dir, "source directory must not be inside target directory")
    if args.backup_dir:
        backup_dir = args.backup_dir.resolve()
        if backup_dir.exists() and any(backup_dir.iterdir()):
            raise MigrationError(f"backup directory is not empty: {backup_dir}")
        assert_not_nested(backup_dir, target_dir, "backup directory must not be inside target directory")
        assert_not_nested(target_dir, backup_dir, "target directory must not be inside backup directory")
    else:
        stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
        backup_dir = target_dir.parent / f".{target_dir.name}-backup-{stamp}"
        if backup_dir.exists():
            raise MigrationError(f"backup directory already exists: {backup_dir}")
    return main_path, source_dir, target_dir, backup_dir


def validate_base_side_car(source: SourceDB, table_names: tuple[str, ...]) -> None:
    expected = set(table_names) | {"schema_migrations"}
    unknown = [table for table in user_tables(source.conn) if table not in expected]
    if unknown:
        raise MigrationError(
            f"unexpected tables in {source.path}: {unknown}; refuse to merge an ambiguous partial split"
        )


def backup_source(path: Path, backup_dir: Path, stamp: str) -> Path:
    digest = hashlib.sha256(str(path).encode()).hexdigest()[:8]
    destination = backup_dir / f"{path.stem}-{digest}-{stamp}.db"
    conn = open_source(path)
    try:
        check_integrity(conn, str(path))
        backup_database(conn, destination)
    finally:
        conn.close()
    return destination


def validate_fk_targets(target_conns: dict[str, sqlite3.Connection], platform_path: Path) -> None:
    checks = [
        ("profiles.db", "profiles", "user_id"),
        ("liki.db", "entitlements", "user_id"),
        ("liki.db", "chat_orders", "user_id"),
        ("platform.db", "payment_intents", "user_id"),
    ]
    for dbname, table, column in checks:
        conn = target_conns.get(dbname)
        if conn is None or not table_exists(conn, table):
            continue
        conn.execute(f"ATTACH DATABASE {qliteral(platform_path)} AS migration_platform")
        try:
            missing = conn.execute(
                f"SELECT count(*) FROM main.{qident(table)} AS t "
                f"LEFT JOIN migration_platform.users AS u ON u.id=t.{qident(column)} "
                f"WHERE t.{qident(column)} <> '' AND u.id IS NULL"
            ).fetchone()[0]
        finally:
            conn.execute("DETACH DATABASE migration_platform")
        if missing:
            raise MigrationError(
                f"{dbname}.{table}.{column} has {missing} values missing from users"
            )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--legacy-main", type=Path, required=True)
    parser.add_argument("--source-dir", type=Path)
    parser.add_argument("--target-dir", type=Path, required=True)
    parser.add_argument("--backup-dir", type=Path)
    parser.add_argument("--confirm-stopped", action="store_true")
    args = parser.parse_args()

    if not args.confirm_stopped:
        parser.error("--confirm-stopped is required after stopping liki-web")

    target_dir: Path | None = None
    target_conns: dict[str, sqlite3.Connection] = {}
    try:
        ensure_no_liki_process()
        main_path, source_dir, target_dir, backup_dir = validate_paths(args)
        target_dir.mkdir(parents=True)
        backup_dir.mkdir(parents=True)

        stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
        main = SourceDB(main_path, open_source(main_path))
        validate_main_schema(main)

        unknown = find_unknown_main_tables(main)
        if unknown:
            raise MigrationError(
                f"legacy main has tables not covered by the migration mapping: {unknown}"
            )

        partial_split_names = ("platform.db", "profiles.db", "referral.db")
        for name in partial_split_names:
            candidate = source_dir / name
            if candidate.exists() and candidate.resolve() != main_path:
                raise MigrationError(
                    f"partial split database found: {candidate}; migrate from the original legacy data directory instead"
                )

        all_sources = [main]
        sidecars: dict[str, SourceDB] = {}
        for name in ("mingshu.db", "hepan.db", *BASE_SIDE_CAR_DB_NAMES):
            path = source_dir / name
            if path.exists() and path.resolve() != main_path:
                db = SourceDB(path, open_source(path))
                if name in SIDE_CAR_TABLES:
                    validate_base_side_car(db, SIDE_CAR_TABLES[name])
                sidecars[name] = db
                all_sources.append(db)

        backups: dict[str, str] = {}
        for source in all_sources:
            destination = backup_source(source.path, backup_dir, stamp)
            backups[str(source.path)] = str(destination)

        def conn_for(name: str) -> sqlite3.Connection:
            if name not in target_conns:
                target_conns[name] = fresh_target(target_dir / name)
            return target_conns[name]

        # Full-copy databases preserve every table, index, trigger, and sequence.
        for name in FULL_COPY_DB_NAMES:
            source = sidecars.get(name)
            if source is None:
                continue
            target_path = target_dir / name
            backup_database(source.conn, target_path)
            conn = sqlite3.connect(target_path)
            conn.execute("PRAGMA foreign_keys=OFF")
            reset_schema_migrations(conn)
            target_conns[name] = conn

        # Optional same-schema databases become the base; overlapping main rows
        # are merged only when there are no primary-key conflicts.
        for name, table_names in SIDE_CAR_TABLES.items():
            source = sidecars.get(name)
            if source is None:
                continue
            target_path = target_dir / name
            if any(user_tables(source.conn)):
                backup_database(source.conn, target_path)
                conn = sqlite3.connect(target_path)
                conn.execute("PRAGMA foreign_keys=OFF")
                reset_schema_migrations(conn)
                target_conns[name] = conn
            for table in table_names:
                if table_exists(source.conn, table):
                    conn = conn_for(name)
                    create_table_source_schema_if_needed(conn, source.conn, table)

        # Split the legacy main database.
        for table, dbname in TABLE_TARGET.items():
            if not table_exists(main.conn, table):
                continue
            target = conn_for(dbname)
            if table_exists(target, table):
                merge_table(target, main, table)
            else:
                copy_table(target, main, table)
            sync_sequences(target, main.conn, [table])

        # Merge rows from optional sidecars that were not chosen as the base.
        for name, table_names in SIDE_CAR_TABLES.items():
            source = sidecars.get(name)
            if source is None:
                continue
            target_path = target_dir / name
            if not target_path.exists():
                continue
            target = conn_for(name)
            for table in table_names:
                if table_exists(target, table) and table_exists(source.conn, table):
                    merge_table(target, source, table)
                    sync_sequences(target, source.conn, [table])
                elif table_exists(source.conn, table):
                    create_table_source_schema_if_needed(target, source.conn, table)
                    copy_table(target, source, table)

        # Check entitlements created by the legacy split script do not retain
        # cross-database foreign keys.
        if "liki.db" in target_conns:
            sql_row = target_conns["liki.db"].execute(
                "SELECT sql FROM sqlite_master WHERE type='table' AND name='entitlements'"
            ).fetchone()
            if sql_row and re.search(r"\bREFERENCES\s+users\s*\(", sql_row[0], re.I):
                raise MigrationError("target entitlements still references users; refusing to emit unsafe split")

        platform_path = target_dir / "platform.db"
        if not platform_path.exists():
            conn_for("platform.db")
        validate_fk_targets(target_conns, platform_path)

        source_inventory: dict[str, dict] = {}
        for source in all_sources:
            source_inventory.update(table_inventory(source.conn, str(source.path)))
        target_inventory: dict[str, dict] = {}
        for name, conn in sorted(target_conns.items()):
            target_inventory.update(table_inventory(conn, name))

        for name, conn in target_conns.items():
            check_integrity(conn, name)
            violations = conn.execute("PRAGMA foreign_key_check").fetchall()
            if violations:
                raise MigrationError(f"foreign key check failed for {name}: {violations}")

        manifest = {
            "version": 1,
            "status": "success",
            "started_at": stamp,
            "source_dir": str(source_dir),
            "legacy_main": str(main_path),
            "target_dir": str(target_dir),
            "backups": backups,
            "source_inventory": source_inventory,
            "target_inventory": target_inventory,
        }
        manifest_path = backup_dir / "migration-manifest.json"
        manifest_path.write_text(
            json.dumps(manifest, ensure_ascii=False, indent=2, sort_keys=True) + "\n"
        )

        for conn in target_conns.values():
            conn.commit()
            conn.close()
        for source in all_sources:
            source.conn.close()

        print("✅ staging split complete")
        print(f"target:   {target_dir}")
        print(f"backups:  {backup_dir}")
        print(f"manifest: {manifest_path}")
        print("next: run liki-web with LIKI_DATA_DIR set to target and smoke-test old data")
        return 0
    except (MigrationError, sqlite3.Error, OSError) as exc:
        for conn in target_conns.values():
            _ = conn.close()
        # A failed migration must never leave a directory that can be mistaken
        # for a valid staging target. Backups and any diagnostic files outside
        # the target are intentionally preserved.
        if target_dir is not None and target_dir.exists():
            shutil.rmtree(target_dir)
        print(f"❌ {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
