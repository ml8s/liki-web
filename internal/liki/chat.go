package liki

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/ml8s/liki-web/internal/migrate"
)

var ErrSessionNotFound = errors.New("liki: session not found")

type SessionStatus string

const (
	SessionActive SessionStatus = "active"
	SessionClosed SessionStatus = "closed"
)

type Session struct {
	ID            string        `json:"id"`
	UserID        string        `json:"user_id"`
	ProfileID     string        `json:"profile_id"`
	Skill         string        `json:"skill"`
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
// legacy SSE endpoint places it in `done.data`; the AG-UI gateway places it in
// `RUN_FINISHED.result`.
type SendMessageResult struct {
	Session          Session `json:"session"`
	ActiveSkill      string  `json:"active_skill"`
	UserMessage      Message `json:"user_message"`
	AssistantMessage Message `json:"assistant_message"`
}

// ChatStore manages the historical unified_chat_sessions and
// unified_chat_messages tables. The table names are retained for schema
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
	})
}

func (s *ChatStore) CreateSession(ctx context.Context, userID, skill, title, profileID string) (Session, error) {
	now := chatFormatTime(s.nowFunc())
	session := Session{ID: s.idFunc(), UserID: userID, Skill: skill, Title: title, ProfileID: profileID, Status: SessionActive}
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx, `
INSERT INTO unified_chat_sessions (id,user_id,profile_id,skill,title,status,created_at,updated_at,last_message_at)
VALUES (?,?,?,?,?,?,?,?,?)
RETURNING created_at,updated_at`,
		session.ID, userID, profileID, skill, title, string(SessionActive), now, now, now,
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
	err := scan(&session.ID, &session.UserID, &session.ProfileID, &session.Skill, &session.Title, &status, &createdAt, &updatedAt, &lastMessage)
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
SELECT id,user_id,profile_id,skill,title,status,created_at,updated_at,last_message_at
FROM unified_chat_sessions WHERE id=? AND user_id=?`, sessionID, userID).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	return session, err
}

func (s *ChatStore) ListSessions(ctx context.Context, userID string) ([]Session, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id,user_id,profile_id,skill,title,status,created_at,updated_at,last_message_at
FROM unified_chat_sessions WHERE user_id=? AND status='active'
ORDER BY COALESCE(last_message_at,updated_at) DESC`, userID)
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

func (s *ChatStore) ListMessages(ctx context.Context, userID, sessionID string) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT m.id,m.session_id,m.user_id,m.role,m.kind,m.content,m.created_at
FROM unified_chat_messages m
JOIN unified_chat_sessions s ON s.id=m.session_id
WHERE m.session_id=? AND m.user_id=?
ORDER BY m.created_at, m.id`, sessionID, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]Message, 0)
	for rows.Next() {
		var message Message
		var kind, createdAt string
		if err := rows.Scan(&message.ID, &message.SessionID, &message.UserID, &message.Role, &kind, &message.Content, &createdAt); err != nil {
			return nil, err
		}
		message.Kind = MessageKind(kind)
		message.CreatedAt = chatMustTime(createdAt)
		out = append(out, message)
	}
	return out, rows.Err()
}

func (s *ChatStore) AppendMessage(ctx context.Context, userID, sessionID, role string, kind MessageKind, content string) (Message, error) {
	now := chatFormatTime(s.nowFunc())
	if kind == "" {
		kind = KindText
	}
	message := Message{ID: s.idFunc(), SessionID: sessionID, UserID: userID, Role: role, Kind: kind, Content: content}
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
INSERT INTO unified_chat_messages (id,session_id,user_id,role,kind,content,created_at)
VALUES (?,?,?,?,?,?,?)
RETURNING created_at`, message.ID, sessionID, userID, role, string(kind), content, now,
	).Scan(&createdAt)
	if err != nil {
		return Message{}, err
	}
	message.CreatedAt = chatMustTime(createdAt)
	_, err = s.db.ExecContext(ctx, `
UPDATE unified_chat_sessions SET updated_at=?, last_message_at=CASE WHEN last_message_at IS NULL OR last_message_at<? THEN ? ELSE last_message_at END
WHERE id=?`, now, now, now, sessionID)
	return message, err
}

func (s *ChatStore) TouchSession(ctx context.Context, sessionID string, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE unified_chat_sessions SET updated_at=? WHERE id=?`, chatFormatTime(now), sessionID)
	return err
}

// AnonymizeUserMessages clears the PII-bearing message content for a user's
// chat history during compliance erasure. Every message kind (user text,
// assistant text, and system_profile payloads) may embed personal data such as
// birth info or personal questions, so content is cleared across all kinds;
// message structure and metadata are preserved for audit continuity.
func (s *ChatStore) AnonymizeUserMessages(ctx context.Context, userID string) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
UPDATE unified_chat_messages SET content=''
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
