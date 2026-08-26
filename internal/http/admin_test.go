package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func stubVerification(_ context.Context, _ string, _ string) {}

func TestAdminSendCode_InvalidBody(t *testing.T) {
	t.Parallel()
	h := handleAdminSendCode(stubVerification)
	req := httptest.NewRequest("POST", "/api/admin/send-code", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "error") {
		t.Errorf("body should contain error: %s", rr.Body.String())
	}
}

// ── Admin verification code flow ──
// These tests share adminCodes state and must NOT use t.Parallel().

func TestAdminSendCode_Success(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@test.com")
	ch := make(chan string, 1)
	h := handleAdminSendCode(func(_ context.Context, email, code string) {
		ch <- email
	})
	body := `{"email":"admin@test.com"}`
	req := httptest.NewRequest("POST", "/api/admin/send-code", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Error("email was not sent (timeout)")
	}
}

func TestAdminSendCode_NonAdminEmail(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@test.com")
	h := handleAdminSendCode(func(_ context.Context, email, code string) {
		t.Error("should not send email for non-admin")
	})
	body := `{"email":"hacker@evil.com"}`
	req := httptest.NewRequest("POST", "/api/admin/send-code", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestAdminVerifyCode_Success(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("ADMIN_EMAIL", "admin@test.com")
	// Send code first
	h := handleAdminSendCode(func(_ context.Context, email, code string) {})
	sr := httptest.NewRequest("POST", "/api/admin/send-code", strings.NewReader(`{"email":"admin@test.com"}`))
	sr.Header.Set("Content-Type", "application/json")
	sw := httptest.NewRecorder()
	h.ServeHTTP(sw, sr)
	if sw.Code != http.StatusOK {
		t.Fatalf("send-code: %d", sw.Code)
	}

	// Peek the code
	adminCodes.mu.Lock()
	code, ok := adminCodes.codes["admin@test.com"]
	adminCodes.mu.Unlock()
	if !ok {
		t.Fatal("no code stored")
	}

	// Verify code
	vh := handleAdminVerifyCode()
	vr := httptest.NewRequest("POST", "/api/admin/verify-code",
		strings.NewReader(fmt.Sprintf(`{"email":"admin@test.com","code":"%s"}`, code.code)))
	vr.Header.Set("Content-Type", "application/json")
	vw := httptest.NewRecorder()
	vh.ServeHTTP(vw, vr)
	if vw.Code != http.StatusOK {
		t.Fatalf("verify-code: %d, body: %s", vw.Code, vw.Body.String())
	}
	// Should have Set-Cookie for JWT
	cookies := vw.Result().Cookies()
	hasToken := false
	for _, c := range cookies {
		if c.Name == "liki_token" {
			hasToken = true
			break
		}
	}
	if !hasToken {
		t.Error("no liki_token cookie set")
	}
}

func TestAdminVerifyCode_WrongCode(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@test.com")
	vh := handleAdminVerifyCode()
	vr := httptest.NewRequest("POST", "/api/admin/verify-code",
		strings.NewReader(`{"email":"admin@test.com","code":"WRONG"}`))
	vr.Header.Set("Content-Type", "application/json")
	vw := httptest.NewRecorder()
	vh.ServeHTTP(vw, vr)
	if vw.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", vw.Code)
	}
}
