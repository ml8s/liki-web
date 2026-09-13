package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type fakeMailer struct {
	codes []string
	err   error
}

func (f *fakeMailer) SendAuthCode(_ context.Context, _ string, code string) error {
	if f.err != nil {
		return f.err
	}
	f.codes = append(f.codes, code)
	return nil
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db, []byte("test-hmac-key"))
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store
}

func newTestService(t *testing.T, store *Store) (*Service, *fakeMailer) {
	t.Helper()
	mailer := &fakeMailer{}
	service, err := NewService(store, mailer, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	return service, mailer
}

func login(t *testing.T, service *Service, email, code string) LoginResult {
	t.Helper()
	result, err := service.VerifyEmailCode(context.Background(), email, code, "192.0.2.10", "pytest")
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestEmailLoginCreatesUserAndSession(t *testing.T) {
	store := newTestStore(t)
	service, mailer := newTestService(t, store)
	email := " User@Example.COM "

	if _, err := service.RequestEmailCode(context.Background(), email, "192.0.2.10", "test-agent"); err != nil {
		t.Fatal(err)
	}
	if len(mailer.codes) != 1 {
		t.Fatalf("codes = %d, want 1", len(mailer.codes))
	}
	result := login(t, service, email, mailer.codes[0])
	if result.User.Email != "user@example.com" {
		t.Fatalf("email = %q", result.User.Email)
	}
	if result.User.Status != UserActive {
		t.Fatalf("status = %q", result.User.Status)
	}
	if result.SessionToken == "" || result.Session.ID == "" {
		t.Fatalf("session not created: %+v", result)
	}

	user, session, err := store.GetSessionUser(context.Background(), result.SessionToken)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != result.User.ID || session.ID != result.Session.ID {
		t.Fatalf("session lookup mismatch: user=%+v session=%+v", user, session)
	}

	identities, err := store.ListIdentities(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(identities) != 1 || identities[0].Provider != ProviderEmail {
		t.Fatalf("identities = %+v", identities)
	}
}

func TestEmailCodeCooldownAndReuse(t *testing.T) {
	store := newTestStore(t)
	service, mailer := newTestService(t, store)
	email := "cooldown@example.com"

	if _, err := service.RequestEmailCode(context.Background(), email, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RequestEmailCode(context.Background(), email, "", ""); !errors.Is(err, ErrCodeCooldown) {
		t.Fatalf("second request err = %v, want cooldown", err)
	}
	// The first code is invalidated when a new code is issued after cooldown.
	store.nowFunc = func() time.Time { return time.Now().UTC().Add(2 * time.Minute) }
	if _, err := service.RequestEmailCode(context.Background(), email, "", ""); err != nil {
		t.Fatal(err)
	}
	if len(mailer.codes) != 2 {
		t.Fatalf("codes = %d, want 2", len(mailer.codes))
	}
	if _, err := service.VerifyEmailCode(context.Background(), email, mailer.codes[0], "", ""); !errors.Is(err, ErrCodeInvalid) {
		t.Fatalf("old code err = %v, want invalid", err)
	}
	if _, err := service.VerifyEmailCode(context.Background(), email, mailer.codes[1], "", ""); err != nil {
		t.Fatalf("new code err = %v", err)
	}
}

func TestEmailCodeLocksAfterMaxAttempts(t *testing.T) {
	store := newTestStore(t)
	service, mailer := newTestService(t, store)
	email := "lock@example.com"
	if _, err := service.RequestEmailCode(context.Background(), email, "", ""); err != nil {
		t.Fatal(err)
	}
	code := mailer.codes[0]

	for i := 0; i < 5; i++ {
		if _, err := service.VerifyEmailCode(context.Background(), email, "000000", "", ""); !errors.Is(err, ErrCodeInvalid) {
			t.Fatalf("attempt %d err = %v, want invalid", i+1, err)
		}
	}
	var attempts int
	var usedAt any
	if err := store.db.QueryRow(`SELECT attempts,used_at FROM auth_codes WHERE email=?`, email).Scan(&attempts, &usedAt); err != nil {
		t.Fatal(err)
	}
	t.Logf("before correct: attempts=%d used=%#v", attempts, usedAt)
	if _, err := service.VerifyEmailCode(context.Background(), email, code, "", ""); !errors.Is(err, ErrCodeLocked) {
		t.Fatalf("locked err = %v, want locked", err)
	}
}

func TestDisabledUserCannotLogin(t *testing.T) {
	store := newTestStore(t)
	service, mailer := newTestService(t, store)
	email := "disabled@example.com"
	if _, err := service.RequestEmailCode(context.Background(), email, "", ""); err != nil {
		t.Fatal(err)
	}
	result := login(t, service, email, mailer.codes[0])
	if _, err := store.db.Exec(`UPDATE users SET status='disabled' WHERE id=?`, result.User.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RequestEmailCode(context.Background(), email, "", ""); !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("issue err = %v, want disabled", err)
	}
	if _, err := service.VerifyEmailCode(context.Background(), email, mailer.codes[0], "", ""); !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("verify err = %v, want disabled", err)
	}
}

func TestExpiredSessionRejected(t *testing.T) {
	store := newTestStore(t)
	service, mailer := newTestService(t, store)
	email := "expired@example.com"
	if _, err := service.RequestEmailCode(context.Background(), email, "", ""); err != nil {
		t.Fatal(err)
	}
	result := login(t, service, email, mailer.codes[0])

	store.nowFunc = func() time.Time { return time.Now().UTC().Add(31 * 24 * time.Hour) }
	if _, _, err := store.GetSessionUser(context.Background(), result.SessionToken); !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("err = %v, want expired", err)
	}
}

func TestSessionRevokeAndList(t *testing.T) {
	store := newTestStore(t)
	service, mailer := newTestService(t, store)
	email := "sessions@example.com"

	request := func() LoginResult {
		if _, err := service.RequestEmailCode(context.Background(), email, "", ""); err != nil {
			t.Fatal(err)
		}
		return login(t, service, email, mailer.codes[len(mailer.codes)-1])
	}
	first := request()
	second := request()

	sessions, err := store.ListSessions(context.Background(), first.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("sessions = %d, want 2", len(sessions))
	}
	if err := store.RevokeSession(context.Background(), first.User.ID, first.Session.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.GetSessionUser(context.Background(), first.SessionToken); !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("revoked err = %v, want expired", err)
	}
	if _, _, err := store.GetSessionUser(context.Background(), second.SessionToken); err != nil {
		t.Fatalf("other session err = %v", err)
	}
	count, err := store.RevokeAllSessions(context.Background(), first.User.ID, second.Session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("revoked count = %d, want 0 (only the active excepted session remains)", count)
	}
}

func newAuthHandler(t *testing.T) (*Handler, *fakeMailer) {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-hmac-key")
	store := newTestStore(t)
	service, mailer := newTestService(t, store)
	handler, err := NewHandler(store, service, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	return handler, mailer
}

func postJSON(t *testing.T, handler http.HandlerFunc, path, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func TestAuthHandlerFlow(t *testing.T) {
	handler, mailer := newAuthHandler(t)

	rec := postJSON(t, handler.HandleSendEmailCode, "/api/auth/email/code", `{"email":"flow@example.com"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("send code = %d, body=%s", rec.Code, rec.Body.String())
	}
	code := mailer.codes[0]

	verify := postJSON(t, handler.HandleVerifyEmailCode, "/api/auth/email/verify", `{"email":"flow@example.com","code":"`+code+`"}`, nil)
	if verify.Code != http.StatusOK {
		t.Fatalf("verify = %d, body=%s", verify.Code, verify.Body.String())
	}
	var cookies []*http.Cookie
	for _, cookie := range verify.Result().Cookies() {
		if cookie.Name == "liki_session" {
			cookies = append(cookies, cookie)
		}
	}
	if len(cookies) != 1 || cookies[0].Value == "" {
		t.Fatal("session cookie not set")
	}

	me := handler.RequireUser(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok {
			t.Fatal("user missing")
		}
		responseOK(w, publicUser(user))
	})
	meRec := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	for _, cookie := range cookies {
		meReq.AddCookie(cookie)
	}
	me(meRec, meReq)
	if meRec.Code != http.StatusOK || !strings.Contains(meRec.Body.String(), "flow@example.com") {
		t.Fatalf("me = %d %s", meRec.Code, meRec.Body.String())
	}

	logout := handler.RequireUser(handler.HandleLogout)
	logoutRec := httptest.NewRecorder()
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	for _, cookie := range cookies {
		logoutReq.AddCookie(cookie)
	}
	logout(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout = %d", logoutRec.Code)
	}
	var clearCookie *http.Cookie
	for _, cookie := range logoutRec.Result().Cookies() {
		if cookie.Name == "liki_session" {
			clearCookie = cookie
		}
	}
	if clearCookie == nil || clearCookie.MaxAge != -1 {
		t.Fatal("logout did not clear cookie")
	}
}

func TestAuthHandlerRejectsBadRequests(t *testing.T) {
	handler, _ := newAuthHandler(t)
	tests := []struct {
		name string
		path string
		body string
		want int
	}{
		{"bad email", "/api/auth/email/code", `{"email":"not-an-email"}`, http.StatusUnprocessableEntity},
		{"missing code", "/api/auth/email/verify", `{"email":"a@b.com"}`, http.StatusUnprocessableEntity},
		{"bad json", "/api/auth/email/code", `{`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := tt.path
			switch tt.path {
			case "/api/auth/email/code":
				target = "/api/auth/email/code"
			case "/api/auth/email/verify":
				target = "/api/auth/email/verify"
			}
			rec := postJSON(t, handler.HandleSendEmailCode, target, tt.body, nil)
			if tt.path == "/api/auth/email/verify" {
				rec = postJSON(t, handler.HandleVerifyEmailCode, target, tt.body, nil)
			}
			if rec.Code != tt.want {
				t.Fatalf("status = %d, body=%s, want %d", rec.Code, rec.Body.String(), tt.want)
			}
		})
	}
}

func TestAuthJSONEncoding(t *testing.T) {
	rec := httptest.NewRecorder()
	responseOK(rec, map[string]string{"ok": "yes"})
	if rec.Code != http.StatusOK || !json.Valid(rec.Body.Bytes()) {
		t.Fatalf("response = %d %s", rec.Code, rec.Body.String())
	}
}
