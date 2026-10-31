package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/metrics"
)

func TestHandleAppFlowEventRecordsBoundedSignal(t *testing.T) {
	key := "liki_web_app_flow_event_total|event=consultation_entry|failure=|intent=liunian|intentGroup=natal|source=legacy_redirect|status=restorable"
	before := metrics.Default.Snapshot()[key]
	request := httptest.NewRequest(http.MethodPost, "/api/analytics/product-event", strings.NewReader(
		`{"event":"consultation_entry","intentGroup":"natal","source":"legacy_redirect","intent":"liunian","status":"restorable"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handleAppFlowEvent()(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if after := metrics.Default.Snapshot()[key]; after != before+1 {
		t.Fatalf("metric = %d, want %d", after, before+1)
	}
}

func TestHandleAppFlowEventRejectsUnknownDimension(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/analytics/product-event", strings.NewReader(
		`{"event":"consultation_entry","intentGroup":"all-zodiac-signs","source":"direct","status":"empty"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handleAppFlowEvent()(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandleAppFlowEventRejectsFreeFormIntent(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/analytics/product-event", strings.NewReader(
		`{"event":"consultation_bridge","intentGroup":"natal","intent":"tell me my future","status":"ok"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handleAppFlowEvent()(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnprocessableEntity)
	}
}
