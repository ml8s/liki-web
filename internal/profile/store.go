package profile

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"liki-web/internal/migrate"
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
	return migrate.Run(ctx, s.db, []migrate.Migration{{
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
	}})
}

func (s *Store) Create(ctx context.Context, userID string, in Input) (Profile, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `
SELECT COUNT(1) FROM profiles WHERE user_id=? AND deleted_at IS NULL`, userID).Scan(&count); err != nil {
		return Profile{}, err
	}
	if count >= MaxProfilesPerUser {
		return Profile{}, ErrQuotaExceeded
	}

	now := formatTime(s.nowFunc())
	p := Profile{
		ID: s.idFunc(), UserID: userID, Version: 1,
		Nickname: strings.TrimSpace(in.Nickname), Name: strings.TrimSpace(in.Name), Gender: in.Gender,
		Relation: strings.TrimSpace(in.Relation), Calendar: in.Calendar, BirthDate: in.BirthDate,
		BirthTime: in.BirthTime, BirthTimeKnown: in.BirthTimeKnown, BirthPlace: strings.TrimSpace(in.BirthPlace),
		Note: strings.TrimSpace(in.Note), ThirdPartyConsent: in.ThirdPartyConsent,
		CreatedAt: mustTime(now), UpdatedAt: mustTime(now),
	}
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx, `
INSERT INTO profiles
(id,user_id,version,nickname,name,gender,relation,calendar,birth_date,birth_time,birth_time_known,birth_place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at)
VALUES (?,?,1,?,?,?,?,?,?,?,?,?,?,?,NULL,NULL,?,?)
RETURNING created_at,updated_at`,
		p.ID, userID, p.Nickname, p.Name, p.Gender, p.Relation, p.Calendar, p.BirthDate,
		p.BirthTime, p.BirthTimeKnown, p.BirthPlace, p.Note, p.ThirdPartyConsent, now, now,
	).Scan(&createdAt, &updatedAt)
	if err != nil {
		return Profile{}, err
	}
	p.CreatedAt = mustTime(createdAt)
	p.UpdatedAt = mustTime(updatedAt)
	return p, nil
}

func (s *Store) Get(ctx context.Context, userID, id string) (Profile, error) {
	var p Profile
	var version int
	var thirdParty, deletedAt, lastUsedAt sql.NullString
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx, `
SELECT id,user_id,version,nickname,name,gender,relation,calendar,birth_date,birth_time,birth_time_known,birth_place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at
FROM profiles WHERE id=? AND user_id=? AND deleted_at IS NULL`, id, userID).Scan(
		&p.ID, &p.UserID, &version, &p.Nickname, &p.Name, &p.Gender, &p.Relation, &p.Calendar,
		&p.BirthDate, &p.BirthTime, &p.BirthTimeKnown, &p.BirthPlace, &p.Note, &thirdParty,
		&lastUsedAt, &deletedAt, &createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, err
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

// List returns profiles ordered by (updated_at, id) DESC, cursor-exclusive.
// cursor format: "<updated_at>|<id>".
func (s *Store) List(ctx context.Context, userID, cursor string, limit int) ([]Profile, string, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := `
SELECT id,user_id,version,nickname,name,gender,relation,calendar,birth_date,birth_time,birth_time_known,birth_place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at
FROM profiles
WHERE user_id=? AND deleted_at IS NULL`
	args := []any{userID}
	if cursor != "" {
		updatedAt, id, ok := strings.Cut(cursor, "|")
		if !ok {
			return nil, "", errors.New("invalid cursor")
		}
		query += ` AND (updated_at < ? OR (updated_at = ? AND id < ?))`
		args = append(args, updatedAt, updatedAt, id)
	}
	query += ` ORDER BY updated_at DESC, id DESC LIMIT ?`
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = rows.Close() }()

	out := make([]Profile, 0, limit)
	for rows.Next() {
		var p Profile
		var version int
		var thirdParty, lastUsedAt, deletedAt sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&p.ID, &p.UserID, &version, &p.Nickname, &p.Name, &p.Gender, &p.Relation, &p.Calendar,
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
		nextCursor = formatTime(last.UpdatedAt) + "|" + last.ID
	}
	return out, nextCursor, nil
}

// Update applies a full user-info replacement guarded by an optimistic lock.
// Returns ErrConflict when expectedVersion does not match the stored version.
func (s *Store) Update(ctx context.Context, userID, id string, expectedVersion int, in Input) (Profile, error) {
	now := formatTime(s.nowFunc())
	result, err := s.db.ExecContext(ctx, `
UPDATE profiles SET
  version=version+1,
  nickname=?, name=?, gender=?, relation=?, calendar=?, birth_date=?,
  birth_time=?, birth_time_known=?, birth_place=?, note=?, third_party_consent=?,
  updated_at=?
WHERE id=? AND user_id=? AND deleted_at IS NULL AND version=?`,
		strings.TrimSpace(in.Nickname), strings.TrimSpace(in.Name), in.Gender, strings.TrimSpace(in.Relation),
		in.Calendar, in.BirthDate, in.BirthTime, in.BirthTimeKnown, strings.TrimSpace(in.BirthPlace),
		strings.TrimSpace(in.Note), in.ThirdPartyConsent, now,
		id, userID, expectedVersion,
	)
	if err != nil {
		return Profile{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Profile{}, err
	}
	if affected == 0 {
		_, getErr := s.Get(ctx, userID, id)
		if getErr != nil {
			return Profile{}, getErr
		}
		return Profile{}, ErrConflict
	}
	return s.Get(ctx, userID, id)
}

func (s *Store) SoftDelete(ctx context.Context, userID, id string) error {
	now := formatTime(s.nowFunc())
	result, err := s.db.ExecContext(ctx, `
UPDATE profiles SET deleted_at=?, version=version+1, updated_at=?
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
UPDATE profiles SET last_used_at=?, updated_at=? WHERE id=? AND user_id=? AND deleted_at IS NULL`,
		now, now, id, userID)
	return err
}

func mustTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed
}

func formatTime(value time.Time) string {
	return value.Format("2006-01-02T15:04:05.000000000Z07:00")
}
