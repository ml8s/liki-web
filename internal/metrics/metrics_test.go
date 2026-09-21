package metrics_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestRenderPrometheus(t *testing.T) {
	rendered := metrics.RenderPrometheus(map[string]uint64{
		"payment_refunded_total|provider=dodo":      2,
		"http_requests_total|path=/api/|status=200": 3,
	})
	for _, want := range []string{
		"# TYPE payment_refunded_total counter",
		`payment_refunded_total{provider="dodo"} 2`,
		`http_requests_total{path="/api/",status="200"} 3`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered=%q want=%q", rendered, want)
		}
	}
}

func TestHandlerPrometheusFormat(t *testing.T) {
	metrics.Default.Inc("test_prom_total", map[string]string{"result": "ok"})
	handler := metrics.Handler("secret")
	req := httptest.NewRequest(http.MethodGet, "/metrics?token=secret&format=prometheus", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/plain") {
		t.Fatalf("content type = %q", got)
	}
	if !strings.Contains(rec.Body.String(), `test_prom_total{result="ok"} 1`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
