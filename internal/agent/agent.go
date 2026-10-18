package agent

// ChatEventType classifies ChatEvent kinds for client-side routing.
type ChatEventType string

// ChatEvent type constants for client-side SSE event routing.
const (
	EventTextDelta ChatEventType = "text-delta"
	EventPhase     ChatEventType = "phase"
	EventThinking  ChatEventType = "thinking"
	// EventThinkingDelta keeps the gateway's liki.thinking protocol mapping
	// covered even though the current runner emits whole thinking phases.
	EventThinkingDelta ChatEventType = "thinking-delta"
)

// ChatEvent is emitted during streaming chat and sent as SSE.
type ChatEvent struct {
	Type    ChatEventType `json:"type"`
	Content string        `json:"content,omitempty"`
	Data    any           `json:"data,omitempty"`
}
