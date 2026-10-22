package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// LoginCodeRequest is the normalized request for a login verification code.
type LoginCodeRequest struct {
	Email     string
	IPHash    string
	UserAgent string
}

// LoginVerification is the normalized request to exchange a code for a session.
type LoginVerification struct {
	Email     string
	Code      string
	IPHash    string
	UserAgent string
	Locale    string
}

// LoginCodeRecord is the persisted state required to verify one code.
type LoginCodeRecord struct {
	ID        string
	CodeHash  []byte
	Attempts  int
	MaxTry    int
	ExpiresAt time.Time
}

// LoginCodeDecision is the pure outcome of code verification. It tells the
// transactional boundary whether the code must be consumed and what attempt
// count should be persisted.
type LoginCodeDecision struct {
	Attempts int
	Consume  bool
	Err      error
}

// issueLoginCode creates the credential and expiration from configured policy.
func issueLoginCode(now time.Time, generator func() (string, error)) (code string, hash []byte, expires time.Time, err error) {
	code, err = generator()
	if err != nil {
		return "", nil, time.Time{}, err
	}
	hash, err = bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, time.Time{}, err
	}
	return code, hash, now.Add(10 * time.Minute), nil
}

// decideLoginCode is the single authority for expiration, lockout, attempt
// counting, and successful consumption.
func decideLoginCode(now time.Time, code string, record LoginCodeRecord) LoginCodeDecision {
	if now.After(record.ExpiresAt) {
		return LoginCodeDecision{Consume: true, Attempts: record.Attempts, Err: ErrCodeExpired}
	}
	if record.Attempts >= record.MaxTry {
		return LoginCodeDecision{Consume: true, Attempts: record.Attempts, Err: ErrCodeLocked}
	}
	if bcrypt.CompareHashAndPassword(record.CodeHash, []byte(code)) != nil {
		return LoginCodeDecision{Consume: false, Attempts: record.Attempts + 1, Err: ErrCodeInvalid}
	}
	return LoginCodeDecision{Consume: true, Attempts: record.Attempts, Err: nil}
}

func (s *Store) IssueEmailCode(ctx context.Context, request LoginCodeRequest) (string, time.Time, error) {
	now := s.nowFunc()
	code, codeHash, expires, err := issueLoginCode(now, s.codeFun)
	if err != nil {
		return "", time.Time{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", time.Time{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.userIsLoginable(ctx, tx, request.Email); err != nil {
		return "", time.Time{}, err
	}

	var latestCreated string
	err = tx.QueryRowContext(ctx, `
SELECT created_at FROM auth_codes
WHERE email=? AND used_at IS NULL
ORDER BY created_at DESC LIMIT 1`, request.Email).Scan(&latestCreated)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", time.Time{}, err
	}
	latest := mustTime(latestCreated)
	if err == nil && latest.Add(s.config.CodeCooldown).After(now) {
		return "", time.Time{}, ErrCodeCooldown
	}
	if _, err = tx.ExecContext(ctx, `
UPDATE auth_codes SET used_at=? WHERE email=? AND used_at IS NULL`,
		now.Format(time.RFC3339), request.Email); err != nil {
		return "", time.Time{}, err
	}

	if _, err = tx.ExecContext(ctx, `
INSERT INTO auth_codes
(id,email,code_hash,purpose,attempts,max_attempts,ip_hash,user_agent,created_at,expires_at)
VALUES (?,?,?,'login',0,?,?,?,?,?)`,
		s.idFunc(), request.Email, string(codeHash), s.config.CodeMaxTry,
		request.IPHash, request.UserAgent, now.Format(time.RFC3339), expires.Format(time.RFC3339)); err != nil {
		return "", time.Time{}, err
	}
	if err = s.audit(ctx, tx, "email_code_issued", "", request.Email, request.IPHash, request.UserAgent, "{}", now); err != nil {
		return "", time.Time{}, err
	}
	if err = tx.Commit(); err != nil {
		return "", time.Time{}, err
	}
	return code, expires, nil
}

func (s *Store) CompleteEmailLogin(ctx context.Context, request LoginVerification) (LoginResult, error) {
	now := s.nowFunc()
	token, err := newSessionToken()
	if err != nil {
		return LoginResult{}, err
	}
	sessionID := s.idFunc()
	expires := now.Add(s.config.SessionTTL)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return LoginResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if err = s.userIsLoginable(ctx, tx, request.Email); err != nil {
		return LoginResult{}, err
	}

	record, err := loginCodeRecordFor(ctx, tx, request.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return LoginResult{}, ErrCodeInvalid
	}
	if err != nil {
		return LoginResult{}, err
	}
	decision := decideLoginCode(now, request.Code, record)
	if decision.Consume {
		_, _ = tx.ExecContext(ctx, `UPDATE auth_codes SET used_at=?,attempts=? WHERE id=?`,
			now.Format(time.RFC3339), decision.Attempts, record.ID)
	}
	if decision.Err != nil {
		if decision.Consume {
			if err = tx.Commit(); err != nil {
				return LoginResult{}, err
			}
			return LoginResult{}, decision.Err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE auth_codes SET attempts=? WHERE id=?`, decision.Attempts, record.ID); err != nil {
			return LoginResult{}, err
		}
		if err = tx.Commit(); err != nil {
			return LoginResult{}, err
		}
		return LoginResult{}, decision.Err
	}

	user, isNewUser, err := s.upsertEmailUser(ctx, tx, request.Email, request.Locale, now)
	if err != nil {
		return LoginResult{}, err
	}
	if err = s.upsertEmailIdentity(ctx, tx, user.ID, request.Email, now); err != nil {
		return LoginResult{}, err
	}

	nowText := now.Format(time.RFC3339)
	if _, err = tx.ExecContext(ctx, `
UPDATE auth_codes SET used_at=? WHERE id=?`, nowText, record.ID); err != nil {
		return LoginResult{}, err
	}

	session := newAuthSession(sessionID, user.ID, token, request.IPHash, request.UserAgent, now, expires)
	if _, err = tx.ExecContext(ctx, `
INSERT INTO auth_sessions
(id,user_id,token_hash,status,ip_hash,user_agent,created_at,expires_at,last_seen_at)
VALUES (?,?,?, 'active',?,?,?,? ,?)`,
		session.ID, user.ID, session.TokenHash, session.IPHash, session.UserAgent,
		nowText, expires.Format(time.RFC3339), nowText); err != nil {
		return LoginResult{}, err
	}

	// Enforce the per-user session cap: keep the newest SessionCap sessions
	// (the just-created one is always retained) and purge older ones along
	// with their ip_hash/user_agent metadata.
	if err = s.enforceSessionCap(ctx, tx, user.ID, sessionID); err != nil {
		return LoginResult{}, err
	}

	if err = s.audit(ctx, tx, "login_succeeded", user.ID, request.Email, request.IPHash, request.UserAgent, "{}", now); err != nil {
		return LoginResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{User: user, Session: session, SessionToken: token, IsNewUser: isNewUser}, nil
}

// newSessionToken creates the browser bearer secret. Only its hash is stored.
func newSessionToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(tokenBytes), nil
}

func (s *Store) sessionTokenHash(token string) string {
	sum := sha256.Sum256([]byte("auth_session:" + token))
	return hex.EncodeToString(sum[:])
}

// loginCodeRecordFor returns the active verification code for an email.
func loginCodeRecordFor(ctx context.Context, q rowQuerier, email string) (LoginCodeRecord, error) {
	var record LoginCodeRecord
	var codeExpires string
	err := q.QueryRowContext(context.Background(), `
SELECT id,code_hash,attempts,max_attempts,expires_at
FROM auth_codes
WHERE email=? AND used_at IS NULL
ORDER BY created_at DESC LIMIT 1`, email).
		Scan(&record.ID, &record.CodeHash, &record.Attempts, &record.MaxTry, &codeExpires)
	if err != nil {
		return LoginCodeRecord{}, err
	}
	record.ExpiresAt = mustTime(codeExpires)
	return record, nil
}

func sessionTokenHash(token string) string {
	sum := sha256.Sum256([]byte("auth_session:" + token))
	return hex.EncodeToString(sum[:])
}

func newAuthSession(
	id, userID, token, ipHash, userAgent string,
	now, expires time.Time,
) AuthSession {
	return AuthSession{
		ID: id, UserID: userID, TokenHash: sessionTokenHash(token),
		Status: statusActive, IPHash: ipHash, UserAgent: userAgent,
		ExpiresAt: expires, CreatedAt: now, LastSeenAt: &now,
	}
}

func (s *Store) upsertEmailUser(ctx context.Context, tx *sql.Tx, email, locale string, now time.Time) (User, bool, error) {
	nowText := now.Format(time.RFC3339)
	newUserID := s.idFunc()
	var user User
	var dbUserID, status, resultLocale string
	var createdAt, updatedAt string
	var lastLogin, deleted sql.NullString
	var existingUserID string
	isNewUser := false
	switch err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE email=?`, email).Scan(&existingUserID); {
	case errors.Is(err, sql.ErrNoRows):
		isNewUser = true
	case err != nil:
		return User{}, false, err
	default:
		newUserID = existingUserID
	}
	err := tx.QueryRowContext(ctx, `
INSERT INTO users (id,email,status,locale,created_at,updated_at,last_login_at)
VALUES (?,?,?, ?,?,?,?)
ON CONFLICT(email) DO UPDATE SET
  updated_at=excluded.updated_at,
  last_login_at=excluded.last_login_at,
  status=CASE WHEN users.status='deleted' THEN 'deleted' ELSE 'active' END
RETURNING id,status,locale,created_at,updated_at,last_login_at,deleted_at`,
		newUserID, email, string(UserActive), locale, nowText, nowText, nowText,
		email, nowText, nowText, nowText,
	).Scan(&dbUserID, &status, &resultLocale, &createdAt, &updatedAt, &lastLogin, &deleted)
	if err != nil {
		return User{}, false, err
	}
	user.ID = dbUserID
	user.Email = email
	user.Status = UserStatus(status)
	user.Locale = resultLocale
	user.CreatedAt = mustTime(createdAt)
	user.UpdatedAt = mustTime(updatedAt)
	if lastLogin.Valid {
		t := mustTime(lastLogin.String)
		user.LastLoginAt = &t
	}
	if deleted.Valid {
		t := mustTime(deleted.String)
		user.DeletedAt = &t
	}
	return user, isNewUser, nil
}

func (s *Store) upsertEmailIdentity(ctx context.Context, tx *sql.Tx, userID, email string, now time.Time) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO auth_identities (id,user_id,provider,provider_user_id,email,created_at,last_used_at)
VALUES (?,?,?,?,?,?,?)
ON CONFLICT(provider, provider_user_id) DO UPDATE SET last_used_at=excluded.last_used_at`,
		s.idFunc(), userID, string(ProviderEmail), email, email, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("upsert email identity: %w", err)
	}
	return nil
}

func (s *Store) enforceSessionCap(ctx context.Context, tx *sql.Tx, userID, currentSessionID string) error {
	if s.config.MaxSessions <= 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
DELETE FROM auth_sessions
WHERE user_id=? AND id<>? AND id NOT IN (
	SELECT id FROM auth_sessions
	WHERE user_id=? AND id<>?
	ORDER BY created_at DESC, id DESC LIMIT ?
)`, userID, currentSessionID, userID, currentSessionID, s.config.MaxSessions-1)
	if err != nil {
		return fmt.Errorf("enforce session cap: %w", err)
	}
	return nil
}
