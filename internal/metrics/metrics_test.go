package metrics_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"liki-web/internal/metrics"
)

func TestHandlerRequiresToken(t *testing.T) {
	handler := metrics.Handler("")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("empty token = %d, want 404", rec.Code)
	}
}

func TestHandlerReturnsCounters(t *testing.T) {
	metrics.Default.Inc("test_total", map[string]string{"result": "ok"})
	handler := metrics.Handler("secret")
	req := httptest.NewRequest(http.MethodGet, "/metrics?token=secret", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Counters map[string]uint64 `json:"counters"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Counters["test_total|result=ok"] != 1 {
		t.Fatalf("counters = %#v", body.Counters)
	}
}
