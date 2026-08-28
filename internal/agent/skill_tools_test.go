package agent

import (
	"context"
	"encoding/json"
	"testing"

	"liki-web/internal/llm"
)

// agent 消息循环 + 工具调用——使用 MockToolRegistry（零依赖 liki skill 文件）。
// skill 工具本身的测试在 liki 仓库 tests/test_skill_tools_e2e.py 中。

func mockToolRegistry() *MockToolRegistry {
	return &MockToolRegistry{
		Results: map[string]json.RawMessage{
			"query": json.RawMessage(`{"八字":[],"紫微":[]}`),
		},
	}
}

func TestAgentLoop_WithMockTool(t *testing.T) {
	registry := mockToolRegistry()
	llmMock := &MockLLM{
		ToolResps: []*llm.ChatResult{
			ChatRes([]llm.ToolCall{
				ToolCall("query", `{"rule":"marriage"}`),
			}, ""),
			ChatRes(nil, "查询完成"),
		},
	}
	a := NewAgent(llmMock, registry, "test prompt")
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "查询"}}
	result, err := a.Execute(context.Background(), msgs, nil, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}
	if result[2].Role != llm.RoleTool {
		t.Fatalf("msg[2] should be tool result, got %s", result[2].Role)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(result[2].Content), &out); err != nil {
		t.Fatalf("tool result 非 JSON: %v", err)
	}
	if result[3].Role != llm.RoleAssistant || result[3].Content != "查询完成" {
		t.Errorf("final = %+v, want assistant final", result[3])
	}
}

func TestAgentLoop_ErrorPropagation(t *testing.T) {
	registry := &MockToolRegistry{
		Errors: map[string]error{
			"query": &mockError{msg: "invalid parameters"},
		},
	}
	llmMock := &MockLLM{
		ToolResps: []*llm.ChatResult{
			ChatRes([]llm.ToolCall{
				ToolCall("query", `{"rule":"marriage"}`),
			}, ""),
			ChatRes(nil, "参数有误"),
		},
	}
	a := NewAgent(llmMock, registry, "")
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "查询"}}
	result, err := a.Execute(context.Background(), msgs, nil, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result[2].Role != llm.RoleTool {
		t.Fatalf("msg[2] should be tool result, got %s", result[2].Role)
	}
	if result[3].Role != llm.RoleAssistant || result[3].Content != "参数有误" {
		t.Errorf("final = %+v, want assistant final", result[3])
	}
}

type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }
