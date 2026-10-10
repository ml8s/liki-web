package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newIdentityFixture(t *testing.T) (*Handler, *Service, *fakeMailer) {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-hmac-key")
	store := newTestStore(t)
	service, mailer := newTestService(t, store)
	handler, err := NewHandler(store, service, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	return handler, service, mailer
}

// requestCode issues an email login code through the fixture mailer.
func requestCode(t *testing.T, service *Service, mailer *fakeMailer, email string) string {
	t.Helper()
	if _, err := service.RequestEmailCode(context.Background(), email, "192.0.2.10", "pytest", "zh-Hans"); err != nil {
		t.Fatal(err)
	}
	if len(mailer.codes) == 0 {
		t.Fatal("mailer produced no code")
	}
	return mailer.codes[len(mailer.codes)-1]
}

func decodeEnvelope(t *testing.T, body []byte, out any) {
	t.Helper()
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode envelope: %v (body=%s)", err, body)
	}
	if len(envelope.Data) == 0 {
		t.Fatalf("missing data envelope: %s", body)
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		t.Fatalf("decode data: %v (data=%s)", err, envelope.Data)
	}
}

func decodeError(t *testing.T, body []byte) (code string) {
	t.Helper()
	var problem struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &problem); err != nil {
		t.Fatalf("decode error: %v (body=%s)", err, body)
	}
	return problem.Type
}

func TestHandleMeRejectsUnauthenticatedRequest(t *testing.T) {
	handler, _, _ := newIdentityFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	handler.HandleMe(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	t.Logf("DEBUG body = %s", rec.Body.String())
	if code := decodeError(t, rec.Body.Bytes()); code != "unauthorized" {
		t.Fatalf("error code = %q", code)
	}
}

func TestHandleMeReturnsCurrentUser(t *testing.T) {
	handler, _, _ := newIdentityFixture(t)
	user := User{ID: "user-1", Email: "user@example.com", Status: UserActive}
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req = req.WithContext(WithAuth(req.Context(), user, AuthSession{ID: "s1"}, "tok"))
	rec := httptest.NewRecorder()
	handler.HandleMe(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var me PublicUser
	decodeEnvelope(t, rec.Body.Bytes(), &me)
	if me.ID != user.ID || me.Email != user.Email {
		t.Fatalf("me = %+v", me)
	}
}

func TestHandleSessionsRejectsUnauthenticatedRequest(t *testing.T) {
	handler, _, _ := newIdentityFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/api/me/sessions", nil)
	rec := httptest.NewRecorder()
	handler.HandleSessions(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	t.Logf("DEBUG body = %s", rec.Body.String())
}

func TestHandleSessionsMarksCurrentSession(t *testing.T) {
	handler, service, mailer := newIdentityFixture(t)
	const email = "Sessions@Example.COM"
	result := login(t, service, email, requestCode(t, service, mailer, email))

	newRequest := func(withToken bool) *http.Request {
		token := ""
		if withToken {
			token = result.SessionToken
		}
		req := httptest.NewRequest(http.MethodGet, "/api/me/sessions", nil)
		return req.WithContext(WithAuth(req.Context(), result.User, result.Session, token))
	}

	t.Run("with session token", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.HandleSessions(rec, newRequest(true))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
		}
		var sessions []PublicSession
		decodeEnvelope(t, rec.Body.Bytes(), &sessions)
		if len(sessions) != 1 {
			t.Fatalf("sessions = %+v", sessions)
		}
		if !sessions[0].IsCurrent {
			t.Fatal("expected the active session to be marked current")
		}
	})

	t.Run("without session token", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.HandleSessions(rec, newRequest(false))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
		}
		var sessions []PublicSession
		decodeEnvelope(t, rec.Body.Bytes(), &sessions)
		if len(sessions) != 1 {
			t.Fatalf("sessions = %+v", sessions)
		}
		if sessions[0].IsCurrent {
			t.Fatal("no session should be current without a token")
		}
	})
}

func TestIdentityFromRequest(t *testing.T) {
	handler, service, mailer := newIdentityFixture(t)
	const email = "Identity@Example.COM"
	result := login(t, service, email, requestCode(t, service, mailer, email))

	t.Run("missing cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/whoami", nil)
		if _, ok := handler.IdentityFromRequest(req); ok {
			t.Fatal("expected no identity without cookie")
		}
	})

	t.Run("active session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/whoami", nil)
		req.AddCookie(&http.Cookie{Name: DefaultConfig().CookieName, Value: result.SessionToken})
		user, ok := handler.IdentityFromRequest(req)
		if !ok {
			t.Fatal("expected identity for active session")
		}
		if user.ID != result.User.ID {
			t.Fatalf("user = %+v, want %+v", user, result.User)
		}
	})

	t.Run("unknown token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/whoami", nil)
		req.AddCookie(&http.Cookie{Name: DefaultConfig().CookieName, Value: "not-a-real-token"})
		if _, ok := handler.IdentityFromRequest(req); ok {
			t.Fatal("expected no identity for unknown token")
		}
	})
}
