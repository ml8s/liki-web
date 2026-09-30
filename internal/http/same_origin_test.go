package http

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
)

func TestSameOriginAllowsSameSitePost(t *testing.T) {
	handler := SameOrigin(false, stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/chat/sessions", nil)
	request.Host = "liki.hk"
	request.Header.Set("Origin", "https://liki.hk")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, stdhttp.StatusNoContent)
	}
}

func TestSameOriginRejectsCrossSitePost(t *testing.T) {
	handler := SameOrigin(false, stdhttp.HandlerFunc(func(stdhttp.ResponseWriter, *stdhttp.Request) {
		t.Fatal("handler must not run")
	}))
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/chat/sessions", nil)
	request.Host = "liki.hk"
	request.Header.Set("Origin", "https://attacker.example")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, stdhttp.StatusForbidden)
	}
}

func TestSameOriginRejectsMissingBrowserContext(t *testing.T) {
	handler := SameOrigin(false, stdhttp.HandlerFunc(func(stdhttp.ResponseWriter, *stdhttp.Request) {
		t.Fatal("handler must not run")
	}))
	request := httptest.NewRequest(stdhttp.MethodDelete, "/api/profiles/id", nil)
	request.Host = "liki.hk"
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, stdhttp.StatusForbidden)
	}
}

func TestSameOriginAllowsSignedWebhookWithoutOrigin(t *testing.T) {
	handler := SameOrigin(false, stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/payments/webhook", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, stdhttp.StatusNoContent)
	}
}

func TestSameOriginRejectsForwardedHostSpoof(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_HOPS", "0")
	handler := SameOrigin(false, stdhttp.HandlerFunc(func(stdhttp.ResponseWriter, *stdhttp.Request) {
		t.Fatal("handler must not run")
	}))
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/chat/sessions", nil)
	request.Host = "internal.example"
	request.Header.Set("X-Forwarded-Host", "attacker.example")
	request.Header.Set("Origin", "https://attacker.example")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, stdhttp.StatusForbidden)
	}
}
