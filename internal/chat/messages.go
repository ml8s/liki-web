package chat

import (
	"context"
	"strings"
	"time"
)

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
UPDATE chat_sessions SET title='', profile_id='', context_json=NULL
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
