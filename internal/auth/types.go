package auth

import (
	"time"
)

type UserStatus string

const (
	UserActive   UserStatus = "active"
	UserDisabled UserStatus = "disabled"
	UserDeleted  UserStatus = "deleted"
)

type User struct {
	ID          string
	Email       string
	Status      UserStatus
	Locale      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastLoginAt *time.Time
	DeletedAt   *time.Time
}

type AuthProvider string

const (
	ProviderEmail  AuthProvider = "email"
	ProviderGoogle AuthProvider = "google"
	ProviderApple  AuthProvider = "apple"
)

type AuthIdentity struct {
	ID          string
	UserID      string
	Provider    AuthProvider
	ProviderUID string
	Email       string
	CreatedAt   time.Time
	LastUsedAt  *time.Time
}

type AuthSession struct {
	ID         string
	UserID     string
	TokenHash  string
	Status     string
	IPHash     string
	UserAgent  string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
	LastSeenAt *time.Time
}

type PublicUser struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	Status      UserStatus `json:"status"`
	Locale      string     `json:"locale"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

type PublicAuthIdentity struct {
	ID         string     `json:"id"`
	Provider   string     `json:"provider"`
	Email      string     `json:"email"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	IsCurrent  bool       `json:"is_current"`
}

type PublicSession struct {
	ID         string     `json:"id"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	IPHash     string     `json:"ip_hash,omitempty"`
	UserAgent  string     `json:"user_agent,omitempty"`
	IsCurrent  bool       `json:"is_current"`
}
