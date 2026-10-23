package httputil

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type validatablePayload struct {
	Name string `json:"name"`
}

func (r validatablePayload) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	return nil
}

func TestDecodeStrictRejectsValidationFailure(t *testing.T) {
	req := strictJSONRequest(t, `{"name":" "}`, "application/json")
	recorder := httptest.NewRecorder()

	if _, ok := DecodeStrict[validatablePayload](recorder, req); ok {
		t.Fatal("invalid domain payload was accepted")
	}
	assertProblemStatus(t, recorder, http.StatusUnprocessableEntity)
}

func TestDecodeLenientIgnoresUnknownFields(t *testing.T) {
	req := strictJSONRequest(t, `{"extra":true}`, "application/json")
	recorder := httptest.NewRecorder()

	if _, ok := Decode[strictPayload](recorder, req); !ok {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
