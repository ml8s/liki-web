// Package adminauth owns platform administrator identity capabilities and
// audit records. Regular app authentication remains in internal/auth.
package adminauth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/migrate"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

type Capability string

const (
	CapabilityEntitlementRead   Capability = "entitlement.read"
	CapabilityEntitlementManage Capability = "entitlement.manage"
	CapabilityPartnerRead       Capability = "partner.read"
	CapabilityPartnerApprove    Capability = "partner.approve"
	CapabilityPartnerFreeze     Capability = "partner.freeze"
	CapabilityPartnerUnfreeze   Capability = "partner.unfreeze"
	CapabilityCommissionRead    Capability = "commission.read"
	CapabilityCommissionVoid    Capability = "commission.void"
	CapabilityPayoutRead        Capability = "payout.read"
	CapabilityPayoutApprove     Capability = "payout.approve"
	CapabilityPayoutReject      Capability = "payout.reject"
	CapabilityPayoutMarkPaid    Capability = "payout.mark_paid"
	CapabilityAuditRead         Capability = "audit.read"
	CapabilityAdminUserManage   Capability = "admin.user.manage"
	CapabilityAdminSuper        Capability = "admin.super"
)

type AdminStatus string

const (
	AdminActive   AdminStatus = "active"
	AdminDisabled AdminStatus = "disabled"
)

type AuditEntry struct {
	ActorUserID string    `json:"actor_user_id,omitempty"`
	ActorEmail  string    `json:"actor_email"`
	Capability  string    `json:"capability,omitempty"`
	Action      string    `json:"action"`
	ObjectType  string    `json:"object_type"`
	ObjectID    string    `json:"object_id"`
	Before      string    `json:"before,omitempty"`
	After       string    `json:"after,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	RequestID   string    `json:"request_id,omitempty"`
	IPHash      string    `json:"ip_hash,omitempty"`
	UserAgent   string    `json:"user_agent,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type AuditItem struct {
	ID         string    `json:"id"`
	ActorEmail string    `json:"actor_email"`
	Action     string    `json:"action"`
	ObjectType string    `json:"object_type"`
	ObjectID   string    `json:"object_id"`
	Reason     string    `json:"reason,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

var ErrAdminDisabled = errors.New("adminauth: admin is disabled")

type Store struct {
	db  *sql.DB
	now func() time.Time
	id  func() string
}

// Open opens the platform database through the shared SQLite policy.
func Open(path string) (*Store, error) {
	db, err := sqliteutil.Open(path)
	if err != nil {
		return nil, err
	}
	return &Store{db: db, now: func() time.Time { return time.Now().UTC() }, id: uuid.NewString}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// AnonymizeAuditLogs tombstones the acting admin's email on historical audit
// rows during account erasure. The pseudonymous actor_user_id, capability,
// action, and object references are preserved for security-audit continuity.
func (s *Store) AnonymizeAuditLogs(ctx context.Context, email, tombstone string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
UPDATE admin_audit_logs SET actor_email=? WHERE actor_email=?`, tombstone, email)
	if err != nil {
		return 0, fmt.Errorf("adminauth: anonymize audit logs: %w", err)
	}
	return res.RowsAffected()
}

func (s *Store) Migrate(ctx context.Context) error {
	return migrate.Run(ctx, s.db, []migrate.Migration{{
		ID: "001_admin_auth",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS admin_users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    user_id TEXT UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('active','disabled')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS admin_capabilities (
    admin_email TEXT NOT NULL,
    capability TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active','revoked')),
    granted_by TEXT NOT NULL DEFAULT 'system',
    granted_at TEXT NOT NULL,
    revoked_at TEXT,
    PRIMARY KEY (admin_email,capability)
);
CREATE TABLE IF NOT EXISTS admin_audit_logs (
    id TEXT PRIMARY KEY,
    actor_user_id TEXT,
    actor_email TEXT NOT NULL,
    capability TEXT,
    action TEXT NOT NULL,
    object_type TEXT NOT NULL,
    object_id TEXT NOT NULL,
    before TEXT,
    after TEXT,
    reason TEXT,
    request_id TEXT,
    ip_hash TEXT,
    user_agent TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_admin_audit_created
    ON admin_audit_logs(created_at);
`)
			return err
		},
	}})
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *Store) BootstrapAdminByEmail(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	if email == "" {
		return errors.New("adminauth: email is required")
	}
	now := formatTime(s.now())
	// Startup is idempotent. It must never reactivate a disabled admin or a
	// revoked capability; those are deliberate security decisions.
	if _, err := s.db.ExecContext(ctx, `
INSERT INTO admin_users (id,email,status,created_at,updated_at)
VALUES (?,?,?, ?,?)
ON CONFLICT(email) DO NOTHING`,
		s.id(), email, string(AdminActive), now, now); err != nil {
		return err
	}
	return s.ensureCapabilityByEmail(ctx, email, CapabilityAdminSuper)
}

func (s *Store) ensureCapabilityByEmail(ctx context.Context, email string, capability Capability) error {
	email = normalizeEmail(email)
	if email == "" || capability == "" {
		return errors.New("adminauth: email and capability are required")
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO admin_capabilities (admin_email,capability,status,granted_by,granted_at)
VALUES (?,?, 'active','system',?)
ON CONFLICT(admin_email,capability) DO NOTHING`,
		email, string(capability), formatTime(s.now()))
	return err
}

// Authorize binds a unified session identity to an admin record, then checks
// that the bound identity is active and holds the requested capability.
func (s *Store) Authorize(ctx context.Context, user auth.User, capability Capability) (bool, error) {
	email := normalizeEmail(user.Email)
	if email == "" || user.ID == "" || user.Status != auth.UserActive {
		return false, nil
	}

	var adminID, boundUserID, adminStatus string
	err := s.db.QueryRowContext(ctx, `
SELECT id,COALESCE(user_id,''),status
FROM admin_users WHERE email=?`, email).Scan(&adminID, &boundUserID, &adminStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if adminStatus != string(AdminActive) {
		return false, nil
	}
	if boundUserID != "" && boundUserID != user.ID {
		return false, nil
	}

	if boundUserID == "" {
		if _, err := s.db.ExecContext(ctx, `
UPDATE admin_users SET user_id=?,updated_at=? WHERE id=?`,
			user.ID, formatTime(s.now()), adminID); err != nil {
			return false, err
		}
	}

	var capabilityStatus string
	if capability == CapabilityAdminSuper {
		err = s.db.QueryRowContext(ctx, `
SELECT status FROM admin_capabilities
WHERE admin_email=? AND capability=?`, email, string(CapabilityAdminSuper)).Scan(&capabilityStatus)
	} else {
		err = s.db.QueryRowContext(ctx, `
SELECT CASE
  WHEN super_capability.status='active' THEN 'active'
  ELSE COALESCE(capability.status,'revoked')
END
FROM admin_users u
LEFT JOIN admin_capabilities AS super_capability
  ON super_capability.admin_email=u.email AND super_capability.capability=?
LEFT JOIN admin_capabilities AS capability
  ON capability.admin_email=u.email AND capability.capability=?
WHERE u.email=?`, string(CapabilityAdminSuper), string(capability), email).Scan(&capabilityStatus)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return capabilityStatus == "active", nil
}

// Capabilities returns the effective capabilities for a bound, active admin.
// It is used to render capability-scoped navigation and never grants access by
// itself: every data route still performs its own Authorize check.
func (s *Store) Capabilities(ctx context.Context, user auth.User) ([]Capability, error) {
	email := normalizeEmail(user.Email)
	if email == "" || user.ID == "" || user.Status != auth.UserActive {
		return []Capability{}, nil
	}

	var adminID, boundUserID, adminStatus string
	err := s.db.QueryRowContext(ctx, `
SELECT id,COALESCE(user_id,''),status
FROM admin_users WHERE email=?`, email).Scan(&adminID, &boundUserID, &adminStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return []Capability{}, nil
	}
	if err != nil {
		return nil, err
	}
	if adminStatus != string(AdminActive) {
		return []Capability{}, nil
	}
	if boundUserID != "" && boundUserID != user.ID {
		return []Capability{}, nil
	}
	if boundUserID == "" {
		if _, err := s.db.ExecContext(ctx, `
UPDATE admin_users SET user_id=?,updated_at=? WHERE id=?`,
			user.ID, formatTime(s.now()), adminID); err != nil {
			return nil, err
		}
	}

	var superActive int
	if err := s.db.QueryRowContext(ctx, `
SELECT COUNT(1) FROM admin_capabilities
WHERE admin_email=? AND capability=? AND status='active'`,
		email, string(CapabilityAdminSuper)).Scan(&superActive); err != nil {
		return nil, err
	}
	if superActive > 0 {
		return CapabilityRegistry(), nil
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT capability FROM admin_capabilities
WHERE admin_email=? AND status='active' ORDER BY capability`, email)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	capabilities := make([]Capability, 0)
	for rows.Next() {
		var capability Capability
		if err := rows.Scan(&capability); err != nil {
			return nil, err
		}
		if IsCapabilityValid(capability) {
			capabilities = append(capabilities, capability)
		}
	}
	return capabilities, rows.Err()
}

func (s *Store) Audit(ctx context.Context, entry AuditEntry) error {
	entry.ActorEmail = normalizeEmail(entry.ActorEmail)
	entry.CreatedAt = s.now()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO admin_audit_logs
(id,actor_user_id,actor_email,capability,action,object_type,object_id,
 before,after,reason,request_id,ip_hash,user_agent,created_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		s.id(), nullable(entry.ActorUserID), entry.ActorEmail, nullable(entry.Capability),
		entry.Action, entry.ObjectType, entry.ObjectID, nullable(entry.Before),
		nullable(entry.After), nullable(entry.Reason), nullable(entry.RequestID),
		nullable(entry.IPHash), nullable(entry.UserAgent), formatTime(entry.CreatedAt))
	if err != nil {
		return fmt.Errorf("adminauth: audit: %w", err)
	}
	return nil
}

func (s *Store) ListAudit(ctx context.Context, limit int) ([]AuditItem, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id,actor_email,action,object_type,object_id,COALESCE(reason,''),created_at
FROM admin_audit_logs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]AuditItem, 0)
	for rows.Next() {
		var item AuditItem
		var createdAt string
		if err := rows.Scan(&item.ID, &item.ActorEmail, &item.Action,
			&item.ObjectType, &item.ObjectID, &item.Reason, &createdAt); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTime(createdAt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func nowUTC() time.Time { return time.Now().UTC() }

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func formatTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func parseTime(v string) time.Time {
	t, _ := time.Parse(time.RFC3339, v)
	return t
}
