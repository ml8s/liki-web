import importlib.util
import json
import sqlite3
import sys
import tempfile
import unittest
import contextlib
import io
from pathlib import Path


SCRIPT = Path(__file__).resolve().parents[1] / "migrate-db.py"
SPEC = importlib.util.spec_from_file_location("migrate_db", SCRIPT)
migrate_db = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = migrate_db
SPEC.loader.exec_module(migrate_db)


def execute(path: Path, statements: str) -> None:
    conn = sqlite3.connect(path)
    conn.executescript(statements)
    conn.commit()
    conn.close()


def make_fixture(root: Path) -> tuple[Path, Path]:
    source = root / "legacy"
    source.mkdir()
    main = source / "liki.db"
    execute(main, """
    CREATE TABLE schema_migrations (id TEXT PRIMARY KEY, applied_at TEXT);
    CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT NOT NULL);
    CREATE TABLE auth_sessions (id TEXT PRIMARY KEY, user_id TEXT NOT NULL);
    CREATE TABLE profiles (id TEXT PRIMARY KEY, user_id TEXT NOT NULL);
    CREATE TABLE products (id TEXT PRIMARY KEY);
    CREATE TABLE entitlements (
      id TEXT PRIMARY KEY,
      user_id TEXT NOT NULL REFERENCES users(id),
      product_id TEXT NOT NULL REFERENCES products(id)
    );
    CREATE TABLE chat_orders (id TEXT PRIMARY KEY, user_id TEXT NOT NULL);
    CREATE TABLE unified_chat_sessions (id TEXT PRIMARY KEY);
    CREATE TABLE unified_chat_messages (id TEXT PRIMARY KEY);
    CREATE TABLE feedback (id TEXT PRIMARY KEY, payload TEXT NOT NULL);
    CREATE TABLE aipay_orders (out_trade_no TEXT PRIMARY KEY, amount TEXT NOT NULL);
    CREATE TABLE orders (order_id TEXT PRIMARY KEY, email TEXT NOT NULL);
    CREATE TABLE chat_messages (id INTEGER PRIMARY KEY AUTOINCREMENT, body TEXT NOT NULL);
    CREATE TABLE verification_codes (code TEXT PRIMARY KEY);
    INSERT INTO users VALUES ('u1', 'u1@example.com');
    INSERT INTO auth_sessions VALUES ('s1', 'u1');
    INSERT INTO profiles VALUES ('p1', 'u1');
    INSERT INTO products VALUES ('liki-chat');
    INSERT INTO entitlements VALUES ('e1', 'u1', 'liki-chat');
    INSERT INTO chat_orders VALUES ('co1', 'u1');
    INSERT INTO feedback VALUES ('f-main', '{}');
    INSERT INTO aipay_orders VALUES ('a-main', '100');
    INSERT INTO orders VALUES ('o-main', 'u1@example.com');
    INSERT INTO chat_messages VALUES (1, 'one'), (10, 'ten');
    INSERT INTO verification_codes VALUES ('v1');
    """)

    execute(source / "naming.db", """
    CREATE TABLE schema_migrations (id TEXT PRIMARY KEY);
    CREATE TABLE orders (order_id TEXT PRIMARY KEY, email TEXT NOT NULL);
    CREATE TABLE chat_messages (id INTEGER PRIMARY KEY AUTOINCREMENT, body TEXT NOT NULL);
    CREATE TABLE verification_codes (code TEXT PRIMARY KEY);
    INSERT INTO orders VALUES ('o-side', 'side@example.com');
    INSERT INTO chat_messages VALUES (12, 'twelve');
    """)

    execute(source / "feedback.db", """
    CREATE TABLE feedback (id TEXT PRIMARY KEY, payload TEXT NOT NULL);
    INSERT INTO feedback VALUES ('f-main', '{}'), ('f-side', '{}');
    """)

    execute(source / "aipay.db", """
    CREATE TABLE aipay_orders (out_trade_no TEXT PRIMARY KEY, amount TEXT NOT NULL);
    INSERT INTO aipay_orders VALUES ('a-side', '200');
    """)

    execute(source / "mingshu.db", """
    CREATE TABLE reports (id TEXT PRIMARY KEY, body TEXT NOT NULL);
    INSERT INTO reports VALUES ('m1', 'mingshu');
    """)
    execute(source / "hepan.db", """
    CREATE TABLE reports (id TEXT PRIMARY KEY, body TEXT NOT NULL);
    INSERT INTO reports VALUES ('h1', 'hepan');
    """)
    return main, source


def invoke(module, args):
    old_argv = sys.argv
    original_process_check = module.ensure_no_liki_process
    # Unit tests must not depend on unrelated liki-web processes on the host.
    module.ensure_no_liki_process = lambda: None
    sys.argv = ["migrate-db.py", *map(str, args)]
    out, err = io.StringIO(), io.StringIO()
    try:
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            return module.main()
    finally:
        sys.argv = old_argv
        module.ensure_no_liki_process = original_process_check


class MigrateDBTest(unittest.TestCase):
    def test_splits_main_and_merges_side_cars(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            main, source = make_fixture(root)
            target = root / "staging"
            backup = root / "backups"
            code = invoke(migrate_db, [
                "--legacy-main", main,
                "--source-dir", source,
                "--target-dir", target,
                "--backup-dir", backup,
                "--confirm-stopped",
            ])
            self.assertEqual(code, 0)

            # Required bounded-context databases exist and old migration history
            # is intentionally reset so the current application migrations run.
            for name in ("platform.db", "profiles.db", "liki.db", "naming.db", "aipay.db", "feedback.db", "mingshu.db", "hepan.db"):
                self.assertTrue((target / name).is_file(), name)
                with sqlite3.connect(target / name) as conn:
                    columns = conn.execute("PRAGMA table_info(schema_migrations)").fetchall()
                    self.assertEqual([row[1] for row in columns], ["scope", "id", "applied_at"])
                    self.assertEqual(conn.execute("SELECT count(*) FROM schema_migrations").fetchone()[0], 0)

            checks = {
                (target / "platform.db"): [
                    ("users", 1), ("auth_sessions", 1),
                ],
                (target / "profiles.db"): [("profiles", 1)],
                (target / "liki.db"): [
                    ("products", 1), ("entitlements", 1), ("chat_orders", 1),
                ],
                (target / "naming.db"): [("orders", 2), ("chat_messages", 3), ("verification_codes", 1)],
                (target / "aipay.db"): [("aipay_orders", 2)],
                (target / "feedback.db"): [("feedback", 2)],
                (target / "mingshu.db"): [("reports", 1)],
                (target / "hepan.db"): [("reports", 1)],
            }
            for path, tables in checks.items():
                with sqlite3.connect(path) as conn:
                    for table, count in tables:
                        actual = conn.execute(f'SELECT count(*) FROM "{table}"').fetchone()[0]
                        self.assertEqual(actual, count, f"{path.name}.{table}")

            with sqlite3.connect(target / "naming.db") as conn:
                self.assertEqual(conn.execute("SELECT seq FROM sqlite_sequence WHERE name='chat_messages'").fetchone()[0], 12)
            with sqlite3.connect(target / "liki.db") as conn:
                sql = conn.execute("SELECT sql FROM sqlite_master WHERE name='entitlements'").fetchone()[0]
                self.assertNotIn("REFERENCES users", sql.upper())
                self.assertNotIn("REFERENCES products", sql.upper())

            manifest = json.loads((backup / "migration-manifest.json").read_text())
            self.assertEqual(manifest["status"], "success")
            self.assertEqual(len(manifest["backups"]), 6)
            self.assertIn("platform.db", manifest["target_inventory"])
            self.assertEqual(manifest["target_inventory"]["platform.db"]["users"]["count"], 1)

    def test_rejects_conflicting_rows_and_removes_partial_target(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            main, source = make_fixture(root)
            with sqlite3.connect(source / "feedback.db") as conn:
                conn.execute("UPDATE feedback SET payload='{\"changed\":true}' WHERE id='f-main'")
            target = root / "staging"
            backup = root / "backups"
            code = invoke(migrate_db, [
                "--legacy-main", main,
                "--source-dir", source,
                "--target-dir", target,
                "--backup-dir", backup,
                "--confirm-stopped",
            ])
            self.assertEqual(code, 1)
            self.assertFalse(target.exists())
            self.assertTrue(backup.is_dir())

    def test_requires_stopped_confirmation(self):
        with self.assertRaises(SystemExit):
            invoke(migrate_db, ["--legacy-main", "/tmp/unused.db", "--target-dir", "/tmp/unused-target"])


if __name__ == "__main__":
    unittest.main()
