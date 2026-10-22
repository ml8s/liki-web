package entitlement

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
