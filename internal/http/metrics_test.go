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
		if strings.Contains(key, "http_requests_total") {
			found = true
			if !strings.Contains(key, "route=GET /api/things/{id}") {
				t.Fatalf("route label must be the pattern, got %q", key)
			}
			if !strings.Contains(key, "status=4xx") || value != 3 {
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
