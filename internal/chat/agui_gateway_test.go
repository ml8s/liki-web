package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"

	"github.com/ml8s/liki-web/internal/agent"
)

func TestAGUIGatewayEmitsStandardBrowserProtocol(t *testing.T) {
	recorder := httptest.NewRecorder()
	recorder.Header().Set("Content-Type", "text/event-stream")
	gateway := newAGUIGateway(recorder, context.Background(), "session-1", "run-1")

	gateway.start()
	gateway.translate(agent.ChatEvent{Type: agent.EventPhase, Content: "bazi_chart"})
	gateway.translate(agent.ChatEvent{Type: agent.EventTextDelta, Content: "答案"})
	gateway.translate(agent.ChatEvent{Type: agent.EventThinking, Content: "hidden reasoning"})
	gateway.finish(SendMessageResult{
		Session:          Session{ID: "session-1"},
		ActiveSkill:      "liki",
		UserMessage:      Message{ID: "user-1"},
		AssistantMessage: Message{ID: "assistant-1", Content: "答案"},
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type = %q", got)
	}

	decoder := aguievents.NewEventDecoder(nil)
	types := make([]aguievents.EventType, 0, 8)
	customs := make([]struct {
		name  string
		value any
	}, 0, 2)
	var finished *aguievents.RunFinishedEvent
	scanner := bufio.NewScanner(strings.NewReader(recorder.Body.String()))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var envelope struct {
			Type aguievents.EventType `json:"type"`
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &envelope); err != nil {
			t.Fatalf("decode envelope: %v", err)
		}
		event, err := decoder.DecodeEvent(string(envelope.Type), []byte(strings.TrimPrefix(line, "data: ")))
		if err != nil {
			t.Fatalf("decode AG-UI event: %v", err)
		}
		types = append(types, event.Type())
		if custom, ok := event.(*aguievents.CustomEvent); ok {
			customs = append(customs, struct {
				name  string
				value any
			}{name: custom.Name, value: custom.Value})
		}
		if ev, ok := event.(*aguievents.RunFinishedEvent); ok {
			finished = ev
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan SSE: %v", err)
	}

	want := []aguievents.EventType{
		aguievents.EventTypeRunStarted,
		aguievents.EventTypeCustom,
		aguievents.EventTypeTextMessageStart,
		aguievents.EventTypeTextMessageContent,
		aguievents.EventTypeTextMessageEnd,
		aguievents.EventTypeRunFinished,
	}
	if len(types) != len(want) {
		t.Fatalf("events = %v, want %v", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("events = %v, want %v", types, want)
		}
	}
	// 自定义事件使用反域名命名；phase 载荷是原始工具名（事实）。
	if len(customs) != 1 ||
		customs[0].name != "hk.liki.phase" || customs[0].value != "bazi_chart" {
		t.Fatalf("phase custom event = %#v", customs)
	}
	if strings.Contains(recorder.Body.String(), "hidden reasoning") {
		t.Fatal("reasoning content must never be forwarded to the browser")
	}
	if finished == nil {
		t.Fatal("run finished event missing")
	}
	result, ok := finished.Result.(map[string]any)
	if !ok || result["active_skill"] != "liki" {
		t.Fatalf("finished result = %#v", finished.Result)
	}
}
