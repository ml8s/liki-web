package enginemcp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/enginemcp"
)

func TestHandlerAllowsOnlyFreeTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":{"content":[{"type":"text","text":"{}"}]}}`))
	}))
	defer server.Close()

	mux := http.NewServeMux()
	handler := enginemcp.NewHandler(enginemcp.NewClient(server.URL, ""))
	handler.RegisterRoutes(mux, func(next http.HandlerFunc) http.HandlerFunc { return next })

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/engine/tools/bazi_chart", strings.NewReader(`{}`)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/engine/tools/time_now", strings.NewReader(`{}`)))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("non-free tool status = %d", recorder.Code)
	}
}
