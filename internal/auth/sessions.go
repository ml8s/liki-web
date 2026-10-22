package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

func (s *Store) GetSessionUser(ctx context.Context, token string) (User, AuthSession, error) {
	tokenHash := s.sessionTokenHash(token)
	row := s.db.QueryRowContext(ctx, `
SELECT u.id,u.email,u.status,u.locale,u.created_at,u.updated_at,u.last_login_at,u.deleted_at,
       s.id,s.token_hash,s.status,s.ip_hash,s.user_agent,s.created_at,s.expires_at,s.revoked_at,s.last_seen_at
FROM auth_sessions s
JOIN users u ON u.id=s.user_id
WHERE s.token_hash=?`, tokenHash)

	var user User
	var session AuthSession
	var userStatus, userLocale, userCreatedAt, userUpdatedAt string
	var userLastLogin, userDeleted sql.NullString
	var sessionCreatedAt, sessionExpiresAt string
	var sessionIPHash, sessionUserAgent, sessionRevoked, sessionLastSeen sql.NullString
	err := row.Scan(
		&user.ID, &user.Email, &userStatus, &userLocale, &userCreatedAt, &userUpdatedAt, &userLastLogin, &userDeleted,
		&session.ID, &session.TokenHash, &session.Status, &sessionIPHash, &sessionUserAgent,
		&sessionCreatedAt, &sessionExpiresAt, &sessionRevoked, &sessionLastSeen,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, AuthSession{}, ErrSessionNotFound
	}
	if err != nil {
		return User{}, AuthSession{}, err
	}
	user.Status = UserStatus(userStatus)
	user.Locale = userLocale
	user.CreatedAt = mustTime(userCreatedAt)
	user.UpdatedAt = mustTime(userUpdatedAt)
	if userLastLogin.Valid {
		t := mustTime(userLastLogin.String)
		user.LastLoginAt = &t
	}
	if userDeleted.Valid {
		t := mustTime(userDeleted.String)
		user.DeletedAt = &t
	}
	if sessionRevoked.Valid {
		t := mustTime(sessionRevoked.String)
		session.RevokedAt = &t
	}
	if sessionLastSeen.Valid {
		t := mustTime(sessionLastSeen.String)
		session.LastSeenAt = &t
	}
	if sessionIPHash.Valid {
		session.IPHash = sessionIPHash.String
	}
	if sessionUserAgent.Valid {
		session.UserAgent = sessionUserAgent.String
	}
	session.CreatedAt = mustTime(sessionCreatedAt)
	session.ExpiresAt = mustTime(sessionExpiresAt)
	now := s.nowFunc()
	if user.Status != UserActive || session.Status != statusActive || now.After(session.ExpiresAt) {
		return User{}, AuthSession{}, ErrSessionExpired
	}
	return user, session, nil
}

func (s *Store) TouchSession(ctx context.Context, sessionID string, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE auth_sessions SET last_seen_at=? WHERE id=?`, now.Format(time.RFC3339), sessionID)
	return err
}

func (s *Store) ListIdentities(ctx context.Context, userID string) ([]AuthIdentity, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id,user_id,provider,provider_user_id,email,created_at,last_used_at
FROM auth_identities WHERE user_id=? ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []AuthIdentity
	for rows.Next() {
		var identity AuthIdentity
		var createdAt string
		var lastUsed sql.NullString
		if err := rows.Scan(&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderUID, &identity.Email, &createdAt, &lastUsed); err != nil {
			return nil, err
		}
		identity.CreatedAt = mustTime(createdAt)
		if lastUsed.Valid {
			t := mustTime(lastUsed.String)
			identity.LastUsedAt = &t
		}
		out = append(out, identity)
	}
	return out, rows.Err()
}

func (s *Store) ListSessions(ctx context.Context, userID string) ([]AuthSession, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id,user_id,token_hash,status,ip_hash,user_agent,created_at,expires_at,revoked_at,last_seen_at
FROM auth_sessions
WHERE user_id=? AND status='active' AND expires_at>?
ORDER BY created_at DESC`, userID, s.nowFunc().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []AuthSession
	for rows.Next() {
		var session AuthSession
		var createdAt, expiresAt string
		var ipHash, userAgent, revoked, lastSeen sql.NullString
		if err := rows.Scan(&session.ID, &session.UserID, &session.TokenHash, &session.Status, &ipHash, &userAgent, &createdAt, &expiresAt, &revoked, &lastSeen); err != nil {
			return nil, err
		}
		if ipHash.Valid {
			session.IPHash = ipHash.String
		}
		if userAgent.Valid {
			session.UserAgent = userAgent.String
		}
		session.CreatedAt = mustTime(createdAt)
		session.ExpiresAt = mustTime(expiresAt)
		if revoked.Valid {
			t := mustTime(revoked.String)
			session.RevokedAt = &t
		}
		if lastSeen.Valid {
			t := mustTime(lastSeen.String)
			session.LastSeenAt = &t
		}
		out = append(out, session)
	}
	return out, rows.Err()
}

func (s *Store) RevokeSession(ctx context.Context, userID, sessionID string) error {
	res, err := s.db.ExecContext(ctx, `
UPDATE auth_sessions SET status=?, revoked_at=?
WHERE user_id=? AND id=? AND status='active'`,
		statusRevoked, s.nowFunc().Format(time.RFC3339), userID, sessionID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (s *Store) RevokeAllSessions(ctx context.Context, userID, exceptSessionID string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
UPDATE auth_sessions SET status=?, revoked_at=?
WHERE user_id=? AND status='active' AND id!=?`,
		statusRevoked, s.nowFunc().Format(time.RFC3339), userID, exceptSessionID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Cleanup expires stale sessions, then purges expired auth codes and
// expired/revoked sessions past the PII retention window (compliance:
// configurable via LIKI_SESSION_RETENTION_DAYS).
func (s *Store) Cleanup(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays < 1 {
		retentionDays = 30
	}
	modifier := fmt.Sprintf("-%d days", retentionDays)
	now := s.nowFunc().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
UPDATE auth_sessions SET status='expired'
WHERE status='active' AND expires_at<=?`, now)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if _, err = s.db.ExecContext(ctx, `
DELETE FROM auth_codes WHERE expires_at < datetime('now', ?)`, modifier); err != nil {
		return n, err
	}
	if _, err = s.db.ExecContext(ctx, `
DELETE FROM auth_sessions
WHERE status IN ('expired','revoked') AND expires_at <= datetime('now', ?)`, modifier); err != nil {
		return n, err
	}
	return n, err
}
