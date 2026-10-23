package chat

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
			// 连续同名工具调用不重复发 phase（去重）。
			aguievents.NewToolCallStartEvent("call-2", "bazi_chart"),
			aguievents.NewToolCallEndEvent("call-2"),
			aguievents.NewToolCallStartEvent("call-3", "naming_generate"),
			aguievents.NewToolCallEndEvent("call-3"),
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
		UserID:              "user-1",
		SessionID:           "session-1",
		ConsultationContext: json.RawMessage(`{"version":1,"scenario":"divination","question":"這個項目能成嗎","casting":{"mode":"auto","order":"bottom_up","yaos":[7,8,8,7,8,9],"dong_yao":[6],"casting_id":"cast_1"},"casting_instant":"2026-10-23T12:00:00Z","location":{"longitude":121.4726,"timezone":"Asia/Shanghai"}}`),
		Profile: &AgentProfile{
			Gender: "male",
			Location: &AgentBirthLocation{
				Longitude: &[]float64{116.4074}[0],
				Timezone:  &[]string{"Asia/Shanghai"}[0],
			},
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
	})
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
	if !ok || profile["gender"] != "male" ||
		profile["location"].(map[string]any)["longitude"] != 116.4074 || profile["location"].(map[string]any)["timezone"] != "Asia/Shanghai" {
		t.Fatalf("profile ForwardedProps = %#v", props["profile"])
	}
	if len(request.Messages) != 3 {
		t.Fatalf("request message count = %d, want user/assistant/user only", len(request.Messages))
	}
	consultation, ok := props["consultation_context"].(map[string]any)
	if !ok || consultation["scenario"] != "divination" || consultation["question"] != "這個項目能成嗎" {
		t.Fatalf("consultation ForwardedProps = %#v", props["consultation_context"])
	}
	casting, ok := consultation["casting"].(map[string]any)
	if !ok || casting["casting_id"] != "cast_1" || casting["order"] != "bottom_up" {
		t.Fatalf("casting ForwardedProps = %#v", consultation["casting"])
	}
	for index, want := range []aguitypes.Role{aguitypes.RoleUser, aguitypes.RoleAssistant, aguitypes.RoleUser} {
		if request.Messages[index].Role != want {
			t.Fatalf("messages[%d].Role = %q, want %q", index, request.Messages[index].Role, want)
		}
	}
	if events[0].Type != agent.EventThinking {
		t.Fatalf("first event = %+v", events[0])
	}
	// phase 事件携带原始工具名（事实），文案由前端 i18n 渲染；连续同名去重。
	var phases []string
	for _, event := range events {
		if event.Type == agent.EventPhase {
			phases = append(phases, event.Content)
		}
	}
	if len(phases) != 2 || phases[0] != "bazi_chart" || phases[1] != "naming_generate" {
		t.Fatalf("phase events = %v, want [bazi_chart naming_generate]", phases)
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
	_, err := runner.Execute(ctx, []llm.Message{{Role: llm.RoleUser, Content: "hello"}}, nil)
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

func (r selectedRunner) Execute(context.Context, []llm.Message, func(agent.ChatEvent)) ([]llm.Message, error) {
	return nil, nil
}
func (r selectedRunner) Prompt() string { return r.name }

func TestServiceRoutesSessionsByExpert(t *testing.T) {
	natal := selectedRunner{name: "natal"}
	divination := selectedRunner{name: "divination"}
	service := NewService(nil, nil, nil)
	service.SetAgentRunners(map[string]Runner{
		ExpertNatal:      natal,
		ExpertDivination: divination,
	})

	if got, err := service.runnerForSession(ExpertNatal); err != nil || got != natal {
		t.Fatalf("runnerForSession(natal) = %#v, %v", got, err)
	}
	if got, err := service.runnerForSession(ExpertDivination); err != nil || got != divination {
		t.Fatalf("runnerForSession(divination) = %#v, %v", got, err)
	}
	// 无专家（四专家之前的极老会话）没有可路由的部署。
	if _, err := service.runnerForSession(""); err == nil {
		t.Fatal("expected ErrRunnerUnavailable for expert-less sessions")
	}
	// 未部署的专家明确不可用。
	if _, err := service.runnerForSession(ExpertFengshui); err == nil {
		t.Fatal("expected ErrRunnerUnavailable for an undeployed expert")
	}
}

func TestServiceRejectsWhenNoAgentRunnerConfigured(t *testing.T) {
	service := NewService(nil, nil, nil)
	if _, err := service.runnerForSession(ExpertNatal); err == nil {
		t.Fatal("expected ErrRunnerUnavailable when agent runner is not configured")
	}
	// 历史无专家会话明确不可用。
	if _, err := service.runnerForSession(""); err == nil {
		t.Fatal("expected ErrRunnerUnavailable for expert-less sessions")
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
