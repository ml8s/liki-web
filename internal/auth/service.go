package auth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Mailer interface {
	SendAuthCode(ctx context.Context, email, code string) error
}

type MailerFunc func(ctx context.Context, email, code string) error

func (fn MailerFunc) SendAuthCode(ctx context.Context, email, code string) error {
	return fn(ctx, email, code)
}

type Config struct {
	CodeTTL       time.Duration
	CodeCooldown  time.Duration
	CodeMaxTry    int
	SessionTTL    time.Duration
	CookieName    string
	SecureCookie  bool
	MaxSessions   int
	EmailResolver func(email string) (string, error)
}

func DefaultConfig() Config {
	return Config{
		CodeTTL:      10 * time.Minute,
		CodeCooldown: time.Minute,
		CodeMaxTry:   5,
		SessionTTL:   30 * 24 * time.Hour,
		CookieName:   "liki_session",
		SecureCookie: true,
		MaxSessions:  20,
		EmailResolver: func(email string) (string, error) {
			return NormalizeEmail(email)
		},
	}
}

func (c *Config) normalize() error {
	if c.CodeTTL <= 0 {
		c.CodeTTL = 10 * time.Minute
	}
	if c.CodeCooldown <= 0 {
		c.CodeCooldown = time.Minute
	}
	if c.CodeMaxTry <= 0 {
		c.CodeMaxTry = 5
	}
	if c.SessionTTL <= 0 {
		c.SessionTTL = 30 * 24 * time.Hour
	}
	if c.CookieName == "" {
		c.CookieName = "liki_session"
	}
	if c.MaxSessions <= 0 {
		c.MaxSessions = 20
	}
	if c.EmailResolver == nil {
		c.EmailResolver = func(email string) (string, error) { return NormalizeEmail(email) }
	}
	return nil
}

var emailDigits = regexp.MustCompile(`^[0-9]{6}$`)

type Service struct {
	store  *Store
	mailer Mailer
	config Config
}

func NewService(store *Store, mailer Mailer, config Config) (*Service, error) {
	if store == nil {
		return nil, errors.New("auth: store is required")
	}
	if mailer == nil {
		return nil, errors.New("auth: mailer is required")
	}
	if err := config.normalize(); err != nil {
		return nil, err
	}
	return &Service{store: store, mailer: mailer, config: config}, nil
}

type CodeIssued struct {
	Email     string
	ExpiresAt time.Time
}

func (s *Service) RequestEmailCode(ctx context.Context, rawEmail, ip, userAgent string) (CodeIssued, error) {
	email, err := s.config.EmailResolver(rawEmail)
	if err != nil {
		return CodeIssued{}, err
	}
	code, expires, err := s.store.IssueEmailCode(ctx, email, s.store.ipHash(ip), truncateUserAgent(userAgent))
	if err != nil {
		return CodeIssued{}, err
	}
	if err = s.mailer.SendAuthCode(ctx, email, code); err != nil {
		return CodeIssued{}, fmt.Errorf("auth: send email code: %w", err)
	}
	return CodeIssued{Email: email, ExpiresAt: expires}, nil
}

func (s *Service) VerifyEmailCode(ctx context.Context, rawEmail, rawCode, ip, userAgent string) (LoginResult, error) {
	email, err := s.config.EmailResolver(rawEmail)
	if err != nil {
		return LoginResult{}, err
	}
	code := strings.TrimSpace(rawCode)
	if !emailDigits.MatchString(code) {
		return LoginResult{}, ErrCodeInvalid
	}
	return s.store.CompleteEmailLogin(ctx, email, code, s.store.ipHash(ip), truncateUserAgent(userAgent), "zh-Hant")
}

func (s *Service) SessionCookieMaxAge() int {
	return int(s.config.SessionTTL.Seconds())
}

func truncateUserAgent(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	if len(value) > 512 {
		return value[:512]
	}
	return value
}
