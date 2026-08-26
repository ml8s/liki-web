package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"liki-web/internal/llm"
)

// LLMClient is the subset of *llm.Client used by the agent. Mock in tests.
type LLMClient interface {
	ChatStreamWithTools(ctx context.Context, messages []llm.Message, tools []llm.ToolDef) (<-chan llm.StreamEvent, error)
	ChatStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error)
}

// ToolRegistry executes named tools and provides their LLM schemas.
type ToolRegistry interface {
	Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error)
	Schemas() []llm.ToolDef
}

// Agent handles LLM conversation via tool-calling. Engine computation is
// delegated to *engine.Service injected through the tool registry.
type Agent struct {
	llm    LLMClient
	tools  ToolRegistry
	prompt string
}

// NewAgent creates a new Agent with the given prompt and tools.
func NewAgent(llmClient LLMClient, tools ToolRegistry, prompt string) *Agent {
	return &Agent{llm: llmClient, tools: tools, prompt: prompt}
}

// ---------------------------------------------------------------------------
// Chat event types for SSE streaming
// ---------------------------------------------------------------------------

// ChatEventType classifies ChatEvent kinds for client-side routing.
type ChatEventType string

// ChatEvent type constants for client-side SSE event routing.
const (
	EventTextDelta     ChatEventType = "text-delta"
	EventPhase         ChatEventType = "phase"
	EventError         ChatEventType = "error"
	EventThinking      ChatEventType = "thinking"
	EventThinkingDelta ChatEventType = "thinking-delta"
)

// ChatEvent is emitted during streaming chat and sent as SSE.
type ChatEvent struct {
	Type    ChatEventType `json:"type"`
	Content string        `json:"content,omitempty"`
	Data    any           `json:"data,omitempty"`
}

func emitIf(fn func(ChatEvent), ev ChatEvent) {
	if fn != nil {
		fn(ev)
	}
}

const (
	chatRoundTimeout = 120 * time.Second
	maxChatRounds    = 20
)

// NamingChat runs the naming chat pipeline with tool-calling.
// onEvent receives SSE events for real-time client feedback; may be nil.
func (a *Agent) NamingChat(ctx context.Context, locale string, messages []llm.Message, onEvent func(ChatEvent)) ([]llm.Message, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("agent: naming chat: no messages")
	}
	emitIf(onEvent, ChatEvent{Type: EventThinking})

	msgs := a.ensureNamingPrompt(locale, messages)
	return a.Execute(ctx, msgs, onEvent, func() string { return phaseMsg(locale) })
}

// execute is the core tool-calling loop shared by all products.
// onEvent receives streaming text/thinking events (nil to skip).
// phaseMsg returns a per-round phase message (nil to skip).
func (a *Agent) Execute(ctx context.Context, msgs []llm.Message, onEvent func(ChatEvent), phaseMsg func() string) ([]llm.Message, error) {
	tools := a.tools.Schemas()

	for round := 0; round < maxChatRounds; round++ {
		roundCtx, roundCancel := context.WithTimeout(ctx, chatRoundTimeout)

		streamCh, err := a.llm.ChatStreamWithTools(roundCtx, msgs, tools)
		if err != nil {
			roundCancel()
			return nil, fmt.Errorf("agent: round %d: %w", round, err)
		}

		var contentBuf strings.Builder
		var reasoningBuf strings.Builder
		var finalToolCalls []llm.ToolCall

		for ev := range streamCh {
			if ev.Content != "" {
				contentBuf.WriteString(ev.Content)
				emitIf(onEvent, ChatEvent{Type: EventTextDelta, Content: ev.Content})
			}
			if ev.ReasoningContent != "" {
				reasoningBuf.WriteString(ev.ReasoningContent)
				emitIf(onEvent, ChatEvent{Type: EventThinkingDelta, Content: ev.ReasoningContent})
			}
			if ev.ToolCalls != nil {
				finalToolCalls = ev.ToolCalls
			}
		}

		assistantContent := contentBuf.String()
		if assistantContent != "" || len(finalToolCalls) > 0 {
			msgs = append(msgs, llm.Message{
				Role:             llm.RoleAssistant,
				Content:          assistantContent,
				ReasoningContent: reasoningBuf.String(),
				ToolCalls:        finalToolCalls,
			})
		}

		if len(finalToolCalls) == 0 {
			roundCancel()
			break
		}

		for _, tc := range finalToolCalls {
			result, err := a.tools.Execute(ctx, tc.Function.Name, json.RawMessage(tc.Function.Arguments))
			if err != nil {
				result = json.RawMessage(fmt.Sprintf(`{"error":%q}`, err.Error()))
			}
			msgs = append(msgs, llm.Message{
				Role:       llm.RoleTool,
				Content:    string(result),
				ToolCallID: tc.ID,
			})

			if phaseMsg != nil {
				emitIf(onEvent, ChatEvent{
					Type:    EventPhase,
					Content: phaseMsg(),
				})
			}
		}
		roundCancel()
	}

	return msgs, nil
}

func (a *Agent) ensureNamingPrompt(locale string, messages []llm.Message) []llm.Message {
	if len(messages) > 0 && messages[0].Role == llm.RoleSystem {
		return messages
	}
	prompt := a.prompt
	if locale == "en" {
		prompt += "\n\nAlways respond in English."
	}
	return append([]llm.Message{{Role: llm.RoleSystem, Content: prompt}}, messages...)
}

func phaseMsg(locale string) string {
	switch {
	case strings.HasPrefix(locale, "zh"):
		return "正在计算命理数据…"
	default:
		return "Computing natal chart data…"
	}
}
