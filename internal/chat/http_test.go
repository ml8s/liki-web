package chat

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/auth"
)

func withTestUser(userID string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.User{ID: userID, Status: auth.UserActive}
		next(w, r.WithContext(auth.WithAuth(r.Context(), user, auth.AuthSession{}, "test-token")))
	}
}

func TestHandleCreateSessionRejectsInvalidConsultationContext(t *testing.T) {
	service, _, _ := newAgentModeTestService(t)
	handler := NewHandler(service, nil, nil)
	server := httptest.NewServer(withTestUser("user-1", handler.HandleCreateSession))
	t.Cleanup(server.Close)

	body := `{
		"skill":"liki",
		"title":"合盘",
		"context":{
			"version":1,
			"intent":"hepan",
			"birth":{
				"gender":"male",
				"date":"1988-01-01",
				"time":"10:00",
				"location":{"longitude":121.4737,"timezone":"Asia/Shanghai"}
			},
			"second_birth":{
				"gender":"female",
				"date":"1990-05-12",
				"time":"08:30",
				"location":{"longitude":112.9823,"timezone":"Asia/Shanghai"}
			},
			"third_party_consent":true
		}
	}`
	response, err := http.Post(server.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d", response.StatusCode)
	}
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != "invalid_consultation_context" {
		t.Fatalf("error code = %q", payload.Code)
	}
}

func TestHandleCreateSessionRejectsUnknownConsultationMember(t *testing.T) {
	service, _, _ := newAgentModeTestService(t)
	handler := NewHandler(service, nil, nil)
	server := httptest.NewServer(withTestUser("user-1", handler.HandleCreateSession))
	t.Cleanup(server.Close)

	body := `{"skill":"liki","context":{"version":1,"intent":"huangli","place":"長沙"}}`
	response, err := http.Post(server.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d", response.StatusCode)
	}
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != "invalid_request" {
		t.Fatalf("error code = %q", payload.Code)
	}
}

func TestHandleCreateSessionKeepsCanonicalIntent(t *testing.T) {
	service, _, _ := newAgentModeTestService(t)
	handler := NewHandler(service, nil, nil)
	server := httptest.NewServer(withTestUser("user-1", handler.HandleCreateSession))
	t.Cleanup(server.Close)

	body := `{"skill":"liki","intent":"fengshui","title":"风水"}`
	response, err := http.Post(server.URL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, body = %s", response.StatusCode, raw)
	}
	var payload struct {
		Data struct {
			Intent string `json:"intent"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.Intent != "fengshui" {
		t.Fatalf("intent = %q, want %q", payload.Data.Intent, "fengshui")
	}
}
