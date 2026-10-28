package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/metrics"
)

func TestHandleClientErrorRecordsSanitizedEvent(t *testing.T) {
	before := metrics.Default.Snapshot()["liki_web_client_error_total|code=engine_protocol_mismatch"]
	body := `{"code":"engine_protocol_mismatch","view":"/app/chat/123e4567-e89b-12d3-a456-426614174000","locale":"zh-Hant","transport":"direct","agent":"natal","detail":"missing required Mcp-Method header\nretry"}`
	request := httptest.NewRequest(http.MethodPost, "/api/client-errors", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handleClientError()(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	after := metrics.Default.Snapshot()["liki_web_client_error_total|code=engine_protocol_mismatch"]
	if after != before+1 {
		t.Fatalf("metric = %d, want %d", after, before+1)
	}
}

func TestHandleClientErrorRejectsInvalidCode(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/client-errors", strings.NewReader(`{"code":"Not Valid!"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handleClientError()(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", recorder.Code)
	}
}

func TestHandleClientErrorRejectsOversizedDetail(t *testing.T) {
	body := `{"code":"engine_unavailable","detail":"` + strings.Repeat("x", 301) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/api/client-errors", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handleClientError()(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", recorder.Code)
	}
}
