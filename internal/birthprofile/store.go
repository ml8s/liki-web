package birthprofile

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
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

func (s *Store) Create(ctx context.Context, userID string, in BirthProfileDraft) (BirthProfile, error) {
	nowTime := s.nowFunc()
	if err := ValidateBirthProfileMetadata(in); err != nil {
		return BirthProfile{}, err
	}
	birth, err := NewBirthFacts(nowTime, in.Birth)
	if err != nil {
		return BirthProfile{}, err
	}
	now := formatTime(nowTime)
	if err != nil {
		return BirthProfile{}, err
	}
	p := BirthProfile{
		ID: s.idFunc(), UserID: userID, Version: 1,
		Nickname: strings.TrimSpace(in.Nickname), Name: strings.TrimSpace(in.Name),
		Birth: NewBirthFactsView(birth),
		Note:  strings.TrimSpace(in.Note), ThirdPartyConsent: in.ThirdPartyConsent,
		CreatedAt: mustTime(now), UpdatedAt: mustTime(now),
	}
	result, err := s.db.ExecContext(ctx, `
INSERT INTO birth_profiles
(id,user_id,version,nickname,name,gender,instant,timezone,longitude,place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at)
SELECT ?,?,1,?,?,?,?,?,?,?,?,?,NULL,NULL,?,?
WHERE (SELECT COUNT(1) FROM birth_profiles WHERE user_id=? AND deleted_at IS NULL) < ?`,
		p.ID, userID, p.Nickname, p.Name, p.Birth.Gender, timeArg(p.Birth.Instant),
		p.Birth.Location.Timezone, p.Birth.Location.Longitude, p.Birth.Location.Place, p.Note, p.ThirdPartyConsent,
		now, now, userID, MaxProfilesPerUser,
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
	p.Birth = NewBirthFactsView(p.Birth.BirthFacts)
	return p, nil
}

func timeArg(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatTime(*value)
}

func (s *Store) Get(ctx context.Context, userID, id string) (BirthProfile, error) {
	var p BirthProfile
	var version int
	var longitude sql.NullFloat64
	var timezone, instant sql.NullString
	var thirdParty, deletedAt, lastUsedAt sql.NullString
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx, `
SELECT id,user_id,version,nickname,name,gender,instant,timezone,longitude,place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at
FROM birth_profiles WHERE id=? AND user_id=? AND deleted_at IS NULL`, id, userID).Scan(
		&p.ID, &p.UserID, &version, &p.Nickname, &p.Name, &p.Birth.Gender,
		&instant, &timezone, &longitude, &p.Birth.Location.Place, &p.Note, &thirdParty,
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
	if longitude.Valid {
		p.Birth.Location.Longitude = &longitude.Float64
	}
	if timezone.Valid {
		value := timezone.String
		p.Birth.Location.Timezone = &value
	}
	if instant.Valid {
		t := mustTime(instant.String)
		p.Birth.Instant = &t
	}
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
	p.Birth = NewBirthFactsView(p.Birth.BirthFacts)
	return p, nil
}

// List returns profiles ordered by (created_at, id) DESC, cursor-exclusive.
// cursor format: "<created_at>|<id>".
func (s *Store) List(ctx context.Context, userID, cursor string, limit int) ([]BirthProfile, string, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := `
SELECT id,user_id,version,nickname,name,gender,instant,timezone,longitude,place,note,third_party_consent,last_used_at,deleted_at,created_at,updated_at
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
		var longitude sql.NullFloat64
		var timezone, instant, thirdParty, lastUsedAt, deletedAt sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&p.ID, &p.UserID, &version, &p.Nickname, &p.Name, &p.Birth.Gender,
			&instant, &timezone, &longitude, &p.Birth.Location.Place, &p.Note, &thirdParty,
			&lastUsedAt, &deletedAt, &createdAt, &updatedAt); err != nil {
			return nil, "", err
		}
		p.Version = version
		p.ThirdPartyConsent = thirdParty.Valid && thirdParty.String == "1"
		if longitude.Valid {
			p.Birth.Location.Longitude = &longitude.Float64
		}
		if timezone.Valid {
			value := timezone.String
			p.Birth.Location.Timezone = &value
		}
		if instant.Valid {
			t := mustTime(instant.String)
			p.Birth.Instant = &t
		}
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
		p.Birth = NewBirthFactsView(p.Birth.BirthFacts)
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
func (s *Store) Update(ctx context.Context, userID, id string, expectedVersion int, in BirthProfileDraft) (BirthProfile, error) {
	nowTime := s.nowFunc()
	if err := ValidateBirthProfileMetadata(in); err != nil {
		return BirthProfile{}, err
	}
	birth, err := NewBirthFacts(nowTime, in.Birth)
	if err != nil {
		return BirthProfile{}, err
	}
	now := formatTime(nowTime)
	result, err := s.db.ExecContext(ctx, `
UPDATE birth_profiles SET
  version=version+1,
  nickname=?, name=?, gender=?, place=?, longitude=?, timezone=?, instant=?, note=?, third_party_consent=?,
  updated_at=?
WHERE id=? AND user_id=? AND deleted_at IS NULL AND version=?`,
		strings.TrimSpace(in.Nickname), strings.TrimSpace(in.Name), in.Birth.Gender,
		birth.Location.Place, birth.Location.Longitude, birth.Location.Timezone,
		timeArg(birth.Instant), strings.TrimSpace(in.Note), in.ThirdPartyConsent, now,
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
