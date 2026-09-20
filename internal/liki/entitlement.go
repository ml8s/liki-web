package liki

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

// EntitlementStatus is the lifecycle state of a capability grant.
type EntitlementStatus string

const (
	StatusActive  EntitlementStatus = "active"
	StatusRevoked EntitlementStatus = "revoked"
	StatusExpired EntitlementStatus = "expired"
)

// Entitlement records a user's right to use one or more skills.
type Entitlement struct {
	ID        string
	UserID    string
	Email     string
	ProductID Product
	Skills    []Skill
	Status    EntitlementStatus
	Source    string
	SourceID  string
	StartsAt  time.Time
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (e Entitlement) MarshalJSON() ([]byte, error) {
	skills := make([]string, 0, len(e.Skills))
	for _, skill := range e.Skills {
		skills = append(skills, string(skill))
	}
	return json.Marshal(struct {
		ID        string    `json:"id"`
		UserID    string    `json:"user_id"`
		Email     string    `json:"email"`
		ProductID string    `json:"product_id"`
		Skills    []string  `json:"skills"`
		Status    string    `json:"status"`
		StartsAt  time.Time `json:"starts_at"`
		ExpiresAt time.Time `json:"expires_at"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}{
		ID: e.ID, UserID: e.UserID, Email: e.Email, ProductID: string(e.ProductID),
		Skills: skills, Status: string(e.Status), StartsAt: e.StartsAt, ExpiresAt: e.ExpiresAt,
		CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	})
}

// EntitlementStore manages the products and entitlements tables in the liki
// product database. It stores platform user IDs as plain TEXT soft references.
type EntitlementStore struct {
	db      *sql.DB
	nowFunc func() time.Time
	idFunc  func() string
}

func NewEntitlementStore(db *sql.DB) *EntitlementStore {
	return &EntitlementStore{db: db, nowFunc: func() time.Time { return time.Now().UTC() }, idFunc: uuid.NewString}
}

func (s *EntitlementStore) Migrate(ctx context.Context) error {
	return migrate.Run(ctx, s.db, []migrate.Migration{{
		// Idempotent schema creation. Never DROP existing tables to
		// avoid destroying production entitlement data on restart.
		ID: "002_single_chat_product",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    skills TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS entitlements (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    email TEXT NOT NULL,
    product_id TEXT NOT NULL,
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
CREATE INDEX IF NOT EXISTS idx_entitlements_user_status_expires
    ON entitlements(user_id, status, expires_at);
CREATE INDEX IF NOT EXISTS idx_entitlements_source
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
INSERT OR IGNORE INTO products (id,name,skills,active) VALUES (?,?,?,1)`,
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
func (s *EntitlementStore) GrantForProduct(
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
		return Entitlement{}, fmt.Errorf("liki: grant requires user, email, source, and source id")
	}
	if duration <= 0 {
		return Entitlement{}, fmt.Errorf("liki: grant duration must be positive")
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
		return Entitlement{}, fmt.Errorf("liki: grant: %w", err)
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
	entitlement.Status = EntitlementStatus(status)
	entitlement.StartsAt = entitlementMustTime(startsAt)
	entitlement.ExpiresAt = entitlementMustTime(expiresAt)
	entitlement.CreatedAt = entitlementMustTime(createdAt)
	entitlement.UpdatedAt = entitlementMustTime(updatedAt)
	return entitlement, nil
}

func entitlementMustTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339, value)
	return parsed
}

func (s *EntitlementStore) latestForSource(ctx context.Context, userID string, productID Product, source, sourceID string) (Entitlement, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id,user_id,email,product_id,skills,status,source,source_id,starts_at,expires_at,created_at,updated_at
FROM entitlements
WHERE user_id=? AND product_id=? AND source=? AND source_id=?
ORDER BY created_at DESC LIMIT 1`,
		userID, string(productID), source, sourceID)
	return scanEntitlement(row.Scan)
}

func (s *EntitlementStore) ListActiveByUser(ctx context.Context, userID string) ([]Entitlement, error) {
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

func (s *EntitlementStore) AllowedSkills(ctx context.Context, userID string) ([]Skill, error) {
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

func (s *EntitlementStore) HasSkill(ctx context.Context, userID string, skill Skill) (bool, error) {
	if skill != SkillLiki {
		return false, nil
	}
	skills, err := s.AllowedSkills(ctx, userID)
	if err != nil {
		return false, err
	}
	return len(skills) > 0, nil
}

func (s *EntitlementStore) ExpireDue(ctx context.Context) (int64, error) {
	now := s.nowFunc().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
UPDATE entitlements SET status='expired', updated_at=?
WHERE status='active' AND expires_at<=?`, now, now)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
