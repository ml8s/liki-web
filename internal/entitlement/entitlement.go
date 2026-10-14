package entitlement

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ml8s/liki-web/internal/migrate"
	"github.com/ml8s/liki-web/internal/product"
)

// EntitlementStatus is the lifecycle state of a capability grant.
type EntitlementStatus string

const (
	Active  EntitlementStatus = "active"
	Revoked EntitlementStatus = "revoked"
	Expired EntitlementStatus = "expired"
)

var (
	ErrEntitlementNotFound          = errors.New("liki: entitlement not found")
	ErrEntitlementAlreadyRevoked    = errors.New("liki: entitlement already revoked")
	ErrRevokedEntitlementImmutable  = errors.New("liki: revoked entitlement cannot be changed")
	ErrEntitlementSearchUnsupported = errors.New("liki: entitlement search requires an email")
)

// Entitlement records a user's right to use one or more skills.
type Entitlement struct {
	ID        string
	UserID    string
	Email     string
	ProductID product.Product
	Skills    []product.Skill
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

// Store manages the products and entitlements tables in the liki
// product database. It stores platform user IDs as plain TEXT soft references.
type Store struct {
	db      *sql.DB
	nowFunc func() time.Time
	idFunc  func() string
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db, nowFunc: func() time.Time { return time.Now().UTC() }, idFunc: uuid.NewString}
}

func (s *Store) Migrate(ctx context.Context) error {
	return migrate.Run(ctx, s.db, []migrate.Migration{{
		// Idempotent schema creation. Never DROP existing tables to
		// avoid destroying production entitlement data on restart.
		ID: "002_single_chat_product",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
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

func (s *Store) latestForSource(ctx context.Context, userID string, productID product.Product, source, sourceID string) (Entitlement, error) {
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

// ListForAdmin is a narrowly scoped support lookup. It requires an email so
// the admin API cannot accidentally become an unbounded customer listing.
func (s *Store) ListForAdmin(ctx context.Context, email, status string, limit int) ([]Entitlement, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	status = strings.ToLower(strings.TrimSpace(status))
	if email == "" {
		return nil, ErrEntitlementSearchUnsupported
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := `
SELECT id,user_id,email,product_id,skills,status,source,source_id,starts_at,expires_at,created_at,updated_at
FROM entitlements WHERE email=?`
	args := []any{email}
	if status != "" {
		if status != string(Active) && status != string(Revoked) && status != string(Expired) {
			return nil, fmt.Errorf("liki: invalid entitlement status %q", status)
		}
		query += ` AND status=?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]Entitlement, 0)
	for rows.Next() {
		item, scanErr := scanEntitlement(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) getForAdmin(ctx context.Context, id string) (Entitlement, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id,user_id,email,product_id,skills,status,source,source_id,starts_at,expires_at,created_at,updated_at
FROM entitlements WHERE id=?`, id)
	item, err := scanEntitlement(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return Entitlement{}, ErrEntitlementNotFound
	}
	return item, err
}

func (s *Store) UpdateExpiration(ctx context.Context, id string, expiresAt time.Time) (string, string, error) {
	current, err := s.getForAdmin(ctx, id)
	if err != nil {
		return "", "", err
	}
	if current.Status == Revoked {
		return "", "", ErrRevokedEntitlementImmutable
	}
	now := s.nowFunc()
	status := Active
	if !expiresAt.After(now) {
		status = Expired
	}
	updated := current
	updated.Status = status
	updated.ExpiresAt = expiresAt
	updated.UpdatedAt = now
	result, err := s.db.ExecContext(ctx, `
UPDATE entitlements SET status=?,expires_at=?,updated_at=?
WHERE id=? AND status<>?`, string(status), formatEntitlementTime(expiresAt),
		formatEntitlementTime(now), id, string(Revoked))
	if err != nil {
		return "", "", err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return "", "", ErrRevokedEntitlementImmutable
	}
	return marshalEntitlementAudit(current), marshalEntitlementAudit(updated), nil
}

func (s *Store) Revoke(ctx context.Context, id string) (string, string, error) {
	current, err := s.getForAdmin(ctx, id)
	if err != nil {
		return "", "", err
	}
	if current.Status == Revoked {
		return "", "", ErrEntitlementAlreadyRevoked
	}
	now := s.nowFunc()
	updated := current
	updated.Status = Revoked
	updated.UpdatedAt = now
	result, err := s.db.ExecContext(ctx, `
UPDATE entitlements SET status=?,updated_at=? WHERE id=?`, string(Revoked),
		formatEntitlementTime(now), id)
	if err != nil {
		return "", "", err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return "", "", ErrEntitlementNotFound
	}
	return marshalEntitlementAudit(current), marshalEntitlementAudit(updated), nil
}

func formatEntitlementTime(value time.Time) string {
	return value.Format(time.RFC3339)
}

func marshalEntitlementAudit(item Entitlement) string {
	payload, err := json.Marshal(struct {
		ID        string `json:"id"`
		UserID    string `json:"user_id"`
		Email     string `json:"email"`
		ProductID string `json:"product_id"`
		Status    string `json:"status"`
		ExpiresAt string `json:"expires_at"`
	}{
		ID: item.ID, UserID: item.UserID, Email: item.Email,
		ProductID: string(item.ProductID), Status: string(item.Status),
		ExpiresAt: item.ExpiresAt.Format(time.RFC3339),
	})
	if err != nil {
		return ""
	}
	return string(payload)
}

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
