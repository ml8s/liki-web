package sse

import (
	"net/http/httptest"
	"testing"
)

func TestBegin(t *testing.T) {
	rec := httptest.NewRecorder()
	flusher, ok := Begin(rec)
	if !ok {
		t.Fatal("recorder should support flushing")
	}
	if flusher == nil {
		t.Fatal("flusher should be non-nil")
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("cache control = %q", got)
	}
}
