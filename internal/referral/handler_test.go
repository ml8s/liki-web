package referral

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"liki-web/internal/auth"
)

func testUser(id string) auth.User {
	now := time.Now().UTC()
	return auth.User{ID: id, Email: id + "@example.com", Status: auth.UserActive, CreatedAt: now, UpdatedAt: now}
}

func authedRequest(user auth.User) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/referral/register", strings.NewReader(`{}`))
	return r.WithContext(auth.WithAuth(r.Context(), user, auth.AuthSession{ID: "session"}, "token"))
}

func newHandlerStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "referral.db"), func(context.Context, string) (bool, error) {
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store
}

func TestRegisterRouteReturnsPartner(t *testing.T) {
	store := newHandlerStore(t)
	handler := NewHandler(store, func(next http.HandlerFunc) http.HandlerFunc { return next }, false)
	rec := httptest.NewRecorder()
	handler.HandleRegister(rec, authedRequest(testUser("u1")))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Data struct {
			Partner PartnerStats `json:"partner"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(response.Data.Partner.Referrer.Code, "") || len(response.Data.Partner.Referrer.Code) != 10 {
		t.Fatalf("partner = %+v", response.Data.Partner)
	}
}

func TestCaptureClickSetsThirtyDayCookie(t *testing.T) {
	handler := NewHandler(newHandlerStore(t), func(next http.HandlerFunc) http.HandlerFunc { return next }, true)
	target := httptest.NewRequest(http.MethodGet, "/?ref=abcdefghij", nil)
	rec := httptest.NewRecorder()
	handler.CaptureClick(http.NotFoundHandler()).ServeHTTP(rec, target)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != AttributionCookieName || cookies[0].Value != "ABCDEFGHIJ" {
		t.Fatalf("cookies = %+v", cookies)
	}
	if cookies[0].MaxAge != int(AttributionCookieTTL.Seconds()) || !cookies[0].HttpOnly || !cookies[0].Secure {
		t.Fatalf("cookie = %+v", cookies[0])
	}
}

func TestPostLoginHookAttributesNewAccountAndClearsCookie(t *testing.T) {
	store := newHandlerStore(t)
	kol, err := store.RegisterReferrer(context.Background(), "kol", "")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(store, func(next http.HandlerFunc) http.HandlerFunc { return next }, false)
	target := httptest.NewRequest(http.MethodPost, "/api/auth/email/verify", nil)
	target.AddCookie(&http.Cookie{Name: AttributionCookieName, Value: strings.ToLower(kol.Code)})
	rec := httptest.NewRecorder()
	handler.PostLoginHook(rec, target, testUser("buyer"), true)

	attribution, ok, err := store.AttributionForUser(context.Background(), "buyer")
	if err != nil || !ok {
		t.Fatalf("attribution = %+v, %v, %v", attribution, ok, err)
	}
	if attribution.ReferrerID != kol.UserID {
		t.Fatalf("attribution = %+v", attribution)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].MaxAge != -1 {
		t.Fatalf("cookies = %+v", cookies)
	}

	// Existing accounts are not attributable merely because a ref cookie exists.
	target = httptest.NewRequest(http.MethodPost, "/api/auth/email/verify", nil)
	target.AddCookie(&http.Cookie{Name: AttributionCookieName, Value: kol.Code})
	rec = httptest.NewRecorder()
	handler.PostLoginHook(rec, target, testUser("existing"), false)
	if _, ok, _ := store.AttributionForUser(context.Background(), "existing"); ok {
		t.Fatal("existing account was attributed")
	}
	_ = url.Values{}
}
