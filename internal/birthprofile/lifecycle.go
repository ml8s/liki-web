package birthprofile

import (
	"context"
	"time"
)

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
  nickname='', name='', gender='', place='', place_id='', location_version=0, longitude=NULL, timezone=NULL, instant=NULL, note='',
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
