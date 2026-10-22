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
)

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
	config     Config
	nowFunc    func() time.Time
	idFunc     func() string
	codeFun    func() (string, error)
	SessionCap int
}

func NewStore(db *sql.DB, hmacKey []byte) *Store {
	config := DefaultConfig()
	return &Store{
		db:         db,
		hmacKey:    hmacKey,
		config:     config,
		SessionCap: config.MaxSessions,
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

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

type rowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Store) ipHash(ip string) string {
	if ip == "" {
		return ""
	}
	sum := hmac.New(sha256.New, s.hmacKey)
	sum.Write([]byte("auth_ip:" + ip))
	return hex.EncodeToString(sum.Sum(nil))
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
