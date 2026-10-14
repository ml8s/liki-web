package partner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/auth"
)

func testUser(id string) auth.User {
	now := time.Now().UTC()
	return auth.User{ID: id, Email: id + "@example.com", Status: auth.UserActive, CreatedAt: now, UpdatedAt: now}
}

func authedRequest(user auth.User) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/partner-portal/kol/application", strings.NewReader(`{"note":"metaphysics creator"}`))
	return r.WithContext(auth.WithAuth(r.Context(), user, auth.AuthSession{ID: "session"}, "token"))
}

func newHandlerStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "referral.db"), []byte("test-referral-payout-secret"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store
}

func TestKOLApplicationRequiresReview(t *testing.T) {
	store := newHandlerStore(t)
	handler := NewHandler(store, func(next http.HandlerFunc) http.HandlerFunc { return next }, false)
	rec := httptest.NewRecorder()
	handler.HandleSubmitKOLApplication(rec, authedRequest(testUser("u1")))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Data struct {
			Application PartnerApplication `json:"application"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Application.Type != RoleKOL || response.Data.Application.Status != ApplicationPending {
		t.Fatalf("application = %+v", response.Data.Application)
	}
	if _, err := store.PartnerByUser(context.Background(), "u1"); err == nil {
		t.Fatal("application immediately created a partner role")
	}
	if err := store.DecideApplication(context.Background(), response.Data.Application.ID, "admin", true, "approved"); err != nil {
		t.Fatal(err)
	}
	partner, err := store.PartnerByUser(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if partner.Role != RoleKOL || partner.Status != PartnerActive {
		t.Fatalf("partner = %+v", partner)
	}
}

func TestReferralLinkSetsThirtyDayCookie(t *testing.T) {
	store := newHandlerStore(t)
	kol, err := store.createKOLPartner(context.Background(), "kol")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(store, func(next http.HandlerFunc) http.HandlerFunc { return next }, true)
	target := httptest.NewRequest(http.MethodGet, "/r/"+kol.Code, nil)
	target.SetPathValue("code", kol.Code)
	rec := httptest.NewRecorder()
	handler.HandleLink(rec, target)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != AttributionCookieName || cookies[0].Value != kol.Code {
		t.Fatalf("cookies = %+v", cookies)
	}
	if cookies[0].MaxAge != int(AttributionCookieTTL.Seconds()) || !cookies[0].HttpOnly || !cookies[0].Secure {
		t.Fatalf("cookie = %+v", cookies[0])
	}
}

func TestScoutApplicationStateAllowsReapplicationAfterRejection(t *testing.T) {
	ctx := context.Background()
	store := newHandlerStore(t)
	handler := NewHandler(store, func(next http.HandlerFunc) http.HandlerFunc { return next }, false)
	app, err := store.SubmitApplication(ctx, RoleScout, "applicant", "first")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DecideApplication(ctx, app.ID, "admin", false, "not a fit"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/partner-portal/scout/application", nil)
	req = req.WithContext(auth.WithAuth(req.Context(), testUser("applicant"), auth.AuthSession{ID: "session"}, "token"))
	rec := httptest.NewRecorder()
	handler.HandleGetScoutApplication(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Data struct {
			CanSubmit bool `json:"can_submit"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Data.CanSubmit {
		t.Fatalf("rejected application blocks reapplication: %s", rec.Body.String())
	}
}

func TestPostLoginHookAttributesNewAccountAndClearsCookie(t *testing.T) {
	store := newHandlerStore(t)
	kol, err := store.createKOLPartner(context.Background(), "kol")
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
}
