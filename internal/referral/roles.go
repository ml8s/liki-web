package referral

import (
	"context"
	"database/sql"
	"strings"
)

func normalizeRole(role PartnerRole) PartnerRole {
	return PartnerRole(strings.ToLower(strings.TrimSpace(string(role))))
}

func (s *Store) partnerByRole(ctx context.Context, userID string, role PartnerRole) (Referrer, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Referrer{}, ErrPartnerNotFound
	}
	row := s.db.QueryRowContext(ctx, `
SELECT r.user_id,r.code,pr.role,r.status,r.recruiter_id,r.created_at,r.updated_at
FROM referrers r
JOIN partner_roles pr ON pr.user_id=r.user_id
WHERE r.user_id=? AND pr.role=? AND pr.status='active'`,
		userID, string(normalizeRole(role)))
	return scanReferrer(row.Scan)
}

func (s *Store) KOLByID(ctx context.Context, userID string) (Referrer, error) {
	return s.partnerByRole(ctx, userID, RoleKOL)
}

func (s *Store) ScoutByID(ctx context.Context, userID string) (Referrer, error) {
	return s.partnerByRole(ctx, userID, RoleScout)
}

func (s *Store) RequireKOL(ctx context.Context, userID string) (Referrer, error) {
	return s.KOLByID(ctx, userID)
}

func (s *Store) RequireScout(ctx context.Context, userID string) (Referrer, error) {
	return s.ScoutByID(ctx, userID)
}

func (s *Store) CountAttributions(ctx context.Context, referrerID string) (int64, error) {
	return s.CountAttributed(ctx, referrerID)
}

func (s *Store) HasRole(ctx context.Context, userID string, role PartnerRole) (bool, error) {
	_, err := s.partnerByRole(ctx, userID, role)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
