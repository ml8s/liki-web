package adminauth

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
)

var (
	ErrAdminNotFound           = errors.New("adminauth: admin not found")
	ErrAdminAlreadyExists      = errors.New("adminauth: admin already exists")
	ErrInvalidCapability       = errors.New("adminauth: invalid capability")
	ErrSuperRequiresSuper      = errors.New("adminauth: admin.super changes require an active admin.super holder")
	ErrInvalidAdminStatus      = errors.New("adminauth: invalid admin status")
	ErrReasonRequired          = errors.New("adminauth: reason is required")
	ErrSelfManagementForbidden = errors.New("adminauth: self management is forbidden")
	ErrLastSuperAdmin          = errors.New("adminauth: last active super admin cannot be changed")
)

// CapabilityRegistry is the only source of grantable capabilities. It prevents
// an authorization payload from creating a typo that silently grants nothing.
func CapabilityRegistry() []Capability {
	return []Capability{
		CapabilityEntitlementRead,
		CapabilityEntitlementManage,
		CapabilityPartnerRead,
		CapabilityPartnerApprove,
		CapabilityPartnerFreeze,
		CapabilityPartnerUnfreeze,
		CapabilityCommissionRead,
		CapabilityCommissionVoid,
		CapabilityPayoutRead,
		CapabilityPayoutApprove,
		CapabilityPayoutReject,
		CapabilityPayoutMarkPaid,
		CapabilityAuditRead,
		CapabilityAdminUserManage,
		CapabilityAdminSuper,
	}
}

func IsCapabilityValid(capability Capability) bool {
	return slices.Contains(CapabilityRegistry(), capability)
}

// actorHoldsSuperTx reports whether the acting admin currently holds an
// active admin.super capability. Granting or revoking admin.super is a
// privileged operation precisely because admin.user.manage alone would
// otherwise enable a full privilege-escalation path (create a controlled
// admin, then promote it to super).
func actorHoldsSuperTx(ctx context.Context, tx *sql.Tx, actorEmail string) (bool, error) {
	var n int
	err := tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM admin_capabilities
WHERE admin_email=? AND capability=? AND status='active'`,
		actorEmail, string(CapabilityAdminSuper)).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

type AdminCapability struct {
	Capability Capability `json:"capability"`
	Status     string     `json:"status"`
	GrantedBy  string     `json:"granted_by"`
	GrantedAt  string     `json:"granted_at"`
	RevokedAt  string     `json:"revoked_at,omitempty"`
}

type AdminAccount struct {
	ID           string            `json:"id"`
	Email        string            `json:"email"`
	UserID       string            `json:"user_id,omitempty"`
	Status       AdminStatus       `json:"status"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
	Capabilities []AdminCapability `json:"capabilities"`
}

// EnsureAdmin registers a future administrator email. A unified account is
// bound only after that email successfully logs in and Authorize verifies it.
func (s *Store) EnsureAdmin(ctx context.Context, actorEmail, email, reason string) error {
	actorEmail = normalizeEmail(actorEmail)
	email = normalizeEmail(email)
	if actorEmail == "" || email == "" || strings.TrimSpace(reason) == "" {
		return ErrReasonRequired
	}
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM admin_users WHERE email=?`, email).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return ErrAdminAlreadyExists
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	now := formatTime(s.now())
	if _, err := tx.ExecContext(ctx, `
INSERT INTO admin_users (id,email,status,created_at,updated_at)
VALUES (?,?,?, ?,?)`, s.id(), email, string(AdminActive), now, now); err != nil {
		return err
	}
	if err := auditAdminActionTx(ctx, tx, s.id(), actorEmail, "admin.created", "admin", email,
		"", string(AdminActive), reason); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ListAdmins(ctx context.Context) ([]AdminAccount, error) {
	adminRows, err := s.db.QueryContext(ctx, `
SELECT id,email,COALESCE(user_id,''),status,created_at,updated_at
FROM admin_users ORDER BY created_at,email`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = adminRows.Close() }()
	accounts := make([]AdminAccount, 0)
	byEmail := make(map[string]int)
	for adminRows.Next() {
		var item AdminAccount
		if err := adminRows.Scan(&item.ID, &item.Email, &item.UserID, &item.Status,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Capabilities = make([]AdminCapability, 0)
		accounts = append(accounts, item)
		byEmail[item.Email] = len(accounts) - 1
	}
	if err := adminRows.Err(); err != nil {
		return nil, err
	}
	_ = adminRows.Close()

	capabilityRows, err := s.db.QueryContext(ctx, `
SELECT admin_email,capability,status,granted_by,granted_at,COALESCE(revoked_at,'')
FROM admin_capabilities
ORDER BY capability`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = capabilityRows.Close() }()
	for capabilityRows.Next() {
		var email string
		var item AdminCapability
		if err := capabilityRows.Scan(&email, &item.Capability, &item.Status,
			&item.GrantedBy, &item.GrantedAt, &item.RevokedAt); err != nil {
			return nil, err
		}
		if index, ok := byEmail[email]; ok {
			accounts[index].Capabilities = append(accounts[index].Capabilities, item)
		}
	}
	return accounts, capabilityRows.Err()
}

func (s *Store) GrantCapability(ctx context.Context, actorEmail, targetEmail string, capability Capability, reason string) error {
	actorEmail = normalizeEmail(actorEmail)
	targetEmail = normalizeEmail(targetEmail)
	if err := validateManagement(actorEmail, targetEmail, reason); err != nil {
		return err
	}
	if !IsCapabilityValid(capability) {
		return ErrInvalidCapability
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if capability == CapabilityAdminSuper {
		held, err := actorHoldsSuperTx(ctx, tx, actorEmail)
		if err != nil {
			return err
		}
		if !held {
			return ErrSuperRequiresSuper
		}
	}
	if err := requireAdminTx(ctx, tx, targetEmail); err != nil {
		return err
	}
	var before string
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(status,'missing') FROM admin_capabilities
WHERE admin_email=? AND capability=?`, targetEmail, string(capability)).Scan(&before); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	now := formatTime(s.now())
	if _, err := tx.ExecContext(ctx, `
INSERT INTO admin_capabilities
(admin_email,capability,status,granted_by,granted_at)
VALUES (?,?,'active',?,?)
ON CONFLICT(admin_email,capability) DO UPDATE SET
  status='active',granted_by=excluded.granted_by,granted_at=excluded.granted_at,revoked_at=NULL`,
		targetEmail, string(capability), actorEmail, now); err != nil {
		return err
	}
	if err := auditAdminActionTx(ctx, tx, s.id(), actorEmail, "admin.capability.granted", "admin_capability",
		capabilityKey(targetEmail, capability), before, "active", reason); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) RevokeCapability(ctx context.Context, actorEmail, targetEmail string, capability Capability, reason string) error {
	actorEmail = normalizeEmail(actorEmail)
	targetEmail = normalizeEmail(targetEmail)
	if err := validateManagement(actorEmail, targetEmail, reason); err != nil {
		return err
	}
	if !IsCapabilityValid(capability) {
		return ErrInvalidCapability
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if capability == CapabilityAdminSuper {
		held, err := actorHoldsSuperTx(ctx, tx, actorEmail)
		if err != nil {
			return err
		}
		if !held {
			return ErrSuperRequiresSuper
		}
	}

	var before string
	if err := tx.QueryRowContext(ctx, `
SELECT status FROM admin_capabilities WHERE admin_email=? AND capability=?`,
		targetEmail, string(capability)).Scan(&before); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAdminNotFound
		}
		return err
	}
	if before == "revoked" {
		return nil
	}
	if capability == CapabilityAdminSuper && before == "active" {
		if err := ensureNotLastSuperTx(ctx, tx, targetEmail); err != nil {
			return err
		}
	}
	now := formatTime(s.now())
	if _, err := tx.ExecContext(ctx, `
UPDATE admin_capabilities SET status='revoked',revoked_at=?
WHERE admin_email=? AND capability=?`, now, targetEmail, string(capability)); err != nil {
		return err
	}
	// admin_capabilities has no updated_at column; revoked_at and audit record
	// the mutation time/actor.
	if err := auditAdminActionTx(ctx, tx, s.id(), actorEmail, "admin.capability.revoked", "admin_capability",
		capabilityKey(targetEmail, capability), before, "revoked", reason); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) SetAdminStatus(ctx context.Context, actorEmail, targetEmail string, status AdminStatus, reason string) error {
	actorEmail = normalizeEmail(actorEmail)
	targetEmail = normalizeEmail(targetEmail)
	if err := validateManagement(actorEmail, targetEmail, reason); err != nil {
		return err
	}
	if status != AdminActive && status != AdminDisabled {
		return ErrInvalidAdminStatus
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var before string
	err = tx.QueryRowContext(ctx, `SELECT status FROM admin_users WHERE email=?`, targetEmail).Scan(&before)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAdminNotFound
	}
	if err != nil {
		return err
	}
	if before == string(status) {
		return nil
	}
	if status == AdminDisabled && before == string(AdminActive) {
		if err := ensureNotLastSuperTx(ctx, tx, targetEmail); err != nil {
			return err
		}
	}
	now := formatTime(s.now())
	if _, err := tx.ExecContext(ctx, `
UPDATE admin_users SET status=?,updated_at=? WHERE email=?`,
		string(status), now, targetEmail); err != nil {
		return err
	}
	action := "admin.enabled"
	if status == AdminDisabled {
		action = "admin.disabled"
	}
	if err := auditAdminActionTx(ctx, tx, s.id(), actorEmail, action, "admin", targetEmail,
		before, string(status), reason); err != nil {
		return err
	}
	return tx.Commit()
}

func validateManagement(actorEmail, targetEmail, reason string) error {
	if actorEmail == "" || targetEmail == "" {
		return ErrAdminNotFound
	}
	if actorEmail == targetEmail {
		return ErrSelfManagementForbidden
	}
	if strings.TrimSpace(reason) == "" {
		return ErrReasonRequired
	}
	return nil
}

func requireAdminTx(ctx context.Context, tx *sql.Tx, email string) error {
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM admin_users WHERE email=?`, email).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return ErrAdminNotFound
	}
	return nil
}

func ensureNotLastSuperTx(ctx context.Context, tx *sql.Tx, targetEmail string) error {
	var activeSupers int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(1)
FROM admin_users u
JOIN admin_capabilities c ON c.admin_email=u.email AND c.capability=? AND c.status='active'
WHERE u.status=? AND u.email<>?`,
		string(CapabilityAdminSuper), string(AdminActive), targetEmail).Scan(&activeSupers); err != nil {
		return err
	}
	if activeSupers == 0 {
		return ErrLastSuperAdmin
	}
	return nil
}

func auditAdminActionTx(ctx context.Context, tx *sql.Tx, id, actorEmail, action, objectType, objectID, before, after, reason string) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO admin_audit_logs
(id,actor_email,action,object_type,object_id,before,after,reason,created_at)
VALUES (?,?,?,?,?,?,?,?,?)`,
		id, actorEmail, action, objectType, objectID, nullable(before),
		nullable(after), nullable(strings.TrimSpace(reason)), formatTime(nowUTC()))
	return err
}

func capabilityKey(email string, capability Capability) string {
	return email + ":" + string(capability)
}
