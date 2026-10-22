package auth

import (
	"context"
	"database/sql"

	"github.com/ml8s/liki-web/internal/migrate"
)

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
