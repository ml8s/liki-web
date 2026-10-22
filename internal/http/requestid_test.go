package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ml8s/liki-web/internal/httputil"
)

func TestRequestIDGeneratesAndPropagates(t *testing.T) {
	var seen string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = httputil.RequestIDFrom(r.Context())
	})
	handler := RequestID(next)

	// 无上游 ID：生成且响应头可见
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/chat/sessions", nil))
	if seen == "" || rec.Header().Get("X-Request-Id") != seen {
		t.Fatalf("generated id mismatch: ctx=%q header=%q", seen, rec.Header().Get("X-Request-Id"))
	}

	// 合法上游 ID：透传
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/chat/sessions", nil)
	req.Header.Set("X-Request-Id", "trace-abc-123")
	handler.ServeHTTP(rec, req)
	if seen != "trace-abc-123" || rec.Header().Get("X-Request-Id") != "trace-abc-123" {
		t.Fatalf("propagated id mismatch: ctx=%q", seen)
	}

	// 非法上游 ID（注入字符/超长）：丢弃并生成新 ID
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/chat/sessions", nil)
	req.Header.Set("X-Request-Id", "bad id\r\ninjected: 1")
	handler.ServeHTTP(rec, req)
	if seen == "bad id\r\ninjected: 1" || seen == "" {
		t.Fatalf("invalid id must be replaced, got %q", seen)
	}
}
