package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"liki-web/internal/migrate"
)

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`)

const (
	codeLength    = 6
	codeMaxTry    = 5
	statusActive  = "active"
	statusRevoked = "revoked"
)

type Store struct {
	db      *sql.DB
	hmacKey []byte
	nowFunc func() time.Time
	idFunc  func() string
	codeFun func() (string, error)
}

func NewStore(db *sql.DB, hmacKey []byte) *Store {
	return &Store{
		db:      db,
		hmacKey: hmacKey,
		nowFunc: func() time.Time { return time.Now().UTC() },
		idFunc:  func() string { return uuid.NewString() },
		codeFun: func() (string, error) {
			b := make([]byte, 3)
			if _, err := rand.Read(b); err != nil {
				return "", err
			}
			n := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
			return strings.TrimSpace(fmt.Sprintf("%06d", n%1000000)), nil
		},
	}
}

type LoginResult struct {
	User         User
	Session      AuthSession
	SessionToken string
	IsNewUser    bool
}

func (s *Store) Migrate(ctx context.Context) error {
	return migrate.Run(ctx, s.db, []migrate.Migration{{
		ID: "001_auth_identity_core",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE COLLATE NOCASE,
    status TEXT NOT NULL CHECK (status IN ('active','disabled','deleted')),
    locale TEXT NOT NULL DEFAULT 'zh-Hant',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    last_login_at TEXT,
    deleted_at TEXT
);

CREATE TABLE IF NOT EXISTS auth_identities (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK (provider IN ('email','google','apple')),
    provider_user_id TEXT NOT NULL,
    email TEXT NOT NULL,
    created_at TEXT NOT NULL,
    last_used_at TEXT,
    UNIQUE(provider, provider_user_id)
);
CREATE INDEX IF NOT EXISTS idx_auth_identities_user ON auth_identities(user_id);

CREATE TABLE IF NOT EXISTS auth_codes (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    code_hash TEXT NOT NULL,
    purpose TEXT NOT NULL DEFAULT 'login',
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    ip_hash TEXT,
    user_agent TEXT,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    used_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_auth_codes_email_created ON auth_codes(email, created_at);

CREATE TABLE IF NOT EXISTS auth_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','revoked','expired')),
    ip_hash TEXT,
    user_agent TEXT,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    revoked_at TEXT,
    last_seen_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_status ON auth_sessions(user_id, status, expires_at);

CREATE TABLE IF NOT EXISTS auth_audit_logs (
    id TEXT PRIMARY KEY,
    event TEXT NOT NULL,
    user_id TEXT,
    email TEXT,
    ip_hash TEXT,
    user_agent TEXT,
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_auth_audit_logs_email_created ON auth_audit_logs(email, created_at);
`)
			return err
		},
	}})
}

func NormalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || len(email) > 254 || !emailPattern.MatchString(email) {
		return "", ErrInvalidEmail
	}
	return email, nil
}

func (s *Store) sessionTokenHash(token string) string {
	sum := sha256.Sum256([]byte("auth_session:" + token))
	return hex.EncodeToString(sum[:])
}

func (s *Store) ipHash(ip string) string {
	if ip == "" {
		return ""
	}
	sum := hmac.New(sha256.New, s.hmacKey)
	sum.Write([]byte("auth_ip:" + ip))
	return hex.EncodeToString(sum.Sum(nil))
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

type rowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Store) audit(ctx context.Context, tx *sql.Tx, event, userID, email, ipHash, userAgent, metadata string, now time.Time) error {
	if metadata == "" {
		metadata = "{}"
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO auth_audit_logs
(id,event,user_id,email,ip_hash,user_agent,metadata,created_at)
VALUES (?,?,?,?,?,?,?,?)`,
		s.idFunc(), event, nullable(userID), nullable(email), nullable(ipHash),
		nullable(userAgent), metadata, now.Format(time.RFC3339))
	return err
}

func (s *Store) userIsLoginable(ctx context.Context, q rowQuerier, email string) error {
	var status UserStatus
	err := q.QueryRowContext(ctx, `SELECT status FROM users WHERE email=?`, email).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	switch status {
	case UserActive:
		return nil
	case UserDisabled:
		return ErrUserDisabled
	case UserDeleted:
		return ErrUserNotFound
	default:
		return ErrUserDisabled
	}
}

func (s *Store) IssueEmailCode(ctx context.Context, rawEmail, ipHash, userAgent string) (string, time.Time, error) {
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		return "", time.Time{}, err
	}
	now := s.nowFunc()
	code, err := s.codeFun()
	if err != nil {
		return "", time.Time{}, err
	}
	codeHash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", time.Time{}, err
	}
	expires := now.Add(10 * time.Minute)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", time.Time{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.userIsLoginable(ctx, tx, email); err != nil {
		return "", time.Time{}, err
	}

	var latestCreated string
	err = tx.QueryRowContext(ctx, `
SELECT created_at FROM auth_codes
WHERE email=? AND used_at IS NULL
ORDER BY created_at DESC LIMIT 1`, email).Scan(&latestCreated)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", time.Time{}, err
	}
	latest := mustTime(latestCreated)
	if err == nil && latest.Add(time.Minute).After(now) {
		return "", time.Time{}, ErrCodeCooldown
	}
	if _, err = tx.ExecContext(ctx, `
UPDATE auth_codes SET used_at=? WHERE email=? AND used_at IS NULL`,
		now.Format(time.RFC3339), email); err != nil {
		return "", time.Time{}, err
	}

	if _, err = tx.ExecContext(ctx, `
INSERT INTO auth_codes
(id,email,code_hash,purpose,attempts,max_attempts,ip_hash,user_agent,created_at,expires_at)
VALUES (?,?,?,'login',0,?,?,?,?,?)`,
		s.idFunc(), email, string(codeHash), codeMaxTry,
		s.ipHash(ipHash), userAgent, now.Format(time.RFC3339), expires.Format(time.RFC3339)); err != nil {
		return "", time.Time{}, err
	}
	if err = s.audit(ctx, tx, "email_code_issued", "", email, s.ipHash(ipHash), userAgent, "{}", now); err != nil {
		return "", time.Time{}, err
	}
	if err = tx.Commit(); err != nil {
		return "", time.Time{}, err
	}
	return code, expires, nil
}

func (s *Store) CompleteEmailLogin(ctx context.Context, rawEmail, code, ipHash, userAgent, locale string) (LoginResult, error) {
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		return LoginResult{}, err
	}
	now := s.nowFunc()
	tokenBytes := make([]byte, 32)
	if _, err = rand.Read(tokenBytes); err != nil {
		return LoginResult{}, err
	}
	token := hex.EncodeToString(tokenBytes)
	sessionID := s.idFunc()
	expires := now.Add(30 * 24 * time.Hour)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return LoginResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if err = s.userIsLoginable(ctx, tx, email); err != nil {
		return LoginResult{}, err
	}

	var storedHash string
	var attempts, maxTry int
	var codeExpires string
	var codeID string
	err = tx.QueryRowContext(ctx, `
SELECT id,code_hash,attempts,max_attempts,expires_at
FROM auth_codes
WHERE email=? AND used_at IS NULL
ORDER BY created_at DESC LIMIT 1`, email).Scan(&codeID, &storedHash, &attempts, &maxTry, &codeExpires)
	if errors.Is(err, sql.ErrNoRows) {
		return LoginResult{}, ErrCodeInvalid
	}
	if err != nil {
		return LoginResult{}, err
	}
	if now.After(mustTime(codeExpires)) {
		_, _ = tx.ExecContext(ctx, `UPDATE auth_codes SET used_at=? WHERE id=?`, now.Format(time.RFC3339), codeID)
		return LoginResult{}, ErrCodeExpired
	}
	if attempts >= maxTry {
		_, _ = tx.ExecContext(ctx, `UPDATE auth_codes SET used_at=? WHERE id=?`, now.Format(time.RFC3339), codeID)
		return LoginResult{}, ErrCodeLocked
	}
	if bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(code)) != nil {
		attempts++
		if _, err = tx.ExecContext(ctx, `
UPDATE auth_codes SET attempts=? WHERE id=?`, attempts, codeID); err != nil {
			return LoginResult{}, err
		}
		if err = tx.Commit(); err != nil {
			return LoginResult{}, err
		}
		return LoginResult{}, ErrCodeInvalid
	}

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
		return LoginResult{}, err
	default:
		newUserID = existingUserID
	}
	err = tx.QueryRowContext(ctx, `
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
		return LoginResult{}, err
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

	_, err = tx.ExecContext(ctx, `
INSERT INTO auth_identities (id,user_id,provider,provider_user_id,email,created_at,last_used_at)
VALUES (?,?,?,?,?,?,?)
ON CONFLICT(provider, provider_user_id) DO UPDATE SET last_used_at=excluded.last_used_at`,
		s.idFunc(), user.ID, string(ProviderEmail), email, email, nowText, nowText)
	if err != nil {
		return LoginResult{}, err
	}

	if _, err = tx.ExecContext(ctx, `
UPDATE auth_codes SET used_at=? WHERE id=?`, nowText, codeID); err != nil {
		return LoginResult{}, err
	}

	if _, err = tx.ExecContext(ctx, `
INSERT INTO auth_sessions
(id,user_id,token_hash,status,ip_hash,user_agent,created_at,expires_at,last_seen_at)
VALUES (?,?,?, 'active',?,?,?,? ,?)`,
		sessionID, user.ID, s.sessionTokenHash(token), s.ipHash(ipHash), userAgent,
		nowText, expires.Format(time.RFC3339), nowText); err != nil {
		return LoginResult{}, err
	}

	if err = s.audit(ctx, tx, "login_succeeded", user.ID, email, s.ipHash(ipHash), userAgent, "{}", now); err != nil {
		return LoginResult{}, err
	}

	session := AuthSession{
		ID: sessionID, UserID: user.ID, TokenHash: s.sessionTokenHash(token),
		Status: statusActive, IPHash: s.ipHash(ipHash), UserAgent: userAgent,
		ExpiresAt: expires, CreatedAt: now, LastSeenAt: &now,
	}
	if err = tx.Commit(); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{User: user, Session: session, SessionToken: token, IsNewUser: isNewUser}, nil
}

func scanUser(scan func(dest ...any) error) (User, error) {
	var user User
	var status, resultLocale, createdAt, updatedAt string
	var lastLogin, deleted sql.NullString
	err := scan(&user.ID, &user.Email, &status, &resultLocale, &createdAt, &updatedAt, &lastLogin, &deleted)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
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
	return user, nil
}

func mustTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		parsed = time.Time{}
	}
	return parsed
}

func (s *Store) GetUser(ctx context.Context, userID string) (User, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id,email,status,locale,created_at,updated_at,last_login_at,deleted_at
FROM users WHERE id=? AND status!='deleted'`, userID)
	return scanUser(row.Scan)
}

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
	var sessionRevoked, sessionLastSeen sql.NullString
	err := row.Scan(
		&user.ID, &user.Email, &userStatus, &userLocale, &userCreatedAt, &userUpdatedAt, &userLastLogin, &userDeleted,
		&session.ID, &session.TokenHash, &session.Status, &session.IPHash, &session.UserAgent,
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
		var revoked, lastSeen sql.NullString
		if err := rows.Scan(&session.ID, &session.UserID, &session.TokenHash, &session.Status, &session.IPHash, &session.UserAgent, &createdAt, &expiresAt, &revoked, &lastSeen); err != nil {
			return nil, err
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

func (s *Store) Cleanup(ctx context.Context) (int64, error) {
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
	_, err = s.db.ExecContext(ctx, `
DELETE FROM auth_codes WHERE expires_at < datetime('now','-30 days')`)
	return n, err
}
