package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testValid struct{ Name string }

func (t testValid) Validate() error { return nil }

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
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if resp.Error.Code != "test_err" {
		t.Errorf("code = %q", resp.Error.Code)
	}
}

func TestDecodeAndValidate_Valid(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"Name":"hello"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	v, ok := DecodeAndValidate[testValid](w, r)
	if !ok {
		t.Error("expected ok")
	}
	if v.Name != "hello" {
		t.Errorf("name = %q", v.Name)
	}
}

func TestDecodeAndValidate_InvalidJSON(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{bad`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	_, ok := DecodeAndValidate[testValid](w, r)
	if ok {
		t.Error("expected not ok")
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

func TestSetJWTCookieOrRespondErr_Success(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-32-chars-long-enough-ok")
	w := httptest.NewRecorder()
	if err := SetJWTCookieOrRespondErr(w, "u@t.com", "oid-2"); err != nil {
		t.Fatalf("error: %v", err)
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestLocaleFromAccept_Exact(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Language", "zh-CN")
	got := LocaleFromAccept(r)
	if got != "zh-Hans" {
		t.Errorf("got %q, want zh-Hans", got)
	}
}

func TestLocaleFromAccept_Fallback(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Language", "ja-JP")
	got := LocaleFromAccept(r)
	if got != "en" {
		t.Errorf("got %q, want en (fallback)", got)
	}
}

func TestLocaleFromAccept_Empty(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	got := LocaleFromAccept(r)
	if got != "en" {
		t.Errorf("got %q, want en (fallback)", got)
	}
}

func TestReturnToken_Verifies(t *testing.T) {
	token := ReturnToken("hepan", "oid-test")
	if !VerifyReturnToken("hepan", "oid-test", token) {
		t.Error("expected valid")
	}
}

func TestReturnToken_WrongProduct(t *testing.T) {
	token := ReturnToken("hepan", "oid-test")
	if VerifyReturnToken("mingshu", "oid-test", token) {
		t.Error("expected invalid")
	}
}

func TestReturnToken_WrongOrderID(t *testing.T) {
	token := ReturnToken("hepan", "oid-test")
	if VerifyReturnToken("hepan", "wrong", token) {
		t.Error("expected invalid")
	}
}

func TestReturnToken_Empty(t *testing.T) {
	if VerifyReturnToken("hepan", "oid", "") {
		t.Error("expected invalid for empty token")
	}
}
