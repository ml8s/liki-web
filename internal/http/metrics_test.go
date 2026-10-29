package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/metrics"
)

func TestMetricsRecordsRoutePatternLowCardinality(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/things/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	handler := Metrics(mux)

	registry := metrics.New()
	old := metrics.Default
	metrics.Default = registry
	t.Cleanup(func() { metrics.Default = old })

	for _, id := range []string{"a", "b", "c"} {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/things/"+id, nil))
	}

	snapshot := registry.Snapshot()
	found := false
	for key, value := range snapshot {
		if strings.Contains(key, metrics.MetricHTTPRequests) {
			found = true
			if !strings.Contains(key, "route=GET /api/things/{id}") {
				t.Fatalf("route label must be the pattern, got %q", key)
			}
			if !strings.Contains(key, "status_class=4xx") || value != 3 {
				t.Fatalf("unexpected counter %q=%d", key, value)
			}
		}
		if strings.Contains(key, "/api/things/a") {
			t.Fatalf("raw path leaked into metrics: %q", key)
		}
	}
	if !found {
		t.Fatalf("no http_requests_total recorded: %v", snapshot)
	}
}

func TestMetricsRouteContract(t *testing.T) {
	oldRegistry := metrics.Default
	metrics.Default = metrics.New()
	t.Cleanup(func() { metrics.Default = oldRegistry })

	mux := http.NewServeMux()
	limiter := NewRateLimiter()
	t.Cleanup(limiter.Stop)
	RegisterRoutes(mux, ServerDeps{MetricsHandler: metrics.Handler()}, "test", "dev", limiter, limiter)
	handler := Metrics(mux)

	assertStatus := func(authorization string, want int) {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		if authorization != "" {
			request.Header.Set("Authorization", authorization)
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != want {
			t.Fatalf("authorization %q: status = %d, want %d", authorization, recorder.Code, want)
		}
	}
	assertStatus("", http.StatusOK)
	assertStatus("Bearer wrong", http.StatusOK)

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	request.Header.Set("Authorization", "Bearer secret")
	handler.ServeHTTP(httptest.NewRecorder(), request)
	for key := range metrics.Default.Snapshot() {
		if strings.Contains(key, metrics.MetricHTTPRequests) &&
			strings.Contains(key, `route=GET /metrics`) &&
			strings.Contains(key, `method=GET`) &&
			strings.Contains(key, `status_class=2xx`) {
			return
		}
	}
	for key := range metrics.Default.DurationSnapshot() {
		if strings.Contains(key, metrics.MetricHTTPRequestDuration) &&
			strings.Contains(key, `route=GET /metrics`) &&
			strings.Contains(key, `method=GET`) &&
			strings.Contains(key, `status_class=2xx`) {
			return
		}
	}
	t.Fatalf("metrics missing route contract counters:\ncounters=%v\ndurations=%v",
		metrics.Default.Snapshot(), metrics.Default.DurationSnapshot())
}
