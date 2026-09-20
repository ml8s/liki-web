package admin

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAdminOnly_RequiresUnifiedSession(t *testing.T) {
	_ = os.Setenv("ADMIN_EMAIL", "admin@liki.hk")
	calls := 0
	handler := adminOnly(func(r *http.Request) (string, bool) {
		return "", false
	}, "admin@liki.hk")(func(w http.ResponseWriter, r *http.Request) {
		calls++
	})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/session", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if calls != 0 {
		t.Fatal("protected handler was called")
	}
}

func TestAdminOnly_AcceptsUnifiedAppSession(t *testing.T) {
	const adminEmail = "admin@liki.hk"
	calls := 0
	handler := adminOnly(func(r *http.Request) (string, bool) {
		return adminEmail, true
	}, adminEmail)(func(w http.ResponseWriter, r *http.Request) {
		calls++
	})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/session", nil)
	req.AddCookie(&http.Cookie{Name: "liki_session", Value: "unified-session"})
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestAdminOnly_RejectsWrongUnifiedAccount(t *testing.T) {
	handler := adminOnly(func(r *http.Request) (string, bool) {
		return "user@example.com", true
	}, "admin@liki.hk")(func(w http.ResponseWriter, r *http.Request) {})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/session", nil)
	req.AddCookie(&http.Cookie{Name: "liki_session", Value: "user-session"})
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}
