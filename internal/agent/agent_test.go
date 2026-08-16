package agent

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"liki-web/internal/llm"
)

func TestNamingChat_EmptyMessages(t *testing.T) {
	t.Parallel()
	a := NewAgent(&MockLLM{}, &MockToolRegistry{}, "prompt")
	_, err := a.NamingChat(context.Background(), "zh-Hans", nil, nil)
	if err == nil || err.Error() != "agent: naming chat: no messages" {
		t.Errorf("expected 'no messages' error, got: %v", err)
	}
}

func TestNamingChat_SimpleResponse(t *testing.T) {
	t.Parallel()
	llmMock := &MockLLM{
		ToolResps: []*llm.ChatResult{
			ChatRes(nil, "根据您的八字，推荐名字：张三"),
		},
	}
	a := NewAgent(llmMock, &MockToolRegistry{}, "test prompt")

	var eventsMu sync.Mutex
	var receivedEvents []ChatEvent
	onEvent := func(ev ChatEvent) {
		eventsMu.Lock()
		receivedEvents = append(receivedEvents, ev)
		eventsMu.Unlock()
	}

	msgs := []llm.Message{{Role: llm.RoleUser, Content: "帮我起名"}}
	result, err := a.NamingChat(context.Background(), "zh-Hans", msgs, onEvent)
	if err != nil {
		t.Fatalf("NamingChat: %v", err)
	}

	// Should have system prompt + original message + assistant response
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}
	if result[0].Role != llm.RoleSystem {
		t.Errorf("msg[0].Role = %q, want system", result[0].Role)
	}
	if result[1].Role != llm.RoleUser || result[1].Content != "帮我起名" {
		t.Errorf("msg[1] = %+v, want user message", result[1])
	}
	if result[2].Role != llm.RoleAssistant || result[2].Content != "根据您的八字，推荐名字：张三" {
		t.Errorf("msg[2] = %+v, want assistant response", result[2])
	}

	// Events: thinking + text-delta
	eventsMu.Lock()
	if len(receivedEvents) < 2 {
		t.Fatalf("expected >=2 events, got %d", len(receivedEvents))
	}
	if receivedEvents[0].Type != EventThinking {
		t.Errorf("event[0].Type = %q, want thinking", receivedEvents[0].Type)
	}
	if receivedEvents[1].Type != EventTextDelta {
		t.Errorf("event[1].Type = %q, want text-delta", receivedEvents[1].Type)
	}
	eventsMu.Unlock()
}

func TestNamingChat_WithToolCallAndExecution(t *testing.T) {
	t.Parallel()
	llmMock := &MockLLM{
		ToolResps: []*llm.ChatResult{
			// Round 1: LLM calls a tool
			ChatRes([]llm.ToolCall{
				ToolCall("bazi.chart", `{"solar_time":"2026-01-01T12:00:00Z","gender":"male"}`),
			}, ""),
			// Round 2: LLM gives final answer
			ChatRes(nil, "排盘完成，日主为甲木，身强。"),
		},
	}
	toolReg := &MockToolRegistry{
		Results: map[string]json.RawMessage{
			"bazi.chart": json.RawMessage(`{"data":{"day_master":"甲","strength":"强"}}`),
		},
	}
	a := NewAgent(llmMock, toolReg, "test prompt")

	msgs := []llm.Message{{Role: llm.RoleUser, Content: "1990年5月20日"}}
	result, err := a.NamingChat(context.Background(), "zh-Hans", msgs, nil)
	if err != nil {
		t.Fatalf("NamingChat: %v", err)
	}

	// Should have: system + user + assistant(tool call) + tool(result) + assistant(final)
	if len(result) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(result))
	}
	// Verify tool call message
	if result[2].Role != llm.RoleAssistant || len(result[2].ToolCalls) != 1 {
		t.Fatalf("msg[2] should be assistant with 1 tool call, got %+v", result[2])
	}
	// Verify tool result message
	if result[3].Role != llm.RoleTool || result[3].Content != `{"data":{"day_master":"甲","strength":"强"}}` {
		t.Errorf("msg[3] = %+v, want tool result", result[3])
	}
	// Verify final response
	if result[4].Role != llm.RoleAssistant || result[4].Content != "排盘完成，日主为甲木，身强。" {
		t.Errorf("msg[4] = %+v, want final answer", result[4])
	}
}

func TestNamingChat_ToolExecutionError(t *testing.T) {
	t.Parallel()
	llmMock := &MockLLM{
		ToolResps: []*llm.ChatResult{
			ChatRes([]llm.ToolCall{
				ToolCall("bazi.chart", `{}`),
			}, ""),
			// Round 2: stop the loop
			ChatRes(nil, "error handled"),
		},
	}
	toolReg := &MockToolRegistry{
		Errors: map[string]error{
			"bazi.chart": errTestExec,
		},
	}
	a := NewAgent(llmMock, toolReg, "prompt")

	msgs := []llm.Message{{Role: llm.RoleUser, Content: "起名"}}
	result, err := a.NamingChat(context.Background(), "en", msgs, nil)
	if err != nil {
		t.Fatalf("NamingChat: %v", err)
	}

	// Tool error should be captured as a tool message with error JSON
	if len(result) < 4 {
		t.Fatalf("expected >=4 messages, got %d", len(result))
	}
	toolMsg := result[3]
	if toolMsg.Role != llm.RoleTool {
		t.Errorf("expected tool role, got %q", toolMsg.Role)
	}
}

func TestNamingChat_LLMError(t *testing.T) {
	t.Parallel()
	llmMock := &MockLLM{
		ToolErrs: []error{errTestLLM},
	}
	a := NewAgent(llmMock, &MockToolRegistry{}, "prompt")

	msgs := []llm.Message{{Role: llm.RoleUser, Content: "起名"}}
	_, err := a.NamingChat(context.Background(), "zh-Hans", msgs, nil)
	if err == nil || err.Error() != "agent: round 0: test llm error" {
		t.Errorf("expected LLM error, got: %v", err)
	}
}

func TestEnsureNamingPrompt_AddsSystemPrompt(t *testing.T) {
	t.Parallel()
	a := NewAgent(&MockLLM{}, &MockToolRegistry{}, "你好")
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "help"}}
	result := a.ensureNamingPrompt("en", msgs)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].Role != llm.RoleSystem || result[0].Content != "你好\n\nAlways respond in English." {
		t.Errorf("system prompt: %+v", result[0])
	}
	if result[1].Role != llm.RoleUser || result[1].Content != "help" {
		t.Errorf("user message preserved: %+v", result[1])
	}
}

func TestEnsureNamingPrompt_SkipsIfAlreadyHasSystem(t *testing.T) {
	t.Parallel()
	a := NewAgent(&MockLLM{}, &MockToolRegistry{}, "prompt")
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "existing"},
		{Role: llm.RoleUser, Content: "help"},
	}
	result := a.ensureNamingPrompt("en", msgs)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].Content != "existing" {
		t.Errorf("expected existing system prompt, got: %q", result[0].Content)
	}
}

func TestPhaseMsg_Chinese(t *testing.T) {
	t.Parallel()
	if msg := phaseMsg("zh-Hans"); msg != "正在计算命理数据…" {
		t.Errorf("zh-Hans: got %q", msg)
	}
	if msg := phaseMsg("zh-Hant"); msg != "正在计算命理数据…" {
		t.Errorf("zh-Hant: got %q", msg)
	}
}

func TestPhaseMsg_English(t *testing.T) {
	t.Parallel()
	if msg := phaseMsg("en"); msg != "Computing natal chart data…" {
		t.Errorf("en: got %q", msg)
	}
}

func TestPhaseMsg_Other(t *testing.T) {
	t.Parallel()
	if msg := phaseMsg("ja"); msg != "Computing natal chart data…" {
		t.Errorf("ja (fallback): got %q", msg)
	}
}

func TestEmitIf_NilCallback(t *testing.T) {
	t.Parallel()
	// Should not panic
	emitIf(nil, ChatEvent{Type: EventThinking})
}

// ── sentinel errors ──

var errTestLLM = errSentinel("test llm error")
var errTestExec = errSentinel("exec failed")

type errSentinel string

func (e errSentinel) Error() string { return string(e) }

func TestExecute_NoTools(t *testing.T) {
	t.Parallel()
	llmMock := &MockLLM{
		ToolResps: []*llm.ChatResult{ChatRes(nil, "好的")},
	}
	a := NewAgent(llmMock, &MockToolRegistry{}, "")
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "你好"}}
	result, err := a.Execute(context.Background(), msgs, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) < 2 {
		t.Fatalf("expected >= 2 messages, got %d", len(result))
	}
	last := result[len(result)-1]
	if last.Role != llm.RoleAssistant || last.Content != "好的" {
		t.Errorf("last message = %s: %q, want assistant: \"好的\"", last.Role, last.Content)
	}
}

func TestExecute_ToolLoop(t *testing.T) {
	t.Parallel()
	llmMock := &MockLLM{
		ToolResps: []*llm.ChatResult{
			ChatRes([]llm.ToolCall{
				{ID: "call-1", Type: "function", Function: llm.FunctionCall{Name: "echo", Arguments: `{"msg":"hello"}`}},
			}, ""),
			ChatRes(nil, "done"),
		},
	}
	a := NewAgent(llmMock, &MockToolRegistry{}, "")
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "test"}}
	result, err := a.Execute(context.Background(), msgs, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) < 4 {
		t.Fatalf("expected >= 4 messages (user+assistant+tool+assistant), got %d", len(result))
	}
	last := result[len(result)-1]
	if last.Role != llm.RoleAssistant || last.Content != "done" {
		t.Errorf("last = %s: %q, want assistant: \"done\"", last.Role, last.Content)
	}
}

func TestExecute_StreamEvents(t *testing.T) {
	t.Parallel()
	llmMock := &MockLLM{
		ToolResps: []*llm.ChatResult{ChatRes(nil, "hello")},
	}
	a := NewAgent(llmMock, &MockToolRegistry{}, "")
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "test"}}

	var events []ChatEvent
	_, err := a.Execute(context.Background(), msgs, func(ev ChatEvent) {
		events = append(events, ev)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ev := range events {
		if ev.Type == "text-delta" && ev.Content == "hello" {
			found = true
		}
	}
	if !found {
		t.Error("expected text-delta event with content 'hello'")
	}
}
