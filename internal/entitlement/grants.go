package entitlement

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ml8s/liki-web/internal/product"
)

func (s *Store) GrantForProduct(
	ctx context.Context,
	userID string,
	email string,
	productID product.Product,
	source string,
	sourceID string,
	duration time.Duration,
) (Entitlement, error) {
	userID = strings.TrimSpace(userID)
	email = strings.ToLower(strings.TrimSpace(email))
	source = strings.TrimSpace(source)
	sourceID = strings.TrimSpace(sourceID)
	definition, err := product.DefinitionFor(productID)
	if err != nil {
		return Entitlement{}, err
	}
	if userID == "" || email == "" || source == "" || sourceID == "" {
		return Entitlement{}, fmt.Errorf("entitlement: grant requires user, email, source, and source id")
	}
	if duration <= 0 {
		return Entitlement{}, fmt.Errorf("entitlement: grant duration must be positive")
	}

	now := s.nowFunc()
	skillsJSON, err := json.Marshal(definition.Skills)
	if err != nil {
		return Entitlement{}, err
	}
	startsAt := now
	// Repeated purchases of the same product stack after the current term.
	var currentExpires sql.NullString
	err = s.db.QueryRowContext(ctx, `
SELECT MAX(expires_at) FROM entitlements
WHERE user_id=? AND product_id=? AND status='active' AND expires_at>?`,
		userID, string(productID), now.Format(time.RFC3339)).Scan(&currentExpires)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Entitlement{}, err
	}
	if currentExpires.Valid && entitlementMustTime(currentExpires.String).After(now) {
		startsAt = entitlementMustTime(currentExpires.String)
	}
	expires := startsAt.Add(duration)

	id := s.idFunc()
	_, err = s.db.ExecContext(ctx, `
INSERT INTO entitlements
(id,user_id,email,product_id,skills,status,source,source_id,starts_at,expires_at,created_at,updated_at)
VALUES (?,?,?,?,?,'active',?,?,?,?,?,?)
ON CONFLICT(source,source_id,product_id) DO NOTHING`,
		id, userID, email, string(productID), string(skillsJSON), source, sourceID,
		startsAt.Format(time.RFC3339), expires.Format(time.RFC3339),
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return Entitlement{}, fmt.Errorf("entitlement: grant: %w", err)
	}
	return s.latestForSource(ctx, userID, productID, source, sourceID)
}

// GrantSubscriptionPeriod makes the provider-owned period authoritative. A
// repeated active/renewed webhook updates the same row instead of stacking a
// second grant after the current term.
func (s *Store) GrantSubscriptionPeriod(
	ctx context.Context,
	userID string,
	email string,
	productID product.Product,
	sourceID string,
	startsAt time.Time,
	expiresAt time.Time,
) (Entitlement, error) {
	userID = strings.TrimSpace(userID)
	email = strings.ToLower(strings.TrimSpace(email))
	sourceID = strings.TrimSpace(sourceID)
	definition, err := product.DefinitionFor(productID)
	if err != nil {
		return Entitlement{}, err
	}
	if userID == "" || email == "" || sourceID == "" {
		return Entitlement{}, fmt.Errorf("entitlement: subscription grant requires user, email, and source id")
	}
	if !startsAt.Before(expiresAt) {
		return Entitlement{}, fmt.Errorf("entitlement: subscription period start must precede end")
	}

	skillsJSON, err := json.Marshal(definition.Skills)
	if err != nil {
		return Entitlement{}, err
	}
	now := s.nowFunc()
	id := s.idFunc()
	_, err = s.db.ExecContext(ctx, `
INSERT INTO entitlements
(id,user_id,email,product_id,skills,status,source,source_id,starts_at,expires_at,created_at,updated_at)
VALUES (?,?,?,?,?,'active','subscription',?,?,?,?,?)
ON CONFLICT(source,source_id,product_id) DO UPDATE SET
    user_id=excluded.user_id,
    email=excluded.email,
    skills=excluded.skills,
    status='active',
    starts_at=excluded.starts_at,
    expires_at=excluded.expires_at,
    updated_at=excluded.updated_at`,
		id, userID, email, string(productID), string(skillsJSON), sourceID,
		startsAt.Format(time.RFC3339), expiresAt.Format(time.RFC3339),
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return Entitlement{}, fmt.Errorf("entitlement: grant subscription period: %w", err)
	}
	return s.latestForSource(ctx, userID, productID, "subscription", sourceID)
}
