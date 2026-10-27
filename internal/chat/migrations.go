package chat

import (
	"context"
	"database/sql"

	"github.com/ml8s/liki-web/internal/migrate"
)

func (s *ChatStore) Migrate(ctx context.Context) error {
	return migrate.Run(ctx, s.db, []migrate.Migration{
		{
			// The unified_* table names keep their prefix from the
			// session-merge migration; they denote the one and only chat
			// model. Renaming is cosmetic and deferred.
			ID: "001_unified_chat_runtime",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS unified_chat_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    skill TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','closed')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    last_message_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_unified_chat_sessions_user_updated
    ON unified_chat_sessions(user_id, status, updated_at);

CREATE TABLE IF NOT EXISTS unified_chat_messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES unified_chat_sessions(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('user','assistant')),
    content TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_unified_chat_messages_session_created
    ON unified_chat_messages(session_id, created_at);
`)
				return err
			},
		},
		{
			// Platform binding: sessions may carry a profile (user info), and
			// messages carry a kind so system/chart messages render distinctly.
			ID: "002_profile_binding",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				if _, err := tx.ExecContext(ctx, `
ALTER TABLE unified_chat_sessions ADD COLUMN profile_id TEXT NOT NULL DEFAULT '';
ALTER TABLE unified_chat_messages ADD COLUMN kind TEXT NOT NULL DEFAULT 'text';
`); err != nil {
					return err
				}
				_, err := tx.ExecContext(ctx, `
CREATE INDEX IF NOT EXISTS idx_unified_chat_sessions_user_profile
    ON unified_chat_sessions(user_id, profile_id);
`)
				return err
			},
		},
		{
			ID: "003_canonical_chat_storage",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
ALTER TABLE unified_chat_sessions RENAME TO chat_sessions;
ALTER TABLE unified_chat_messages RENAME TO chat_messages;
DROP INDEX IF EXISTS idx_unified_chat_sessions_user_updated;
DROP INDEX IF EXISTS idx_unified_chat_sessions_user_profile;
DROP INDEX IF EXISTS idx_unified_chat_messages_session_created;
CREATE INDEX idx_chat_sessions_user_updated
    ON chat_sessions(user_id, status, updated_at);
CREATE INDEX idx_chat_sessions_user_profile
    ON chat_sessions(user_id, profile_id);
CREATE INDEX idx_chat_messages_session_created
    ON chat_messages(session_id, created_at);
`)
				return err
			},
		},
		{
			// User-initiated deletion keeps rows for audit continuity; every
			// user-facing query filters deleted_at IS NULL.
			ID: "004_session_soft_delete",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
ALTER TABLE chat_sessions ADD COLUMN deleted_at TEXT;
`)
				return err
			},
		},
		{
			// 四专家独立部署：会话绑定专家（endpoint 选择维度）。历史会话
			// expert=''（四专家之前的极老会话）没有可路由的部署，明确不可用。
			ID: "005_session_expert",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
ALTER TABLE chat_sessions ADD COLUMN expert TEXT NOT NULL DEFAULT '';
`)
				return err
			},
		},
		{
			// 列表查询约定：按用户列表按「最近活动」排序且必须能走索引。
			// 表达式索引与查询的 ORDER BY 表达式逐字一致（EXPLAIN 验收无临时排序）。
			ID: "chat_006_sessions_recent_index",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_recent
    ON chat_sessions(user_id, COALESCE(last_message_at, updated_at) DESC);
`)
				return err
			},
		},
		{
			// keyset 分页索引必须覆盖完整排序键 (created_at, id)：
			// 旧索引缺 id 列，id 并列时走临时排序（EXPLAIN 验收）。
			ID: "chat_007_messages_keyset_index",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
DROP INDEX IF EXISTS idx_chat_messages_session_created;
CREATE INDEX IF NOT EXISTS idx_chat_messages_session_created
    ON chat_messages(session_id, created_at, id);
`)
				return err
			},
		},
		{
			// Tool-to-Chat continuity: deterministic/non-reproducible consultation
			// facts live in their own required-value table. A session without a
			// tool context is represented by the absence of a row, not NULL.
			ID: "chat_008_consultation_context",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
CREATE TABLE chat_consultation_contexts (
    session_id TEXT PRIMARY KEY REFERENCES chat_sessions(id) ON DELETE CASCADE,
    context_json TEXT NOT NULL CHECK (
        json_valid(context_json) AND json_type(context_json) = 'object'
    )
);
`)
				return err
			},
		},
		{
			// 四专家迁移前的会话没有可路由的 expert。它们无法打开，也不应
			// 出现在「最近会话」里；保留行与消息，状态改为 closed。
			ID: "chat_009_close_legacy_sessions",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
UPDATE chat_sessions
SET status='closed'
WHERE expert='' AND status='active' AND deleted_at IS NULL
`)
				return err
			},
		},
		{
			// Runtime callbacks and the direct browser transport are retired.
			// Product messages are authoritative, so this legacy callback table
			// is removed rather than kept as a hidden second source of truth.
			ID: "chat_010_drop_legacy_run_records",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS run_records`)
				return err
			},
		},
	})
}
