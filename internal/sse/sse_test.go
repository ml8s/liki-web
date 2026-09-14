package sse

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBeginAndWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	flusher, ok := Begin(rec)
	if !ok {
		t.Fatal("recorder should support flushing")
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type = %q", got)
	}

	event := map[string]any{"type": "text-delta", "content": "你好"}
	if err := Write(rec, flusher, event); err != nil {
		t.Fatal(err)
	}

	want := "data: " + string(mustJSON(event)) + "\n\n"
	if rec.Body.String() != want {
		t.Fatalf("body = %q, want %q", rec.Body.String(), want)
	}
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(data))
}
