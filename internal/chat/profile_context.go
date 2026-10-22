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
	Profile   *AgentProfile
}

// AgentBirthLocation is the minimal computation location. Place is omitted so
// display text cannot become a hidden chart input.
type AgentBirthLocation struct {
	Longitude *float64 `json:"longitude,omitempty"`
	Timezone  *string  `json:"timezone,omitempty"`
}

// AgentProfile is the minimum birth facts required by a remote Agent to call
// deterministic chart tools. It deliberately excludes identity, contact,
// consent, notes, and database identifiers.
type AgentProfile struct {
	Gender   string              `json:"gender"`
	Instant  *time.Time          `json:"instant,omitempty"`
	Location *AgentBirthLocation `json:"location,omitempty"`
}

func withSessionBinding(ctx context.Context, binding sessionBinding) context.Context {
	return context.WithValue(ctx, sessionBindingKey{}, binding)
}

func sessionBindingFrom(ctx context.Context) (sessionBinding, bool) {
	binding, ok := ctx.Value(sessionBindingKey{}).(sessionBinding)
	return binding, ok
}
