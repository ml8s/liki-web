package partner

import (
	"context"
	"strings"
)

// createKOLPartner is a test fixture for the approved KOL capability. It
// intentionally bypasses application review.
// CreateKOLPartner creates the KOL role directly. Production applications use
// SubmitApplication and admin approval; this helper is available for migrations
// and trusted seed flows that must bypass an application decision.
func (s *Store) createKOLPartner(ctx context.Context, userID string) (Partner, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Partner{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := ensurePartnerTx(ctx, tx, s, strings.TrimSpace(userID), RoleKOL); err != nil {
		return Partner{}, err
	}
	if err := tx.Commit(); err != nil {
		return Partner{}, err
	}
	return s.PartnerByUser(ctx, strings.TrimSpace(userID))
}

func (s *Store) commission(ctx context.Context, provider, paymentID string) (Commission, error) {
	return s.commissionTx(ctx, s.db, provider, paymentID)
}
