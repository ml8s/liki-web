package chat

import "time"

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
