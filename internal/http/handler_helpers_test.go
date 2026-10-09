package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/ml8s/liki-web/internal/httputil"
)

type testReq struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (r testReq) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required),
		validation.Field(&r.Age, validation.Required, validation.Min(1)),
	)
}

func TestDecodeAndValidate_Valid(t *testing.T) {
	t.Parallel()
	body := `{"name":"test","age":25}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	w := httptest.NewRecorder()

	req, ok := httputil.Decode[testReq](w, r)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if req.Name != "test" {
		t.Errorf("Name = %s, want test", req.Name)
	}
	if req.Age != 25 {
		t.Errorf("Age = %d, want 25", req.Age)
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestDecodeAndValidate_InvalidJSON(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{bad`))
	w := httptest.NewRecorder()

	_, ok := httputil.DecodeAndValidate[testReq](w, r)
	if ok {
		t.Error("expected ok=false for bad JSON")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDecodeAndValidate_EmptyBody(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("POST", "/", strings.NewReader(``))
	w := httptest.NewRecorder()

	_, ok := httputil.DecodeAndValidate[testReq](w, r)
	if ok {
		t.Error("expected ok=false for empty body")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDecodeAndValidate_ValidationError(t *testing.T) {
	t.Parallel()
	body := `{"name":"","age":0}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	w := httptest.NewRecorder()

	_, ok := httputil.DecodeAndValidate[testReq](w, r)
	if ok {
		t.Error("expected ok=false")
	}
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestDecodeAndValidate_MissingField(t *testing.T) {
	t.Parallel()
	body := `{"name":"test"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	w := httptest.NewRecorder()

	_, ok := httputil.DecodeAndValidate[testReq](w, r)
	if ok {
		t.Error("expected ok=false")
	}
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestDecodeAndValidate_WritesErrorResponse(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{bad`))
	w := httptest.NewRecorder()

	httputil.DecodeAndValidate[testReq](w, r)

	body, err := io.ReadAll(w.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "error") {
		t.Error("error response should contain 'error' key")
	}
}

func TestLangToLocale(t *testing.T) {
	t.Parallel()
	tests := []struct {
		lang     string
		expected string
	}{
		{"zh", "zh-Hant"},
		{"zh-CN", "zh-Hans"},
		{"zh-Hans", "zh-Hans"},
		{"hk", "zh-Hant"},
		{"zh-Hant", "zh-Hant"},
		{"en", "en"},
		{"unknown", "zh-Hant"},
		{"", "zh-Hant"},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			if got := langToLocale(tt.lang); got != tt.expected {
				t.Errorf("langToLocale(%q) = %q, want %q", tt.lang, got, tt.expected)
			}
		})
	}
}
