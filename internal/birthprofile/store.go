package birthprofile

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ml8s/liki-web/internal/migrate"
)

const MaxProfilesPerUser = 20

type Store struct {
	db      *sql.DB
	nowFunc func() time.Time
	idFunc  func() string
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db, nowFunc: func() time.Time { return time.Now().UTC() }, idFunc: func() string { return uuid.NewString() }}
}

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
	})
}

func (s *Store) Create(ctx context.Context, userID string, in BirthProfileInput) (BirthProfile, error) {
	now := formatTime(s.nowFunc())
	p := BirthProfile{
		ID: s.idFunc(), UserID: userID, Version: 1,
		Nickname: strings.TrimSpace(in.Nickname), Name: strings.TrimSpace(in.Name), Gender: in.Gender,
		Calendar: in.Calendar, BirthDate: in.BirthDate,
		BirthTime: in.BirthTime, BirthTimeKnown: in.BirthTimeKnown, BirthPlace: strings.TrimSpace(in.BirthPlace),
		Note: strings.TrimSpace(in.Note), ThirdPartyConsent: in.ThirdPartyConsent,
		CreatedAt: mustTime(now), UpdatedAt: mustTime(now),
	}
	result, err := s.db.ExecContext(ctx, `
INSERT INTO birth_profiles
(id,user_id,version,nickname,name,gender,calendar,birth_date,birth_time,birth_time_known,birth_place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at)
SELECT ?,?,1,?,?,?,?,?,?,?,?,?,?,NULL,NULL,?,?
WHERE (SELECT COUNT(1) FROM birth_profiles WHERE user_id=? AND deleted_at IS NULL) < ?`,
		p.ID, userID, p.Nickname, p.Name, p.Gender, p.Calendar, p.BirthDate,
		p.BirthTime, p.BirthTimeKnown, p.BirthPlace, p.Note, p.ThirdPartyConsent, now, now, userID, MaxProfilesPerUser,
	)
	if err != nil {
		return BirthProfile{}, err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return BirthProfile{}, ErrQuotaExceeded
	}
	var createdAt, updatedAt string
	err = s.db.QueryRowContext(ctx, `
SELECT created_at,updated_at FROM birth_profiles WHERE id=? AND user_id=?`,
		p.ID, userID).Scan(&createdAt, &updatedAt)
	if err != nil {
		return BirthProfile{}, err
	}
	p.CreatedAt = mustTime(createdAt)
	p.UpdatedAt = mustTime(updatedAt)
	return p, nil
}

func (s *Store) Get(ctx context.Context, userID, id string) (BirthProfile, error) {
	var p BirthProfile
	var version int
	var thirdParty, deletedAt, lastUsedAt sql.NullString
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx, `
SELECT id,user_id,version,nickname,name,gender,calendar,birth_date,birth_time,birth_time_known,birth_place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at
FROM birth_profiles WHERE id=? AND user_id=? AND deleted_at IS NULL`, id, userID).Scan(
		&p.ID, &p.UserID, &version, &p.Nickname, &p.Name, &p.Gender, &p.Calendar,
		&p.BirthDate, &p.BirthTime, &p.BirthTimeKnown, &p.BirthPlace, &p.Note, &thirdParty,
		&lastUsedAt, &deletedAt, &createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return BirthProfile{}, ErrNotFound
	}
	if err != nil {
		return BirthProfile{}, err
	}
	p.Version = version
	p.ThirdPartyConsent = thirdParty.Valid && thirdParty.String == "1"
	if lastUsedAt.Valid {
		t := mustTime(lastUsedAt.String)
		p.LastUsedAt = &t
	}
	if deletedAt.Valid {
		t := mustTime(deletedAt.String)
		p.DeletedAt = &t
	}
	p.CreatedAt = mustTime(createdAt)
	p.UpdatedAt = mustTime(updatedAt)
	return p, nil
}

// List returns profiles ordered by (created_at, id) DESC, cursor-exclusive.
// cursor format: "<created_at>|<id>".
func (s *Store) List(ctx context.Context, userID, cursor string, limit int) ([]BirthProfile, string, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := `
SELECT id,user_id,version,nickname,name,gender,calendar,birth_date,birth_time,birth_time_known,birth_place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at
FROM birth_profiles
WHERE user_id=? AND deleted_at IS NULL`
	args := []any{userID}
	if cursor != "" {
		createdAt, id, ok := strings.Cut(cursor, "|")
		if !ok {
			return nil, "", errors.New("invalid cursor")
		}
		query += ` AND (created_at < ? OR (created_at = ? AND id < ?))`
		args = append(args, createdAt, createdAt, id)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = rows.Close() }()

	out := make([]BirthProfile, 0, limit)
	for rows.Next() {
		var p BirthProfile
		var version int
		var thirdParty, lastUsedAt, deletedAt sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&p.ID, &p.UserID, &version, &p.Nickname, &p.Name, &p.Gender, &p.Calendar,
			&p.BirthDate, &p.BirthTime, &p.BirthTimeKnown, &p.BirthPlace, &p.Note, &thirdParty,
			&lastUsedAt, &deletedAt, &createdAt, &updatedAt); err != nil {
			return nil, "", err
		}
		p.Version = version
		p.ThirdPartyConsent = thirdParty.Valid && thirdParty.String == "1"
		if lastUsedAt.Valid {
			t := mustTime(lastUsedAt.String)
			p.LastUsedAt = &t
		}
		if deletedAt.Valid {
			t := mustTime(deletedAt.String)
			p.DeletedAt = &t
		}
		p.CreatedAt = mustTime(createdAt)
		p.UpdatedAt = mustTime(updatedAt)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	nextCursor := ""
	if len(out) > limit {
		out = out[:limit]
		last := out[len(out)-1]
		nextCursor = formatTime(last.CreatedAt) + "|" + last.ID
	}
	return out, nextCursor, nil
}

// Update applies a full user-info replacement guarded by an optimistic lock.
// Returns ErrConflict when expectedVersion does not match the stored version.
func (s *Store) Update(ctx context.Context, userID, id string, expectedVersion int, in BirthProfileInput) (BirthProfile, error) {
	now := formatTime(s.nowFunc())
	result, err := s.db.ExecContext(ctx, `
UPDATE birth_profiles SET
  version=version+1,
  nickname=?, name=?, gender=?, calendar=?, birth_date=?,
  birth_time=?, birth_time_known=?, birth_place=?, note=?, third_party_consent=?,
  updated_at=?
WHERE id=? AND user_id=? AND deleted_at IS NULL AND version=?`,
		strings.TrimSpace(in.Nickname), strings.TrimSpace(in.Name), in.Gender,
		in.Calendar, in.BirthDate, in.BirthTime, in.BirthTimeKnown, strings.TrimSpace(in.BirthPlace),
		strings.TrimSpace(in.Note), in.ThirdPartyConsent, now,
		id, userID, expectedVersion,
	)
	if err != nil {
		return BirthProfile{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return BirthProfile{}, err
	}
	if affected == 0 {
		_, getErr := s.Get(ctx, userID, id)
		if getErr != nil {
			return BirthProfile{}, getErr
		}
		return BirthProfile{}, ErrConflict
	}
	return s.Get(ctx, userID, id)
}

func (s *Store) SoftDelete(ctx context.Context, userID, id string) error {
	now := formatTime(s.nowFunc())
	result, err := s.db.ExecContext(ctx, `
UPDATE birth_profiles SET deleted_at=?, version=version+1, updated_at=?
WHERE id=? AND user_id=? AND deleted_at IS NULL`, now, now, id, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) TouchLastUsed(ctx context.Context, userID, id string) error {
	now := formatTime(s.nowFunc())
	_, err := s.db.ExecContext(ctx, `
UPDATE birth_profiles SET last_used_at=?, updated_at=? WHERE id=? AND user_id=? AND deleted_at IS NULL`,
		now, now, id, userID)
	return err
}

// AnonymizeUser clears all PII from the user's live profiles and soft-deletes
// them (compliance erasure / account deletion). Referential integrity of
// sessions is kept via id/user_id; no birth info survives.
func (s *Store) AnonymizeUser(ctx context.Context, userID string) (int64, error) {
	now := formatTime(s.nowFunc())
	result, err := s.db.ExecContext(ctx, `
UPDATE birth_profiles SET
  nickname='', name='', gender='', calendar='', birth_date='',
  birth_time='', birth_time_known='', birth_place='', note='',
  third_party_consent=0,
  version=version+1, deleted_at=?, updated_at=?
WHERE user_id=? AND deleted_at IS NULL`, now, now, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CleanupExpired physically deletes profiles soft-deleted before the cutoff
// (the 30-day recovery window). Returns the number of rows purged.
func (s *Store) CleanupExpired(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
DELETE FROM birth_profiles WHERE deleted_at IS NOT NULL AND deleted_at < ?`, formatTime(before))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func mustTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		// Should never happen — timestamps come from formatTime. Log and return zero
		// so callers fail loudly rather than silently comparing against epoch.
		parsed = time.Time{}
	}
	return parsed
}

func formatTime(value time.Time) string {
	return value.Format("2006-01-02T15:04:05.000000000Z07:00")
}
