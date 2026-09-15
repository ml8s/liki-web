package entitlement

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"liki-web/internal/migrate"
)

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
		// This is an unreleased runtime boundary, not a compatibility release:
		// old report-product grants must not leak into the unified Chat model.
		ID: "002_single_chat_product",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
DROP TABLE IF EXISTS entitlements;
DROP TABLE IF EXISTS products;

CREATE TABLE products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    skills TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE entitlements (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    product_id TEXT NOT NULL REFERENCES products(id),
    skills TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active','revoked','expired')),
    source TEXT NOT NULL,
    source_id TEXT NOT NULL,
    starts_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(source, source_id, product_id)
);
CREATE INDEX idx_entitlements_user_status_expires
    ON entitlements(user_id, status, expires_at);
CREATE INDEX idx_entitlements_source
    ON entitlements(source, source_id);
`)
			if err != nil {
				return err
			}
			for _, definition := range ProductCatalog() {
				skillsJSON, err := json.Marshal(definition.Skills)
				if err != nil {
					return err
				}
				if _, err := tx.ExecContext(ctx, `
INSERT INTO products (id,name,skills,active) VALUES (?,?,?,1)`,
					string(definition.ID), definition.Name, string(skillsJSON)); err != nil {
					return err
				}
			}
			return nil
		},
	}})
}

// GrantForProduct grants a commercial Chat product to an existing platform
// user. User creation belongs to auth; entitlement never synthesizes users
// from payment webhooks.
func (s *Store) GrantForProduct(
	ctx context.Context,
	userID string,
	email string,
	productID Product,
	source string,
	sourceID string,
	duration time.Duration,
) (Entitlement, error) {
	userID = strings.TrimSpace(userID)
	email = strings.ToLower(strings.TrimSpace(email))
	source = strings.TrimSpace(source)
	sourceID = strings.TrimSpace(sourceID)
	definition, err := ProductDefinitionFor(productID)
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
	if currentExpires.Valid && mustTime(currentExpires.String).After(now) {
		startsAt = mustTime(currentExpires.String)
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

func scanEntitlement(scan func(dest ...any) error) (Entitlement, error) {
	var entitlement Entitlement
	var skillsJSON, status, startsAt, expiresAt, createdAt, updatedAt string
	err := scan(
		&entitlement.ID, &entitlement.UserID, &entitlement.Email, &entitlement.ProductID,
		&skillsJSON, &status, &entitlement.Source, &entitlement.SourceID,
		&startsAt, &expiresAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return Entitlement{}, err
	}
	if err := json.Unmarshal([]byte(skillsJSON), &entitlement.Skills); err != nil {
		return Entitlement{}, err
	}
	entitlement.Status = Status(status)
	entitlement.StartsAt = mustTime(startsAt)
	entitlement.ExpiresAt = mustTime(expiresAt)
	entitlement.CreatedAt = mustTime(createdAt)
	entitlement.UpdatedAt = mustTime(updatedAt)
	return entitlement, nil
}

func mustTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339, value)
	return parsed
}

func (s *Store) latestForSource(ctx context.Context, userID string, productID Product, source, sourceID string) (Entitlement, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id,user_id,email,product_id,skills,status,source,source_id,starts_at,expires_at,created_at,updated_at
FROM entitlements
WHERE user_id=? AND product_id=? AND source=? AND source_id=?
ORDER BY created_at DESC LIMIT 1`,
		userID, string(productID), source, sourceID)
	return scanEntitlement(row.Scan)
}

func (s *Store) ListActiveByUser(ctx context.Context, userID string) ([]Entitlement, error) {
	now := s.nowFunc().Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx, `
SELECT id,user_id,email,product_id,skills,status,source,source_id,starts_at,expires_at,created_at,updated_at
FROM entitlements
WHERE user_id=? AND status='active' AND starts_at<=? AND expires_at>?
ORDER BY expires_at DESC, created_at DESC`, userID, now, now)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]Entitlement, 0)
	for rows.Next() {
		item, scanErr := scanEntitlement(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) AllowedSkills(ctx context.Context, userID string) ([]Skill, error) {
	items, err := s.ListActiveByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		for _, skill := range item.Skills {
			if skill == SkillLiki {
				return []Skill{SkillLiki}, nil
			}
		}
	}
	return []Skill{}, nil
}

func (s *Store) HasSkill(ctx context.Context, userID string, skill Skill) (bool, error) {
	if skill != SkillLiki {
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

// NewEntitlementStore is an explicit constructor alias used by callers that
// prefer a descriptive store name.
func NewEntitlementStore(db *sql.DB) *Store { return NewStore(db) }
