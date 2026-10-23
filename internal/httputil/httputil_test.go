package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()
	RespondJSON(w, http.StatusOK, map[string]string{"hello": "world"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp struct {
		Data map[string]string `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if resp.Data["hello"] != "world" {
		t.Errorf("body = %q", w.Body.String())
	}
}

func TestRespondAppError(t *testing.T) {
	w := httptest.NewRecorder()
	RespondAppError(w, &AppError{Code: "test_err", Message: "test msg", HTTPStatus: 422})
	if w.Code != 422 {
		t.Fatalf("status = %d, want 422", w.Code)
	}
	var resp struct {
		Type   string `json:"type"`
		Title  string `json:"title"`
		Code   string `json:"code"`
		Detail string `json:"detail"`
	}
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if resp.Type != "test_err" || resp.Code != "test_err" {
		t.Errorf("type/code = %q/%q, want test_err", resp.Type, resp.Code)
	}
	if resp.Title != http.StatusText(http.StatusUnprocessableEntity) {
		t.Errorf("title = %q, want the human-readable status text %q", resp.Title, http.StatusText(http.StatusUnprocessableEntity))
	}
	if resp.Detail != "test msg" {
		t.Errorf("detail = %q", resp.Detail)
	}
}

func TestJWTSecret_FromEnv(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-test-secret")
	got := JWTSecret()
	if string(got) != "my-test-secret" {
		t.Errorf("got %q", string(got))
	}
}

func TestValidateJWTSecret_Valid(t *testing.T) {
	if err := ValidateJWTSecret("a-32-char-minimum-length-string!"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateJWTSecret_Empty(t *testing.T) {
	if err := ValidateJWTSecret(""); err == nil {
		t.Error("expected error for empty secret")
	}
}

func TestJWTAuth_NoCookie(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	_, _, ok := JWTAuth(r)
	if ok {
		t.Error("expected not ok")
	}
}

func TestSetJWTCookie(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-32-chars-long-enough-ok")
	w := httptest.NewRecorder()
	if err := SetJWTCookie(w, "user@t.com", "oid-1"); err != nil {
		t.Fatalf("SetJWTCookie: %v", err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no cookies")
	}
	if cookies[0].Name != "liki_token" {
		t.Errorf("cookie name = %q", cookies[0].Name)
	}
}
