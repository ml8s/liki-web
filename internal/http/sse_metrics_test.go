package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ml8s/liki-web/internal/httputil"
)

func TestMetricsWrapperPreservesSSEFlushing(t *testing.T) {
	var sawFlusher bool
	handler := Metrics(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := httputil.BeginSSE(w)
		sawFlusher = ok
		if ok {
			flusher.Flush()
		}
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/chat/sessions/session/ag-ui", nil))
	if !sawFlusher {
		t.Fatal("metrics wrapper removed http.Flusher")
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type = %q, want text/event-stream", got)
	}
}
