package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/metrics"
)

func TestHandlePageViewRecordsAggregateCounter(t *testing.T) {
	before := metrics.Default.Snapshot()["analytics_pageview_total"]
	request := httptest.NewRequest(http.MethodPost, "/api/analytics/pageview", strings.NewReader(`{"path":"/zh-Hans/"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handlePageView()(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	if after := metrics.Default.Snapshot()["analytics_pageview_total"]; after != before+1 {
		t.Fatalf("counter = %d, want %d", after, before+1)
	}
}

func TestHandlePageViewRejectsInvalidJSON(t *testing.T) {
	before := metrics.Default.Snapshot()["analytics_pageview_total"]
	request := httptest.NewRequest(http.MethodPost, "/api/analytics/pageview", strings.NewReader(`{`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handlePageView()(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if after := metrics.Default.Snapshot()["analytics_pageview_total"]; after != before {
		t.Fatalf("counter = %d, want unchanged %d", after, before)
	}
}
