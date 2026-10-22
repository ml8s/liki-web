package chat

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ml8s/liki-web/internal/migrate"
)

var ErrSessionNotFound = errors.New("liki: session not found")

// ErrInvalidMessageCursor 表示分页游标格式非法（由 HTTP 层转 400）。
var ErrInvalidMessageCursor = errors.New("chat: invalid message cursor")

type SessionStatus string

const (
	SessionActive SessionStatus = "active"
)

type Session struct {
	ID            string        `json:"id"`
	UserID        string        `json:"user_id"`
	ProfileID     string        `json:"profile_id"`
	Skill         string        `json:"skill"`
	Expert        string        `json:"expert"`
	Title         string        `json:"title"`
	Status        SessionStatus `json:"status"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	LastMessageAt *time.Time    `json:"last_message_at"`
}

type MessageKind string

const (
	KindText          MessageKind = "text"
	KindSystemProfile MessageKind = "system_profile"
)

type Message struct {
	ID        string      `json:"id"`
	SessionID string      `json:"session_id"`
	UserID    string      `json:"user_id"`
	Role      string      `json:"role"`
	Kind      MessageKind `json:"kind"`
	Content   string      `json:"content"`
	CreatedAt time.Time   `json:"created_at"`
}

// SendMessageResult is the canonical completion payload for Liki Chat. The
// AG-UI gateway places it in `RUN_FINISHED.result`.
type SendMessageResult struct {
	Session          Session `json:"session"`
	ActiveSkill      string  `json:"active_skill"`
	UserMessage      Message `json:"user_message"`
	AssistantMessage Message `json:"assistant_message"`
}

// ChatStore manages the historical chat_sessions and
// chat_messages tables. The table names are retained for schema
// compatibility; the product is named Liki Chat.
type ChatStore struct {
	db      *sql.DB
	nowFunc func() time.Time
	idFunc  func() string
}

func NewChatStore(db *sql.DB) *ChatStore {
	return &ChatStore{db: db, nowFunc: func() time.Time { return time.Now().UTC() }, idFunc: uuid.NewString}
}

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
	})
}

func (s *ChatStore) CreateSession(ctx context.Context, userID, skill, expert, title, profileID string) (Session, error) {
	now := chatFormatTime(s.nowFunc())
	session := Session{ID: s.idFunc(), UserID: userID, Skill: skill, Expert: expert, Title: title, ProfileID: profileID, Status: SessionActive}
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx, `
INSERT INTO chat_sessions (id,user_id,profile_id,skill,expert,title,status,created_at,updated_at,last_message_at)
VALUES (?,?,?,?,?,?,?,?,?,?)
RETURNING created_at,updated_at`,
		session.ID, userID, profileID, skill, expert, title, string(SessionActive), now, now, now,
	).Scan(&createdAt, &updatedAt)
	if err != nil {
		return Session{}, err
	}
	session.CreatedAt = chatMustTime(createdAt)
	session.UpdatedAt = chatMustTime(updatedAt)
	return session, nil
}

func scanSession(scan func(dest ...any) error) (Session, error) {
	var session Session
	var status, createdAt, updatedAt string
	var lastMessage sql.NullString
	err := scan(&session.ID, &session.UserID, &session.ProfileID, &session.Skill, &session.Expert, &session.Title, &status, &createdAt, &updatedAt, &lastMessage)
	if err != nil {
		return Session{}, err
	}
	session.Status = SessionStatus(status)
	session.CreatedAt = chatMustTime(createdAt)
	session.UpdatedAt = chatMustTime(updatedAt)
	if lastMessage.Valid {
		t := chatMustTime(lastMessage.String)
		session.LastMessageAt = &t
	}
	return session, nil
}

func (s *ChatStore) GetSession(ctx context.Context, userID, sessionID string) (Session, error) {
	session, err := scanSession(s.db.QueryRowContext(ctx, `
SELECT id,user_id,profile_id,skill,expert,title,status,created_at,updated_at,last_message_at
FROM chat_sessions WHERE id=? AND user_id=? AND deleted_at IS NULL`, sessionID, userID).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	return session, err
}

// sessionListLimit 是会话列表上限：UI 语义即「最近会话」，超限部分不进列表
// （将来若需全量历史再引入游标分页，届时复用 messages 的 keyset 约定）。
const sessionListLimit = 200

func (s *ChatStore) ListSessions(ctx context.Context, userID string) ([]Session, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id,user_id,profile_id,skill,expert,title,status,created_at,updated_at,last_message_at
FROM chat_sessions WHERE user_id=? AND status='active' AND deleted_at IS NULL
ORDER BY COALESCE(last_message_at,updated_at) DESC
LIMIT ?`, userID, sessionListLimit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]Session, 0)
	for rows.Next() {
		session, err := scanSession(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, session)
	}
	return out, rows.Err()
}

// RenameSession updates a live, user-owned session title. updated_at is
// intentionally untouched: renaming is metadata, not activity, and the list
// orders by last activity.
func (s *ChatStore) RenameSession(ctx context.Context, userID, sessionID, title string) (Session, error) {
	result, err := s.db.ExecContext(ctx, `
UPDATE chat_sessions SET title=? WHERE id=? AND user_id=? AND deleted_at IS NULL`, title, sessionID, userID)
	if err != nil {
		return Session{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Session{}, err
	}
	if affected == 0 {
		return Session{}, ErrSessionNotFound
	}
	return s.GetSession(ctx, userID, sessionID)
}

// SoftDeleteSession hides a user-owned session from every user-facing query.
// Messages stay in place for audit continuity; compliance erasure is handled
// separately by AnonymizeUserMessages.
func (s *ChatStore) SoftDeleteSession(ctx context.Context, userID, sessionID string) error {
	result, err := s.db.ExecContext(ctx, `
UPDATE chat_sessions SET deleted_at=? WHERE id=? AND user_id=? AND deleted_at IS NULL`,
		chatFormatTime(s.nowFunc()), sessionID, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrSessionNotFound
	}
	return nil
}

// 消息分页上限：默认页 200 条，服务端硬上限 500 条（防大会话全量加载）。
const (
	messagePageDefault = 200
	messagePageMax     = 500
)

// agentHistoryLimit 是喂给代理模型的历史消息条数上限（最新 N 条）。
const agentHistoryLimit = 50

// ListMessages 返回一页消息（按时间升序，便于直接渲染），并给出「更早一页」的游标。
// 游标格式 "<created_at>|<id>"（与 birthprofile 分页约定一致）；空游标 = 最新一页；
// nextCursor 非空表示还有更早的消息。
func (s *ChatStore) ListMessages(ctx context.Context, userID, sessionID, before string, limit int) ([]Message, string, error) {
	if limit <= 0 || limit > messagePageMax {
		limit = messagePageDefault
	}
	query := `
SELECT id,session_id,user_id,role,kind,content,created_at
FROM chat_messages
WHERE session_id=? AND user_id=?`
	args := []any{sessionID, userID}
	if before != "" {
		createdAt, id, ok := strings.Cut(before, "|")
		if !ok {
			return nil, "", ErrInvalidMessageCursor
		}
		query += ` AND (created_at < ? OR (created_at = ? AND id < ?))`
		args = append(args, createdAt, createdAt, id)
	}
	// 倒序取 limit+1 条：多出的一条用于判断是否还有更早消息。
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = rows.Close() }()
	desc := make([]Message, 0, limit+1)
	for rows.Next() {
		var message Message
		var kind, createdAt string
		if err := rows.Scan(&message.ID, &message.SessionID, &message.UserID, &message.Role, &kind, &message.Content, &createdAt); err != nil {
			return nil, "", err
		}
		message.Kind = MessageKind(kind)
		message.CreatedAt = chatMustTime(createdAt)
		desc = append(desc, message)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	nextCursor := ""
	if len(desc) > limit {
		desc = desc[:limit]
		oldest := desc[len(desc)-1]
		nextCursor = chatFormatTime(oldest.CreatedAt) + "|" + oldest.ID
	}
	// 反转为时间升序（渲染顺序）。
	out := make([]Message, len(desc))
	for i, message := range desc {
		out[len(desc)-1-i] = message
	}
	return out, nextCursor, nil
}

func (s *ChatStore) AppendMessage(ctx context.Context, userID, sessionID, role string, kind MessageKind, content string) (Message, error) {
	now := chatFormatTime(s.nowFunc())
	if kind == "" {
		kind = KindText
	}
	message := Message{ID: s.idFunc(), SessionID: sessionID, UserID: userID, Role: role, Kind: kind, Content: content}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var createdAt string
	err = tx.QueryRowContext(ctx, `
INSERT INTO chat_messages (id,session_id,user_id,role,kind,content,created_at)
VALUES (?,?,?,?,?,?,?)
RETURNING created_at`, message.ID, sessionID, userID, role, string(kind), content, now,
	).Scan(&createdAt)
	if err != nil {
		return Message{}, err
	}
	message.CreatedAt = chatMustTime(createdAt)
	if _, err := tx.ExecContext(ctx, `
UPDATE chat_sessions SET updated_at=?, last_message_at=CASE WHEN last_message_at IS NULL OR last_message_at<? THEN ? ELSE last_message_at END
WHERE id=?`, now, now, now, sessionID); err != nil {
		return Message{}, err
	}
	if err := tx.Commit(); err != nil {
		return Message{}, err
	}
	return message, nil
}

// PendingMessage is one message of an exchange written by AppendExchange.
type PendingMessage struct {
	Role    string
	Kind    MessageKind
	Content string
}

// AppendExchange writes an ordered batch of messages in one transaction. The
// run-completed callback compensates a failed write by deleting the run record
// and relying on retry replay, so a partially written exchange would duplicate
// the already-written messages on retry; the batch commits or rolls back whole.
func (s *ChatStore) AppendExchange(ctx context.Context, userID, sessionID string, messages ...PendingMessage) ([]Message, error) {
	if len(messages) == 0 {
		return nil, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	written := make([]Message, 0, len(messages))
	last := ""
	for _, pending := range messages {
		kind := pending.Kind
		if kind == "" {
			kind = KindText
		}
		// 每条消息取独立时间戳：列表按 created_at 排序，同刻消息的次序会
		// 退化到随机 id 上。
		now := chatFormatTime(s.nowFunc())
		message := Message{ID: s.idFunc(), SessionID: sessionID, UserID: userID, Role: pending.Role, Kind: kind, Content: pending.Content}
		var createdAt string
		err := tx.QueryRowContext(ctx, `
INSERT INTO chat_messages (id,session_id,user_id,role,kind,content,created_at)
VALUES (?,?,?,?,?,?,?)
RETURNING created_at`, message.ID, sessionID, userID, pending.Role, string(kind), pending.Content, now,
		).Scan(&createdAt)
		if err != nil {
			return nil, err
		}
		message.CreatedAt = chatMustTime(createdAt)
		written = append(written, message)
		last = now
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE chat_sessions SET updated_at=?, last_message_at=CASE WHEN last_message_at IS NULL OR last_message_at<? THEN ? ELSE last_message_at END
WHERE id=?`, last, last, last, sessionID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return written, nil
}

func (s *ChatStore) TouchSession(ctx context.Context, sessionID string, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE chat_sessions SET updated_at=? WHERE id=?`, chatFormatTime(now), sessionID)
	return err
}

// AnonymizeUserMessages clears the PII-bearing message content for a user's
// chat history during compliance erasure. Every message kind (user text,
// assistant text, and system_profile payloads) may embed personal data such as
// birth info or personal questions, so content is cleared across all kinds.
// Session titles are user-authored free text and profile_id links to the (now
// soft-deleted) profile, so both are cleared as well; message structure and
// metadata are preserved for audit continuity.
func (s *ChatStore) AnonymizeUserMessages(ctx context.Context, userID string) (int64, error) {
	if _, err := s.db.ExecContext(ctx, `
UPDATE chat_sessions SET title='', profile_id=''
WHERE user_id=?`, userID); err != nil {
		return 0, err
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE chat_messages SET content=''
WHERE user_id=?`, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func chatMustTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		parsed = time.Time{}
	}
	return parsed
}

// chatFormatTime uses fixed nanosecond precision so lexicographic UTC timestamp
// ordering remains deterministic even when messages are written in quick
// succession. Go's RFC3339Nano strips trailing zeroes and can break ordering.
func chatFormatTime(value time.Time) string {
	return value.Format("2006-01-02T15:04:05.000000000Z07:00")
}
