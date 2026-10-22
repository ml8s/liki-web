package chat

import (
	"context"
	"time"
)

type sessionBindingKey struct{}

// sessionBinding carries the authenticated session identity into tool
// execution so a server-side tool can prove which profile it is allowed to
// touch. Set by Service.SendMessage before the runner runs.
type sessionBinding struct {
	UserID    string
	SessionID string
	ProfileID string
	Profile   *AgentProfileContext
}

// AgentProfileContext is the minimum birth facts required by a remote Agent to
// call deterministic chart tools. It deliberately excludes identity, contact,
// consent, notes, and database identifiers.
type AgentProfileContext struct {
	Gender         string     `json:"gender"`
	BirthLongitude *float64   `json:"birth_longitude,omitempty"`
	BirthTimezone  *string    `json:"birth_timezone,omitempty"`
	BirthInstant   *time.Time `json:"birth_instant,omitempty"`
}

func withSessionBinding(ctx context.Context, binding sessionBinding) context.Context {
	return context.WithValue(ctx, sessionBindingKey{}, binding)
}

func sessionBindingFrom(ctx context.Context) (sessionBinding, bool) {
	binding, ok := ctx.Value(sessionBindingKey{}).(sessionBinding)
	return binding, ok
}
