package auth

import (
	"context"
	"database/sql"
	"errors"

	"time"

	"github.com/google/uuid"
)

// ErasureResult reports how many auth records were removed. The user row is
// retained as a pseudonymous tombstone so financial/audit references never
// dangle; its login identity and contact PII are destroyed.
type ErasureResult struct {
	AlreadyErased bool
	Identities    int64
	Sessions      int64
	Codes         int64
	AuditRecords  int64
}

// EraseUser permanently removes login identity and contact PII. It is
// idempotent: a second request returns AlreadyErased without creating another
// tombstone. This method must be called only after product PII cleanup has
// succeeded, so a failed request leaves the login usable for retry/support.
func (s *Store) EraseUser(ctx context.Context, userID, tombstone string) (ErasureResult, error) {
	if userID == "" {
		return ErasureResult{}, errors.New("auth: user id is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ErasureResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var oldEmail, status string
	err = tx.QueryRowContext(ctx, `SELECT email,status FROM users WHERE id=?`, userID).Scan(&oldEmail, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErasureResult{}, ErrUserNotFound
	}
	if err != nil {
		return ErasureResult{}, err
	}
	if UserStatus(status) == UserDeleted {
		return ErasureResult{AlreadyErased: true}, tx.Commit()
	}

	if tombstone == "" {
		tombstone = "deleted-" + uuid.NewString() + "@liki.invalid"
	}
	now := s.nowFunc().Format(time.RFC3339Nano)

	// Login identities are the strongest link to the external provider; they
	// are not needed to retain the pseudonymous financial/audit record.
	identities, err := eraseCount(tx.ExecContext(ctx, `DELETE FROM auth_identities WHERE user_id=?`, userID))
	if err != nil {
		return ErasureResult{}, err
	}
	// Hard-delete sessions so tokens, IP hashes and user agents disappear while
	// guaranteedly revoking every active browser session.
	sessions, err := eraseCount(tx.ExecContext(ctx, `DELETE FROM auth_sessions WHERE user_id=?`, userID))
	if err != nil {
		return ErasureResult{}, err
	}
	codes, err := eraseCount(tx.ExecContext(ctx, `DELETE FROM auth_codes WHERE email=?`, oldEmail))
	if err != nil {
		return ErasureResult{}, err
	}

	// Replace users.email with an unlinked, non-deliverable tombstone. The
	// unique index remains usable without keeping the original address.
	if _, err := tx.ExecContext(ctx, `
UPDATE users
SET email=?,status=?,deleted_at=?,last_login_at=NULL,updated_at=?
WHERE id=?`, tombstone, string(UserDeleted), now, now, userID); err != nil {
		return ErasureResult{}, err
	}

	// Security events retain event type and pseudonymous user id, but lose the
	// original email, raw IP hash and browser fingerprint.
	auditRecords, err := eraseCount(tx.ExecContext(ctx, `
UPDATE auth_audit_logs
SET email=?,ip_hash=NULL,user_agent=NULL
WHERE user_id=? OR email=?`, tombstone, userID, oldEmail))
	if err != nil {
		return ErasureResult{}, err
	}

	if err := s.audit(ctx, tx, "account.erased", userID, tombstone, "", "", `{}`, s.nowFunc()); err != nil {
		return ErasureResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ErasureResult{}, err
	}
	return ErasureResult{
		Identities: identities, Sessions: sessions, Codes: codes, AuditRecords: auditRecords,
	}, nil
}

func eraseCount(result sql.Result, err error) (int64, error) {
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
