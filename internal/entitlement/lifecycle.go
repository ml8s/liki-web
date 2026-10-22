package entitlement

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/ml8s/liki-web/internal/product"
)

func (s *Store) AllowedSkills(ctx context.Context, userID string) ([]product.Skill, error) {

	items, err := s.ListActiveByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if slices.Contains(item.Skills, product.SkillConsultation) {
			return []product.Skill{product.SkillConsultation}, nil
		}
	}
	return []product.Skill{}, nil
}

func (s *Store) HasSkill(ctx context.Context, userID string, skill product.Skill) (bool, error) {
	if skill != product.SkillConsultation {
		return false, nil
	}
	skills, err := s.AllowedSkills(ctx, userID)
	if err != nil {
		return false, err
	}
	return len(skills) > 0, nil
}

func (s *Store) ExpireDue(ctx context.Context) (int64, error) {
	now := s.nowFunc().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
UPDATE entitlements SET status='expired', updated_at=?
WHERE status='active' AND expires_at<=?`, now, now)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// AnonymizeUser removes contact email from entitlement records while retaining
// the entitlement term and pseudonymous user id for support/audit.
func (s *Store) AnonymizeUser(ctx context.Context, userID, oldEmail, tombstone string) (int64, error) {
	if userID == "" || tombstone == "" {
		return 0, fmt.Errorf("entitlement: anonymize requires user and tombstone")
	}
	res, err := s.db.ExecContext(ctx, `
UPDATE entitlements SET email=?
WHERE user_id=? OR (? != '' AND email=?)`, tombstone, userID, oldEmail, oldEmail)
	if err != nil {
		return 0, fmt.Errorf("entitlement: anonymize entitlements: %w", err)
	}
	return res.RowsAffected()
}

// ProvisionProduct adapts grant behavior to the sales domain's port.
func (s *Store) ProvisionProduct(ctx context.Context, userID, email string, productID product.Product, source, sourceID string, duration time.Duration) error {
	_, err := s.GrantForProduct(ctx, userID, email, productID, source, sourceID, duration)
	return err
}

// ProvisionSubscriptionPeriod adapts subscription grant behavior to the sales domain's port.
func (s *Store) ProvisionSubscriptionPeriod(ctx context.Context, userID, email string, productID product.Product, sourceID string, startsAt, expiresAt time.Time) error {
	_, err := s.GrantSubscriptionPeriod(ctx, userID, email, productID, sourceID, startsAt, expiresAt)
	return err
}
