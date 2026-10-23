package httputil

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type strictPayload struct {
	Name string `json:"name"`
}

func strictJSONRequest(t *testing.T, body, contentType string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req
}

func TestDecodeStrictJSONValid(t *testing.T) {
	req := strictJSONRequest(t, `{"name":"liki"}`, "application/json")
	recorder := httptest.NewRecorder()

	payload, ok := DecodeStrictJSON[strictPayload](recorder, req, DecodeOptions{})
	if !ok {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if payload.Name != "liki" {
		t.Fatalf("name=%q", payload.Name)
	}
}

func assertProblemStatus(t *testing.T, recorder *httptest.ResponseRecorder, status int) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status=%d want=%d body=%s", recorder.Code, status, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("content-type=%q", got)
	}
}

func TestDecodeStrictJSONRejectsUnknownFields(t *testing.T) {
	req := strictJSONRequest(t, `{"name":"liki","schema_version":"2"}`, "application/json")
	recorder := httptest.NewRecorder()

	if _, ok := DecodeStrictJSON[strictPayload](recorder, req, DecodeOptions{}); ok {
		t.Fatal("unknown field was accepted")
	}
	assertProblemStatus(t, recorder, http.StatusBadRequest)
}

func TestDecodeStrictJSONRejectsTrailingJSON(t *testing.T) {
	req := strictJSONRequest(t, `{"name":"liki"} {"name":"again"}`, "application/json")
	recorder := httptest.NewRecorder()

	if _, ok := DecodeStrictJSON[strictPayload](recorder, req, DecodeOptions{}); ok {
		t.Fatal("trailing JSON was accepted")
	}
	assertProblemStatus(t, recorder, http.StatusBadRequest)
}

func TestDecodeStrictJSONRejectsBodyOverLimit(t *testing.T) {
	req := strictJSONRequest(t, `{"name":"`+strings.Repeat("x", 32)+`"}`, "application/json")
	recorder := httptest.NewRecorder()

	if _, ok := DecodeStrictJSON[strictPayload](recorder, req, DecodeOptions{MaxBytes: 16}); ok {
		t.Fatal("oversized body was accepted")
	}
	assertProblemStatus(t, recorder, http.StatusRequestEntityTooLarge)
}

func TestDecodeStrictJSONRequiresJSONContentType(t *testing.T) {
	req := strictJSONRequest(t, `{"name":"liki"}`, "text/plain")
	recorder := httptest.NewRecorder()

	if _, ok := DecodeStrictJSON[strictPayload](recorder, req, DecodeOptions{}); ok {
		t.Fatal("non-JSON content type was accepted")
	}
	assertProblemStatus(t, recorder, http.StatusUnsupportedMediaType)
}
