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
)

// defaultSessionCap caps concurrent sessions per user when Service config
// does not override it.
const defaultSessionCap = 20

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`)

const (
	codeLength    = 6
	codeMaxTry    = 5
	statusActive  = "active"
	statusRevoked = "revoked"
)

type Store struct {
	db         *sql.DB
	hmacKey    []byte
	nowFunc    func() time.Time
	idFunc     func() string
	codeFun    func() (string, error)
	SessionCap int
}

func NewStore(db *sql.DB, hmacKey []byte) *Store {
	return &Store{
		db:         db,
		hmacKey:    hmacKey,
		SessionCap: defaultSessionCap,
		nowFunc:    func() time.Time { return time.Now().UTC() },
		idFunc:     func() string { return uuid.NewString() },
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

	// Enforce the per-user session cap: keep the newest SessionCap sessions
	// (the just-created one is always retained) and purge older ones along
	// with their ip_hash/user_agent metadata.
	if s.SessionCap > 0 {
		if _, err := tx.ExecContext(ctx, `
DELETE FROM auth_sessions
WHERE user_id=? AND id<>? AND id NOT IN (
	SELECT id FROM auth_sessions
	WHERE user_id=? AND id<>?
	ORDER BY created_at DESC, id DESC LIMIT ?
)`, user.ID, sessionID, user.ID, sessionID, s.SessionCap-1); err != nil {
			return LoginResult{}, err
		}
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

// GetUserByEmail resolves an active platform identity for privileged support
// workflows. Deleted users remain invisible and disabled users are returned so
// callers can distinguish "cannot compensate" from "does not exist".
func (s *Store) GetUserByEmail(ctx context.Context, rawEmail string) (User, error) {
	email := strings.ToLower(strings.TrimSpace(rawEmail))
	if email == "" {
		return User{}, ErrUserNotFound
	}
	row := s.db.QueryRowContext(ctx, `
SELECT id,email,status,locale,created_at,updated_at,last_login_at,deleted_at
FROM users WHERE email=? AND status!='deleted'`, email)
	return scanUser(row.Scan)
}
