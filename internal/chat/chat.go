package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrSessionNotFound = errors.New("liki: session not found")

// ErrInvalidMessageCursor 表示分页游标格式非法（由 HTTP 层转 400）。
var ErrInvalidMessageCursor = errors.New("chat: invalid message cursor")

type SessionStatus string

const (
	SessionActive SessionStatus = "active"
)

type Session struct {
	ID            string          `json:"id"`
	UserID        string          `json:"user_id"`
	ProfileID     string          `json:"profile_id"`
	Skill         string          `json:"skill"`
	Expert        string          `json:"expert"`
	Title         string          `json:"title"`
	Status        SessionStatus   `json:"status"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	LastMessageAt *time.Time      `json:"last_message_at"`
	Context       json.RawMessage `json:"context,omitempty"`
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

func (s *ChatStore) CreateSession(ctx context.Context, userID, skill, expert, title, profileID string) (Session, error) {
	return s.CreateSessionWithContext(ctx, userID, skill, expert, title, profileID, nil)
}

func (s *ChatStore) CreateSessionWithContext(
	ctx context.Context,
	userID, skill, expert, title, profileID string,
	consultationContext json.RawMessage,
) (Session, error) {
	now := chatFormatTime(s.nowFunc())
	session := Session{
		ID: s.idFunc(), UserID: userID, Skill: skill, Expert: expert,
		Title: title, ProfileID: profileID, Status: SessionActive,
		Context: consultationContext,
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var createdAt, updatedAt string
	err = tx.QueryRowContext(ctx, `
INSERT INTO chat_sessions (id,user_id,profile_id,skill,expert,title,status,created_at,updated_at,last_message_at)
VALUES (?,?,?,?,?,?,?,?,?,?)
RETURNING created_at,updated_at`,
		session.ID, userID, profileID, skill, expert, title, string(SessionActive), now, now, now,
	).Scan(&createdAt, &updatedAt)
	if err != nil {
		return Session{}, err
	}
	if len(session.Context) != 0 {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO chat_consultation_contexts (session_id,context_json) VALUES (?,?)`,
			session.ID, string(session.Context)); err != nil {
			return Session{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Session{}, err
	}
	session.CreatedAt = chatMustTime(createdAt)
	session.UpdatedAt = chatMustTime(updatedAt)
	return session, nil
}

func scanSession(scan func(dest ...any) error, consultationContext *sql.NullString) (Session, error) {
	var session Session
	var status, createdAt, updatedAt string
	var lastMessage sql.NullString
	dest := []any{
		&session.ID, &session.UserID, &session.ProfileID, &session.Skill, &session.Expert,
		&session.Title, &status, &createdAt, &updatedAt, &lastMessage,
	}
	if consultationContext != nil {
		dest = append(dest, consultationContext)
	}
	err := scan(dest...)
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
	if consultationContext != nil && consultationContext.Valid {
		session.Context = json.RawMessage(consultationContext.String)
	}
	return session, nil
}

func (s *ChatStore) GetSession(ctx context.Context, userID, sessionID string) (Session, error) {
	var consultationContext sql.NullString
	session, err := scanSession(s.db.QueryRowContext(ctx, `
SELECT
    s.id,s.user_id,s.profile_id,s.skill,s.expert,s.title,s.status,s.created_at,s.updated_at,s.last_message_at,
    c.context_json
FROM chat_sessions AS s
LEFT JOIN chat_consultation_contexts AS c ON c.session_id=s.id
WHERE s.id=? AND s.user_id=? AND s.deleted_at IS NULL`, sessionID, userID).Scan, &consultationContext)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, err
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
		session, err := scanSession(rows.Scan, nil)
		if err != nil {
			return nil, err
		}
		// Session lists are collection projections; consultation context is
		// returned only by the owner/entitlement-checked detail endpoint.
		session.Context = nil
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
