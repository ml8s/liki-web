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
	request := httptest.NewRequest(stdhttp.MethodDelete, "/api/birth-profiles/id", nil)
	request.Host = "liki.hk"
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, stdhttp.StatusForbidden)
	}
}

func TestSameOriginAllowsAnonymousFeedbackWithoutBrowserContext(t *testing.T) {
	handler := SameOrigin(false, stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/feedback", nil)
	request.Host = "liki.hk"
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, stdhttp.StatusNoContent)
	}
}

func TestSameOriginAllowsFeedbackPatchWithoutBrowserContext(t *testing.T) {
	// PATCH /api/feedback/{id} must be covered by the /api/feedback prefix.
	handler := SameOrigin(false, stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	request := httptest.NewRequest(stdhttp.MethodPatch, "/api/feedback/abc-123", nil)
	request.Host = "liki.hk"
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusNoContent {
		t.Fatalf("status = %d, want %d (feedback PATCH is a script API)", recorder.Code, stdhttp.StatusNoContent)
	}
}

func TestSameOriginAllowsBearerWithoutOrigin(t *testing.T) {
	// Explicit header credentials are exempt: browsers never attach them
	// cross-site, so CSRF cannot forge them.
	handler := SameOrigin(false, stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	request := httptest.NewRequest(stdhttp.MethodPatch, "/api/feedback/abc-123", nil)
	request.Host = "liki.hk"
	request.Header.Set("Authorization", "Bearer test-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusNoContent {
		t.Fatalf("status = %d, want %d (Bearer requests are CSRF-immune)", recorder.Code, stdhttp.StatusNoContent)
	}
}

func TestSameOriginAllowsAipayWithoutPaymentProof(t *testing.T) {
	// The first aipay request has no Payment-Proof yet (it returns 402).
	handler := SameOrigin(false, stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusPaymentRequired)
	}))
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/aipay", nil)
	request.Host = "liki.hk"
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusPaymentRequired {
		t.Fatalf("status = %d, want %d", recorder.Code, stdhttp.StatusPaymentRequired)
	}
}

func TestSameOriginRejectsCrossSiteFeedback(t *testing.T) {
	handler := SameOrigin(false, stdhttp.HandlerFunc(func(stdhttp.ResponseWriter, *stdhttp.Request) {
		t.Fatal("handler must not run")
	}))
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/feedback", nil)
	request.Host = "liki.hk"
	request.Header.Set("Origin", "https://attacker.example")
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

func TestSameOriginAllowsInternalServerToServerWithoutOrigin(t *testing.T) {
	handler := SameOrigin(false, stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/internal/server-to-server", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusNoContent {
		t.Fatalf("status = %d, want %d (internal endpoints are server-to-server)", recorder.Code, stdhttp.StatusNoContent)
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
