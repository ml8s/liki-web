package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBeginSSE(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	flusher, ok := BeginSSE(rec)
	if !ok {
		t.Fatal("recorder should support flushing")
	}
	if flusher == nil {
		t.Fatal("flusher should not be nil")
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type = %q, want text/event-stream", got)
	}
}

func TestBeginSSERejectsNonFlusher(t *testing.T) {
	t.Parallel()
	_, ok := BeginSSE(nonFlusherWriter{})
	if ok {
		t.Fatal("non-flusher writer should be rejected")
	}
}

type nonFlusherWriter struct{}

func (nonFlusherWriter) Header() http.Header       { return http.Header{} }
func (nonFlusherWriter) Write([]byte) (int, error) { return 0, nil }
func (nonFlusherWriter) WriteHeader(int)           {}
