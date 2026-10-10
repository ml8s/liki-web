package liki

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	aguisse "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"

	"github.com/ml8s/liki-web/internal/agent"
	"github.com/ml8s/liki-web/internal/llm"
)

func newTestAgentRunner(t *testing.T, server *httptest.Server) *AgentRunner {
	t.Helper()
	runner, err := NewAgentRunner(AgentRunnerConfig{
		BaseURL: server.URL,
		Token:   "internal-secret",
		Timeout: 1_000_000_000,
		Product: "liki-chat",
		Locale:  "zh-Hant",
	})
	if err != nil {
		t.Fatalf("NewAgentRunner() error = %v", err)
	}
	return runner
}

func TestAgentRunnerExecuteTranslatesAGUIAndUsesAuthoritativeAnswer(t *testing.T) {
	var request aguitypes.RunAgentInput
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer internal-secret" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("X-Liki-User-ID"); got != "user-1" {
			t.Errorf("X-Liki-User-ID = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		writer := aguisse.NewSSEWriter()
		events := []aguievents.Event{
			aguievents.NewRunStartedEvent("thread-1", "run-1"),
			aguievents.NewToolCallStartEvent("call-1", "bazi_chart"),
			aguievents.NewToolCallEndEvent("call-1"),
			aguievents.NewTextMessageContentEvent("message-1", "流式"),
			aguievents.NewTextMessageContentEvent("message-1", "内容"),
		}
		for _, event := range events {
			if err := writer.WriteEvent(r.Context(), w, event); err != nil {
				t.Errorf("write event: %v", err)
				return
			}
		}
		finished := aguievents.NewRunFinishedEvent("thread-1", "run-1")
		finished.Result = map[string]any{"answer": " 权威答案 ", "source_tools": []string{"bazi_chart"}}
		if err := writer.WriteEvent(r.Context(), w, finished); err != nil {
			t.Errorf("write finished: %v", err)
		}
	}))
	defer server.Close()

	runner := newTestAgentRunner(t, server)
	ctx := withSessionBinding(context.Background(), sessionBinding{
		UserID:    "user-1",
		SessionID: "session-1",
		Profile: &AgentProfileContext{
			Calendar:       "solar",
			Gender:         "male",
			BirthDate:      "1990-05-20",
			BirthTime:      "12:00",
			BirthTimeKnown: "known",
			BirthPlace:     "Beijing",
		},
	})
	input := []llm.Message{
		{Role: llm.RoleSystem, Content: "secret product prompt"},
		{Role: llm.RoleUser, Content: "earlier"},
		{Role: llm.RoleAssistant, Content: "previous answer"},
		{Role: llm.RoleSystem, Content: "profile payload"},
		{Role: llm.RoleUser, Content: "latest"},
	}
	events := make([]agent.ChatEvent, 0, 5)
	result, err := runner.Execute(ctx, input, func(event agent.ChatEvent) {
		events = append(events, event)
	}, func() string { return "using engine" })
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result) != 1 || result[0].Role != llm.RoleAssistant || result[0].Content != "权威答案" {
		t.Fatalf("result = %+v", result)
	}

	if request.ThreadID != "session-1" || request.RunID == "" {
		t.Fatalf("request IDs = %q/%q", request.ThreadID, request.RunID)
	}
	props, ok := request.ForwardedProps.(map[string]any)
	if !ok || props["product"] != "liki-chat" || props["locale"] != "zh-Hant" {
		t.Fatalf("ForwardedProps = %#v", request.ForwardedProps)
	}
	profile, ok := props["profile"].(map[string]any)
	if !ok || profile["calendar"] != "solar" || profile["gender"] != "male" ||
		profile["birth_date"] != "1990-05-20" || profile["birth_time"] != "12:00" ||
		profile["birth_time_known"] != "known" || profile["birth_place"] != "Beijing" {
		t.Fatalf("profile ForwardedProps = %#v", props["profile"])
	}
	if len(request.Messages) != 3 {
		t.Fatalf("request message count = %d, want user/assistant/user only", len(request.Messages))
	}
	for index, want := range []aguitypes.Role{aguitypes.RoleUser, aguitypes.RoleAssistant, aguitypes.RoleUser} {
		if request.Messages[index].Role != want {
			t.Fatalf("messages[%d].Role = %q, want %q", index, request.Messages[index].Role, want)
		}
	}
	if events[0].Type != agent.EventThinking {
		t.Fatalf("first event = %+v", events[0])
	}
	if events[1].Type != agent.EventPhase || events[1].Content != "正在排盘推演…" {
		t.Fatalf("tool event = %+v", events[1])
	}
	textEvents := make([]agent.ChatEvent, 0, 2)
	for _, event := range events {
		if event.Type == agent.EventTextDelta {
			textEvents = append(textEvents, event)
		}
	}
	if len(textEvents) != 2 || textEvents[0].Content != "流式" || textEvents[1].Content != "内容" {
		t.Fatalf("text events = %+v", textEvents)
	}
}

func TestAgentRunnerExecuteReturnsProtocolErrorWithoutSyntheticAnswer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		writer := aguisse.NewSSEWriter()
		_ = writer.WriteEvent(r.Context(), w, aguievents.NewRunStartedEvent("thread-1", "run-1"))
		_ = writer.WriteEvent(r.Context(), w, aguievents.NewRunErrorEvent("engine unavailable"))
	}))
	defer server.Close()

	runner := newTestAgentRunner(t, server)
	ctx := withSessionBinding(context.Background(), sessionBinding{UserID: "user-1", SessionID: "session-1"})
	_, err := runner.Execute(ctx, []llm.Message{{Role: llm.RoleUser, Content: "hello"}}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "engine unavailable") {
		t.Fatalf("Execute() error = %v, want engine unavailable", err)
	}
}

func TestAgentRunnerEndpointNormalizesBaseURL(t *testing.T) {
	tests := map[string]string{
		"http://agent:8083":          "http://agent:8083/ag-ui",
		"http://agent:8083/":         "http://agent:8083/ag-ui",
		"http://agent:8083/internal": "http://agent:8083/internal/ag-ui",
	}
	for base, want := range tests {
		if got := endpointFor(base); got != want {
			t.Fatalf("endpointFor(%q) = %q, want %q", base, got, want)
		}
	}
}

func TestAgentRunnerUsesRequestLocale(t *testing.T) {
	runner, err := NewAgentRunner(AgentRunnerConfig{
		BaseURL: "http://agent:8083",
		Product: "liki-chat",
		Locale:  "zh-Hant",
	})
	if err != nil {
		t.Fatal(err)
	}
	input, err := runner.newRunInput(
		[]llm.Message{{Role: llm.RoleUser, Content: "hello"}},
		sessionBinding{SessionID: "session-1"},
		"en",
	)
	if err != nil {
		t.Fatal(err)
	}
	props, ok := input.ForwardedProps.(map[string]any)
	if !ok || props["locale"] != "en" {
		t.Fatalf("ForwardedProps = %#v, want request locale en", input.ForwardedProps)
	}
}

func TestAgentRunnerDefaultsToProductFallbackLocale(t *testing.T) {
	runner, err := NewAgentRunner(AgentRunnerConfig{
		BaseURL: "http://agent:8083",
		Product: "liki-chat",
	})
	if err != nil {
		t.Fatal(err)
	}
	input, err := runner.newRunInput(
		[]llm.Message{{Role: llm.RoleUser, Content: "hello"}},
		sessionBinding{SessionID: "session-1"},
		runner.config.Locale,
	)
	if err != nil {
		t.Fatal(err)
	}
	props, ok := input.ForwardedProps.(map[string]any)
	if !ok || props["locale"] != "zh-Hant" {
		t.Fatalf("ForwardedProps = %#v, want fallback locale zh-Hant", input.ForwardedProps)
	}
}

type selectedRunner struct {
	name string
}

func (r selectedRunner) Execute(context.Context, []llm.Message, func(agent.ChatEvent), func() string) ([]llm.Message, error) {
	return nil, nil
}
func (r selectedRunner) Prompt() string { return r.name }

func TestServiceRoutesAllSessionsToAgentRunner(t *testing.T) {
	remote := selectedRunner{name: "remote"}
	service := NewService(nil, nil, nil)
	service.SetAgentRunner(remote)

	if got, err := service.runnerForSession(SkillLiki, ""); err != nil || got != remote {
		t.Fatalf("runnerForSession(unbound) = %#v, %v", got, err)
	}
	if got, err := service.runnerForSession(SkillLiki, "profile-1"); err != nil || got != remote {
		t.Fatalf("runnerForSession(profile) = %#v, %v", got, err)
	}
}

func TestServiceRejectsWhenNoAgentRunnerConfigured(t *testing.T) {
	service := NewService(nil, nil, nil)
	if _, err := service.runnerForSession(SkillLiki, ""); err == nil {
		t.Fatal("expected ErrRunnerUnavailable when agent runner is not configured")
	}
}

func TestLatestAGUIUserText(t *testing.T) {
	input := aguitypes.RunAgentInput{Messages: []aguitypes.Message{
		{ID: "1", Role: aguitypes.RoleUser, Content: "old"},
		{ID: "2", Role: aguitypes.RoleAssistant, Content: "answer"},
		{ID: "3", Role: aguitypes.RoleUser, Content: " latest "},
	}}
	content, err := latestAGUIUserText(input)
	if err != nil || content != "latest" {
		t.Fatalf("latestAGUIUserText() = %q, %v", content, err)
	}

	input.Messages = []aguitypes.Message{input.Messages[1]}
	if _, err := latestAGUIUserText(input); err == nil {
		t.Fatal("latestAGUIUserText() error = nil, want missing user message")
	}
}

func TestToolPhase(t *testing.T) {
	t.Parallel()
	cases := []struct{ tool, want string }{
		{"load_skill", "正在安排合适的老师…"},
		{"transfer_to_agent", "正在安排合适的老师…"},
		{"city_coords", "正在校准时空坐标…"},
		{"tianwen_time", "正在校准时空坐标…"},
		{"bazi_chart", "正在排盘推演…"},
		{"bazi_fullchart", "正在排盘推演…"},
		{"compute_factors", "正在排盘推演…"},
		{"period_query", "正在推演大运流年…"},
		{"period_query", "正在推演大运流年…"},
		{"qimen_cast", "正在起局…"},
		{"liuyao_chart", "正在起卦…"},
		{"naming_generate", "正在斟酌用字…"},
		{"fengshui_analyze", "正在勘察方位…"},
		{"unknown_tool", "正在计算命理数据…"},
		{"", "正在计算命理数据…"},
	}
	for _, tc := range cases {
		if got := toolPhase(tc.tool); got != tc.want {
			t.Errorf("toolPhase(%q) = %q, want %q", tc.tool, got, tc.want)
		}
	}
}
