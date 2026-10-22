package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/adminauth"
	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/partner"
	"github.com/ml8s/liki-web/internal/sales"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestCompensationEndpointRequiresActivePlatformUserAndCreatesAuditedGrant(t *testing.T) {
	ctx := context.Background()
	authorizer := openAdminAuth(t)
	authDB, err := sqliteutil.Open(filepath.Join(t.TempDir(), "platform-auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = authDB.Close() })
	users := auth.NewStore(authDB, []byte("test-secret"), auth.DefaultConfig())
	if err := users.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := authDB.Exec(`
INSERT INTO users(id,email,status,locale,created_at,updated_at)
VALUES('user-1','customer@example.com','active','zh-Hans',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := authDB.Exec(`
INSERT INTO users(id,email,status,locale,created_at,updated_at)
VALUES('user-disabled','disabled@example.com','disabled','zh-Hans',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := authDB.Exec(`
INSERT INTO users(id,email,status,locale,created_at,updated_at,deleted_at)
VALUES('user-deleted','deleted@example.com','deleted','zh-Hans',?,?,'2026-01-01T00:00:00Z')`, now, now); err != nil {
		t.Fatal(err)
	}

	entitlementDB, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = entitlementDB.Close() })
	entitlements := entitlement.NewStore(entitlementDB)
	if err := entitlements.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	if err := authorizer.BootstrapAdminByEmail(ctx, "support@example.com"); err != nil {
		t.Fatal(err)
	}
	reviewer := auth.User{ID: "user-support", Email: "support@example.com", Status: auth.UserActive}
	notifications := &compensationNotifications{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, nil, authorizer, identityFor(reviewer), nil,
		entitlements, users, nil, nil, notifications.record)

	disabledRecorder := httptest.NewRecorder()
	mux.ServeHTTP(disabledRecorder, httptest.NewRequest(http.MethodPost, "/api/admin/entitlements/compensation",
		strings.NewReader(`{"email":"disabled@example.com","template":"support_7d","reason":"check"}`)))
	if disabledRecorder.Code != http.StatusConflict {
		t.Fatalf("disabled user status=%d body=%s", disabledRecorder.Code, disabledRecorder.Body.String())
	}
	deletedRecorder := httptest.NewRecorder()
	mux.ServeHTTP(deletedRecorder, httptest.NewRequest(http.MethodPost, "/api/admin/entitlements/compensation",
		strings.NewReader(`{"email":"deleted@example.com","template":"support_7d","reason":"check"}`)))
	if deletedRecorder.Code != http.StatusNotFound {
		t.Fatalf("deleted user status=%d body=%s", deletedRecorder.Code, deletedRecorder.Body.String())
	}
	invalidRecorder := httptest.NewRecorder()
	mux.ServeHTTP(invalidRecorder, httptest.NewRequest(http.MethodPost, "/api/admin/entitlements/compensation",
		strings.NewReader(`{"email":"customer@example.com","template":"support_999d","reason":"check"}`)))
	if invalidRecorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid template status=%d body=%s", invalidRecorder.Code, invalidRecorder.Body.String())
	}

	request := httptest.NewRequest(http.MethodPost, "/api/admin/entitlements/compensation",
		strings.NewReader(`{"email":"customer@example.com","template":"support_30d","reason":"service incident","idempotency_key":"ticket-1"}`))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	items, err := entitlements.ListForAdmin(ctx, "customer@example.com", "active", 10)
	if err != nil || len(items) != 1 {
		t.Fatalf("entitlements=%+v err=%v", items, err)
	}
	if items[0].Source != "admin_compensation" || items[0].Status != entitlement.Active {
		t.Fatalf("unexpected entitlement: %+v", items[0])
	}
	if got := items[0].ExpiresAt.Sub(items[0].StartsAt); got < 29*24*time.Hour || got > 31*24*time.Hour {
		t.Fatalf("unexpected compensation duration: %s", got)
	}
	if len(notifications.items) != 1 || notifications.items[0].template != "support_30d" {
		t.Fatalf("unexpected compensation notifications: %+v", notifications.items)
	}

	// A timeout followed by the same operator retry must not stack access.
	request = httptest.NewRequest(http.MethodPost, "/api/admin/entitlements/compensation",
		strings.NewReader(`{"email":"customer@example.com","template":"support_30d","reason":"service incident","idempotency_key":"ticket-1"}`))
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("retry status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	retried, err := entitlements.ListForAdmin(ctx, "customer@example.com", "active", 10)
	if err != nil || len(retried) != 1 || retried[0].ID != items[0].ID {
		t.Fatalf("retry duplicated entitlement: %+v err=%v", retried, err)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/admin/entitlements/compensation",
		strings.NewReader(`{"email":"missing@example.com","template":"support_7d","reason":"support"}`))
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("missing user status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCompensationNotificationFailureIsAudited(t *testing.T) {
	ctx := context.Background()
	authorizer := openAdminAuth(t)
	authDB, err := sqliteutil.Open(filepath.Join(t.TempDir(), "platform-auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = authDB.Close() })
	users := auth.NewStore(authDB, []byte("test-secret"), auth.DefaultConfig())
	if err := users.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := authDB.Exec(`
INSERT INTO users(id,email,status,locale,created_at,updated_at)
VALUES('user-1','customer@example.com','active','en',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	entitlementDB, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = entitlementDB.Close() })
	entitlements := entitlement.NewStore(entitlementDB)
	if err := entitlements.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := authorizer.BootstrapAdminByEmail(ctx, "admin@example.com"); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	RegisterRoutes(mux, nil, authorizer, identityFor(activeAdmin()), nil,
		entitlements, users, nil, nil, func(context.Context, auth.User, entitlement.Entitlement, string) error {
			return errors.New("outbox unavailable")
		})
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/admin/entitlements/compensation",
		strings.NewReader(`{"email":"customer@example.com","template":"support_7d","reason":"incident"}`)))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			EmailQueued bool `json:"email_queued"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.EmailQueued {
		t.Fatal("queued flag should be false after notifier failure")
	}
	audit, err := authorizer.ListAudit(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range audit {
		if item.Action == "entitlement.compensation_notification_failed" &&
			strings.Contains(item.Reason, "incident") &&
			strings.Contains(item.Reason, "outbox unavailable") {
			found = true
		}
	}
	if !found {
		t.Fatalf("notification failure audit missing: %+v", audit)
	}
}

func TestFinanceSummaryCombinesSalesAndPartnerTotals(t *testing.T) {
	ctx := context.Background()
	authorizer := openAdminAuth(t)

	commerceDB, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = commerceDB.Close() })
	orders := sales.NewOrderStore(commerceDB)
	subscriptions := sales.NewSubscriptionStore(commerceDB)
	if err := orders.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := subscriptions.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := commerceDB.Exec(`
INSERT INTO sales_orders(id,user_id,email,product_id,amount_minor,currency,status,payment_intent_id,created_at,updated_at,paid_at)
VALUES('order-1','user-1','customer@example.com','liki-chat',1000,'USD','paid','intent-1',?, ?,?)`, now, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := commerceDB.Exec(`
INSERT INTO sales_subscription_payments(id,subscription_id,user_id,product_id,provider,provider_payment_id,
 amount_minor_units,refunded_minor_units,currency_code,status,created_at,updated_at)
VALUES('payment-1','subscription-1','user-1','liki-chat','dodo','provider-payment',2000,500,'USD','paid',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := commerceDB.Exec(`
INSERT INTO sales_subscription_payment_refunds(id,payment_id,provider,provider_refund_id,
 amount_minor_units,currency_code,created_at)
VALUES('refund-1','payment-1','dodo','provider-refund',500,'USD',?)`, now); err != nil {
		t.Fatal(err)
	}

	partnerSummary := fakePartnerFinance{commission: 200, payoutPaid: 100, payoutPending: 0}
	commerce := sales.NewSalesStores(orders, subscriptions)
	mux := http.NewServeMux()
	RegisterRoutes(mux, nil, authorizer, identityFor(activeAdmin()), nil, nil, nil, commerce, partnerSummary, nil)

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/admin/finance/summary?days=30", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			OneTimePaid      []sales.AdminMoney `json:"one_time_paid"`
			SubscriptionPaid []sales.AdminMoney `json:"subscription_paid"`
			Refunds          []sales.AdminMoney `json:"subscription_refunded"`
			NetRevenue       []sales.AdminMoney `json:"net_product_revenue"`
			CommissionNet    []sales.AdminMoney `json:"partner_commission_net"`
			PayoutPaid       []sales.AdminMoney `json:"partner_payout_paid"`
			PendingPayout    []sales.AdminMoney `json:"partner_pending_payout"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	assertMoney(t, response.Data.OneTimePaid, "USD", 1000)
	assertMoney(t, response.Data.SubscriptionPaid, "USD", 2000)
	assertMoney(t, response.Data.Refunds, "USD", 500)
	assertMoney(t, response.Data.NetRevenue, "USD", 2500)
	assertMoney(t, response.Data.CommissionNet, "USD", 200)
	assertMoney(t, response.Data.PayoutPaid, "USD", 100)
	assertMoney(t, response.Data.PendingPayout, "USD", 0)
}

func openAdminAuth(t *testing.T) *adminauth.Store {
	t.Helper()
	authorizer, err := adminauth.Open(filepath.Join(t.TempDir(), "platform.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = authorizer.Close() })
	if err := authorizer.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := authorizer.BootstrapAdminByEmail(context.Background(), "admin@example.com"); err != nil {
		t.Fatal(err)
	}
	return authorizer
}

func activeAdmin() auth.User {
	return auth.User{ID: "user-admin", Email: "admin@example.com", Status: auth.UserActive}
}

func identityFor(user auth.User) func(*http.Request) (auth.User, bool) {
	return func(*http.Request) (auth.User, bool) { return user, true }
}

func assertMoney(t *testing.T, items []sales.AdminMoney, currency string, amount int64) {
	t.Helper()
	for _, item := range items {
		if item.Currency == currency && item.AmountMinor == amount {
			return
		}
	}
	t.Fatalf("missing %s %d in %+v", currency, amount, items)
}

type fakePartnerFinance struct {
	commission    int64
	payoutPaid    int64
	payoutPending int64
}

func (f fakePartnerFinance) AdminFinanceSummary(_ context.Context, start, end time.Time) (partner.AdminFinanceSummary, error) {
	_ = start
	_ = end
	return partner.AdminFinanceSummary{
		CommissionNet:       []partner.AdminMoney{{Currency: "USD", AmountMinor: f.commission}},
		PayoutPaid:          []partner.AdminMoney{{Currency: "USD", AmountMinor: f.payoutPaid}},
		PendingPayoutAmount: []partner.AdminMoney{{Currency: "USD", AmountMinor: f.payoutPending}},
	}, nil
}

type compensationNotifications struct {
	items []compensationNotification
}

type compensationNotification struct {
	user        auth.User
	entitlement entitlement.Entitlement
	template    string
}

func (n *compensationNotifications) record(_ context.Context, user auth.User, entitlement entitlement.Entitlement, template string) error {
	n.items = append(n.items, compensationNotification{user: user, entitlement: entitlement, template: template})
	return nil
}
