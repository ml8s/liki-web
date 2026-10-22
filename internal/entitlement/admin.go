package entitlement

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

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
