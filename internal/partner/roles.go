package partner

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

func normalizeRole(role PartnerRole) PartnerRole {
	return PartnerRole(strings.ToLower(strings.TrimSpace(string(role))))
}

func (s *Store) partnerByRole(ctx context.Context, userID string, role PartnerRole) (Partner, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Partner{}, ErrPartnerNotFound
	}
	row := s.db.QueryRowContext(ctx, `
SELECT r.user_id,r.code,pr.role,r.status,r.recruiter_id,r.created_at,r.updated_at
FROM partners r
JOIN partner_roles pr ON pr.user_id=r.user_id
WHERE r.user_id=? AND pr.role=? AND pr.status='active'`,
		userID, string(normalizeRole(role)))
	return scanPartner(row.Scan)
}

func (s *Store) KOLByID(ctx context.Context, userID string) (Partner, error) {
	return s.partnerByRole(ctx, userID, RoleKOL)
}

func (s *Store) ScoutByID(ctx context.Context, userID string) (Partner, error) {
	return s.partnerByRole(ctx, userID, RoleScout)
}

// PartnerForPayout permits retired role capabilities for settling historical
// earnings, while the partner-level fraud freeze still blocks withdrawal.
func (s *Store) PartnerForPayout(ctx context.Context, userID string, role PartnerRole) (Partner, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Partner{}, ErrPartnerNotFound
	}
	row := s.db.QueryRowContext(ctx, `
SELECT r.user_id,r.code,pr.role,r.status,r.recruiter_id,r.created_at,r.updated_at
FROM partners r
JOIN partner_roles pr ON pr.user_id=r.user_id
WHERE r.user_id=? AND pr.role=? AND r.status='active'
  AND pr.status IN ('active','retired')`,
		userID, string(normalizeRole(role)))
	return scanPartner(row.Scan)
}

// ProgrammeRoleStatus resolves capability state without treating a historical
// retired capability as an active dashboard identity.
func (s *Store) ProgrammeRoleStatus(ctx context.Context, userID string, role PartnerRole) (RoleStatus, bool, error) {
	var status string
	err := s.db.QueryRowContext(ctx, `
SELECT pr.status
FROM partners r
JOIN partner_roles pr ON pr.user_id=r.user_id
WHERE r.user_id=? AND pr.role=?`,
		userID, string(normalizeRole(role))).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return RoleStatus(status), true, nil
}
