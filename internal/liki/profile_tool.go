package liki

import (
	"context"
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
	Calendar       string `json:"calendar"`
	Gender         string `json:"gender"`
	BirthDate      string `json:"birth_date"`
	BirthTime      string `json:"birth_time,omitempty"`
	BirthTimeKnown string `json:"birth_time_known"`
	BirthPlace     string `json:"birth_place,omitempty"`
}

func withSessionBinding(ctx context.Context, binding sessionBinding) context.Context {
	return context.WithValue(ctx, sessionBindingKey{}, binding)
}

func sessionBindingFrom(ctx context.Context) (sessionBinding, bool) {
	binding, ok := ctx.Value(sessionBindingKey{}).(sessionBinding)
	return binding, ok
}
