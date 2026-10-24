package birthprofile

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

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
    updated_at TEXT NOT NULL,
    calendar TEXT NOT NULL DEFAULT '',
    birth_date TEXT NOT NULL DEFAULT '',
    birth_time TEXT NOT NULL DEFAULT '',
    birth_time_known TEXT NOT NULL DEFAULT ''
);
INSERT INTO birth_profiles_exact
(id,user_id,version,nickname,name,gender,birth_instant,birth_timezone,birth_longitude,birth_place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at,calendar,birth_date,birth_time,birth_time_known)
SELECT id,user_id,version,nickname,name,gender,
       CASE WHEN calendar='solar' AND birth_time_known='known' THEN birth_utc END AS birth_instant,
       CASE WHEN calendar='solar' AND birth_time_known='known' THEN birth_timezone END,
       CASE WHEN calendar='solar' AND birth_time_known='known' THEN birth_longitude END,
       birth_place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at,
       calendar,birth_date,birth_time,birth_time_known
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
		{
			// 007/008 重建时保留了旧的公历/农历字段；这里先回填可确定的公历时间，
			// 把原始输入归档到 birth_profiles_legacy，然后才删掉旧列。
			// 已执行过旧版 007 的库没有这些列，直接跳过。
			ID: "009_birth_profile_backfill",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				var legacy int
				if err := tx.QueryRowContext(ctx,
					`SELECT COUNT(*) FROM pragma_table_info('birth_profiles') WHERE name='birth_date'`,
				).Scan(&legacy); err != nil {
					return err
				}
				if legacy == 0 {
					return nil
				}
				if _, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS birth_profiles_legacy (
    id TEXT PRIMARY KEY,
    calendar TEXT NOT NULL DEFAULT '',
    birth_date TEXT NOT NULL DEFAULT '',
    birth_time TEXT NOT NULL DEFAULT '',
    birth_time_known TEXT NOT NULL DEFAULT ''
);
INSERT OR REPLACE INTO birth_profiles_legacy (id,calendar,birth_date,birth_time,birth_time_known)
SELECT id,calendar,birth_date,birth_time,birth_time_known FROM birth_profiles;
`); err != nil {
					return err
				}

				type legacyRow struct{ id, date, clock string }
				var pending []legacyRow
				rows, err := tx.QueryContext(ctx,
					`SELECT id,birth_date,birth_time FROM birth_profiles
					 WHERE calendar='solar' AND birth_date<>'' AND birth_time<>''`)
				if err != nil {
					return err
				}
				for rows.Next() {
					var r legacyRow
					if err := rows.Scan(&r.id, &r.date, &r.clock); err != nil {
						_ = rows.Close()
						return err
					}
					pending = append(pending, r)
				}
				if err := rows.Err(); err != nil {
					_ = rows.Close()
					return err
				}
				_ = rows.Close()

				location, err := time.LoadLocation("Asia/Shanghai")
				if err != nil {
					return err
				}
				for _, r := range pending {
					instant, ok := parseLegacyCivil(r.date, r.clock, location)
					if !ok {
						continue
					}
					if _, err := tx.ExecContext(ctx,
						`UPDATE birth_profiles SET instant=?, timezone=? WHERE id=? AND instant IS NULL`,
						instant.UTC().Format("2006-01-02T15:04:05.000000000Z07:00"),
						"Asia/Shanghai", r.id,
					); err != nil {
						return err
					}
				}

				_, err = tx.ExecContext(ctx, `
CREATE TABLE birth_profiles_canonical (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    nickname TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    gender TEXT NOT NULL,
    instant TEXT,
    timezone TEXT,
    longitude REAL,
    place TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    third_party_consent INTEGER NOT NULL DEFAULT 0,
    last_used_at TEXT,
    deleted_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
INSERT INTO birth_profiles_canonical
(id,user_id,version,nickname,name,gender,instant,timezone,longitude,place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at)
SELECT id,user_id,version,nickname,name,gender,instant,timezone,longitude,place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at
FROM birth_profiles;
DROP TABLE birth_profiles;
ALTER TABLE birth_profiles_canonical RENAME TO birth_profiles;
CREATE INDEX idx_birth_profiles_user_updated
    ON birth_profiles(user_id, deleted_at, updated_at DESC);
CREATE INDEX idx_birth_profiles_user_created
    ON birth_profiles(user_id, created_at DESC);
`)
				return err
			},
		},
		{
			// The raw pre-canonical archive was only needed while 009 backfilled
			// civil birth times. Canonical rows are authoritative; no compatibility
			// reader exists.
			ID: "010_drop_birth_profile_legacy_archive",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS birth_profiles_legacy;`)
				return err
			},
		},
	})
}

// parseLegacyCivil parses the pre-canonical birth_date/birth_time pair as a
// civil time in loc. It intentionally accepts unpadded month/day/hour values.
func parseLegacyCivil(date, clock string, loc *time.Location) (time.Time, bool) {
	dateParts := strings.Split(strings.TrimSpace(date), "-")
	clockParts := strings.Split(strings.TrimSpace(clock), ":")
	if len(dateParts) != 3 || len(clockParts) < 2 {
		return time.Time{}, false
	}
	numbers := make([]int, 5)
	raw := []string{dateParts[0], dateParts[1], dateParts[2], clockParts[0], clockParts[1]}
	for index, value := range raw {
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return time.Time{}, false
		}
		numbers[index] = parsed
	}
	year, month, day, hour, minute := numbers[0], numbers[1], numbers[2], numbers[3], numbers[4]
	if year < 1900 || year > 2100 || month < 1 || month > 12 || day < 1 || day > 31 ||
		hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return time.Time{}, false
	}
	civil := time.Date(year, time.Month(month), day, hour, minute, 0, 0, loc)
	if civil.Day() != day || int(civil.Month()) != month || civil.Year() != year {
		return time.Time{}, false
	}
	return civil, true
}
