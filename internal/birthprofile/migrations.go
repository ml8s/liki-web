package birthprofile

import (
	"context"
	"database/sql"

	"github.com/ml8s/liki-web/internal/migrate"
)

func (s *Store) Migrate(ctx context.Context) error {
	return migrate.Run(ctx, s.db, []migrate.Migration{
		{
			ID: "001_profiles_platform",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS profiles (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    nickname TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    gender TEXT NOT NULL,
    relation TEXT NOT NULL DEFAULT '',
    calendar TEXT NOT NULL,
    birth_date TEXT NOT NULL,
    birth_time TEXT NOT NULL DEFAULT '',
    birth_time_known TEXT NOT NULL,
    birth_place TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    third_party_consent INTEGER NOT NULL DEFAULT 0,
    last_used_at TEXT,
    deleted_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_profiles_user_updated
    ON profiles(user_id, deleted_at, updated_at);
`)
				return err
			},
		},
		{
			ID: "002_profiles_drop_relation",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				// Check if column exists (idempotent)
				var cnt int
				err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('profiles') WHERE name='relation'`).Scan(&cnt)
				if err != nil || cnt == 0 {
					return nil // already dropped or table doesn't exist
				}
				// SQLite doesn't support DROP COLUMN before 3.35; recreate table
				_, err = tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS profiles_new (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    nickname TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    gender TEXT NOT NULL,
    calendar TEXT NOT NULL,
    birth_date TEXT NOT NULL,
    birth_time TEXT NOT NULL DEFAULT '',
    birth_time_known TEXT NOT NULL,
    birth_place TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    third_party_consent INTEGER NOT NULL DEFAULT 0,
    last_used_at TEXT,
    deleted_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
INSERT INTO profiles_new (id,user_id,version,nickname,name,gender,calendar,birth_date,birth_time,birth_time_known,birth_place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at)
SELECT id,user_id,version,nickname,name,gender,calendar,birth_date,birth_time,birth_time_known,birth_place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at FROM profiles;
DROP TABLE profiles;
ALTER TABLE profiles_new RENAME TO profiles;
CREATE INDEX IF NOT EXISTS idx_profiles_user_updated
    ON profiles(user_id, deleted_at, updated_at);
`)
				return err
			},
		},
		{
			ID: "003_profiles_cursor_index",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
CREATE INDEX IF NOT EXISTS idx_profiles_user_created
    ON profiles(user_id, deleted_at, created_at);
`)
				return err
			},
		},
		{
			ID: "004_canonical_birth_profile",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
ALTER TABLE profiles RENAME TO birth_profiles;
DROP INDEX IF EXISTS idx_profiles_user_updated;
DROP INDEX IF EXISTS idx_profiles_user_created;
CREATE INDEX idx_birth_profiles_user_updated
    ON birth_profiles(user_id, deleted_at, updated_at DESC);
CREATE INDEX idx_birth_profiles_user_created
    ON birth_profiles(user_id, created_at DESC);
`)
				return err
			},
		},
		{
			ID: "005_profiles_birth_longitude",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
ALTER TABLE birth_profiles ADD COLUMN birth_longitude REAL;
`)
				return err
			},
		},
		{
			ID: "006_profiles_canonical_time",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
ALTER TABLE birth_profiles ADD COLUMN birth_timezone TEXT;
ALTER TABLE birth_profiles ADD COLUMN birth_utc TEXT;
`)
				return err
			},
		},
		{
			ID: "007_profiles_exact_only",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
CREATE TABLE birth_profiles_exact (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    nickname TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    gender TEXT NOT NULL,
    birth_instant TEXT,
    birth_timezone TEXT,
    birth_longitude REAL,
    birth_place TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    third_party_consent INTEGER NOT NULL DEFAULT 0,
    last_used_at TEXT,
    deleted_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
INSERT INTO birth_profiles_exact
(id,user_id,version,nickname,name,gender,birth_instant,birth_timezone,birth_longitude,birth_place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at)
SELECT id,user_id,version,nickname,name,gender,
       CASE WHEN calendar='solar' AND birth_time_known='known' THEN birth_utc END AS birth_instant,
       CASE WHEN calendar='solar' AND birth_time_known='known' THEN birth_timezone END,
       CASE WHEN calendar='solar' AND birth_time_known='known' THEN birth_longitude END,
       birth_place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at
FROM birth_profiles;
DROP TABLE birth_profiles;
ALTER TABLE birth_profiles_exact RENAME TO birth_profiles;
CREATE INDEX idx_birth_profiles_user_updated
    ON birth_profiles(user_id, deleted_at, updated_at DESC);
CREATE INDEX idx_birth_profiles_user_created
    ON birth_profiles(user_id, created_at DESC);
`)
				return err
			},
		},
		{
			ID: "008_canonical_birth_columns",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
ALTER TABLE birth_profiles RENAME COLUMN birth_instant TO instant;
ALTER TABLE birth_profiles RENAME COLUMN birth_timezone TO timezone;
ALTER TABLE birth_profiles RENAME COLUMN birth_longitude TO longitude;
ALTER TABLE birth_profiles RENAME COLUMN birth_place TO place;
`)
				return err
			},
		},
	})
}
