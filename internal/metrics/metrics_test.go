package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/metrics"
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

func TestHandlerRejectsIncorrectBearerToken(t *testing.T) {
	handler := metrics.Handler("secret")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer not-secret")
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if got := rec.Header().Get("WWW-Authenticate"); got != "Bearer" {
		t.Fatalf("www-authenticate = %q, want Bearer", got)
	}
}

func TestHandlerReturnsPrometheusExposition(t *testing.T) {
	metrics.Default.Inc("test_total", map[string]string{"result": "ok"})
	metrics.Default.ObserveDurationSeconds(
		"test_duration_seconds_total", map[string]string{"result": "ok"}, 0.125,
	)
	handler := metrics.Handler("secret")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/plain; version=0.0.4; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
	for _, want := range []string{
		"# TYPE test_total counter\ntest_total{result=\"ok\"} 1\n",
		"# TYPE test_duration_seconds_total counter\ntest_duration_seconds_total{result=\"ok\"} 0.125000\n",
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("body missing %q:\n%s", want, rec.Body.String())
		}
	}
}

func TestRenderPrometheusUsesContractNamesAndSeconds(t *testing.T) {
	rendered := metrics.RenderPrometheus(map[string]uint64{
		"liki_web_payment_refunded_total|provider=dodo": 2,
	}, map[string]float64{
		"liki_web_http_request_duration_seconds_total|method=GET|route=/metrics|status_class=2xx": 1.25,
	})
	for _, want := range []string{
		"# TYPE liki_web_payment_refunded_total counter",
		`liki_web_payment_refunded_total{provider="dodo"} 2`,
		"# TYPE liki_web_http_request_duration_seconds_total counter",
		`liki_web_http_request_duration_seconds_total{method="GET",route="/metrics",status_class="2xx"} 1.250000`,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered=%q want=%q", rendered, want)
		}
	}
}

func TestHandlerPrometheusIgnoresFormatParameter(t *testing.T) {
	metrics.Default.Inc("test_prom_total", map[string]string{"result": "ok"})
	handler := metrics.Handler("secret")
	req := httptest.NewRequest(http.MethodGet, "/metrics?format=json", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handler(rec, req)
	if got := rec.Header().Get("Content-Type"); got != "text/plain; version=0.0.4; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
	if !strings.Contains(rec.Body.String(), `test_prom_total{result="ok"} 1`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
