package liki

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
	gateway.translate(agent.ChatEvent{Type: agent.EventPhase, Content: "排盘中"})
	gateway.translate(agent.ChatEvent{Type: agent.EventTextDelta, Content: "答案"})
	gateway.translate(agent.ChatEvent{Type: agent.EventThinkingDelta, Content: "hidden reasoning"})
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
		aguievents.EventTypeCustom,
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
	if finished == nil {
		t.Fatal("run finished event missing")
	}
	result, ok := finished.Result.(map[string]any)
	if !ok || result["active_skill"] != "liki" {
		t.Fatalf("finished result = %#v", finished.Result)
	}
}
