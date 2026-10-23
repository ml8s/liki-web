package partner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/payment"
)

func newPartnerJourneyStore(t *testing.T) (*Store, Partner, Partner) {
	t.Helper()
	ctx := context.Background()
	store := newAdminStore(t)
	scout, err := store.CreateScoutPartner(ctx, "scout", "admin", "standalone scout")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Attribute(ctx, "kol", scout.Code); err != nil {
		t.Fatal(err)
	}
	kol, err := store.createKOLPartner(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Attribute(ctx, "buyer", kol.Code); err != nil {
		t.Fatal(err)
	}
	sale := paidCommission(payment.ProviderXunhu, "buyer", 50000)
	if _, _, err := store.EnqueueCommissionPayment(ctx, sale); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionPayments(ctx, 10); err != nil || n != 1 {
		t.Fatalf("processed=%d err=%v", n, err)
	}
	return store, kol, scout
}

func roleAPIRequest(user auth.User, method, target string, body string) *http.Request {
	if body == "" {
		return httptest.NewRequest(method, target, nil).WithContext(auth.WithAuth(context.Background(), user, auth.AuthSession{}, ""))
	}
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request.WithContext(auth.WithAuth(request.Context(), user, auth.AuthSession{}, ""))
}

func decodeAPI(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Data
}

func TestPartnerHTTPJourneysStayRoleScoped(t *testing.T) {
	store, kol, scout := newPartnerJourneyStore(t)
	handler := NewHandler(store, nil, false)
	kolUser := auth.User{ID: kol.UserID, Email: "kol@example.com", Status: auth.UserActive}
	scoutUser := auth.User{ID: scout.UserID, Email: "scout@example.com", Status: auth.UserActive}

	// KOL sees only its own ledger balance and can request its own payout.
	rec := httptest.NewRecorder()
	handler.HandleKOLCommissions(rec, roleAPIRequest(kolUser, http.MethodGet, "/api/partner-portal/kol/commissions", ""))
	data := decodeAPI(t, rec)
	balances, _ := data["balances"].([]any)
	if len(balances) != 1 {
		t.Fatalf("kol balances=%v", data["balances"])
	}
	rec = httptest.NewRecorder()
	handler.HandleCreateKOLPayout(rec, roleAPIRequest(kolUser, http.MethodPost, "/api/partner-portal/kol/payouts", `{"method":"alipay","account":"kol@example.com"}`))
	data = decodeAPI(t, rec)
	payout, _ := data["payout"].(map[string]any)
	if payout["role"] != string(RoleKOL) || payout["amount_minor"] != float64(25000) || payout["account"] != "k***@example.com" {
		t.Fatalf("kol payout=%v", payout)
	}

	// Scout sees its own channel balance and can request an independent payout.
	rec = httptest.NewRecorder()
	handler.HandleScoutCommissions(rec, roleAPIRequest(scoutUser, http.MethodGet, "/api/partner-portal/scout/commissions", ""))
	data = decodeAPI(t, rec)
	balances, _ = data["balances"].([]any)
	if len(balances) != 1 {
		t.Fatalf("scout balances=%v", data["balances"])
	}
	rec = httptest.NewRecorder()
	handler.HandleCreateScoutPayout(rec, roleAPIRequest(scoutUser, http.MethodPost, "/api/partner-portal/scout/payouts", `{"method":"paypal","account":"scout@example.com"}`))
	data = decodeAPI(t, rec)
	payout, _ = data["payout"].(map[string]any)
	if payout["role"] != string(RoleScout) || payout["amount_minor"] != float64(10000) || payout["account"] != "s***@example.com" {
		t.Fatalf("scout payout=%v", payout)
	}

	// A request across the role boundary is treated as nonexistent, not as a
	// recognized role error, so neither surface advertises the other.
	rec = httptest.NewRecorder()
	handler.HandleKOLSession(rec, roleAPIRequest(scoutUser, http.MethodGet, "/api/partner-portal/kol/session", ""))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("scout -> kol API status=%d body=%s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	handler.HandleScoutSession(rec, roleAPIRequest(kolUser, http.MethodGet, "/api/partner-portal/scout/session", ""))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("kol -> scout API status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRoleScopedHandlerRejectsOtherRole(t *testing.T) {
	ctx := context.Background()
	store := newAdminStore(t)
	kol, err := store.createKOLPartner(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(store, nil, false)
	user := auth.User{ID: kol.UserID, Email: "kol@example.com", Status: auth.UserActive}
	request := httptest.NewRequest(http.MethodGet, "/api/partner-portal/scout/session", nil).WithContext(auth.WithAuth(ctx, user, auth.AuthSession{}, ""))
	recorder := httptest.NewRecorder()
	called := false
	handler.requireScout(func(http.ResponseWriter, *http.Request) { called = true })(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	if called {
		t.Fatal("scout handler was called for KOL")
	}
}
