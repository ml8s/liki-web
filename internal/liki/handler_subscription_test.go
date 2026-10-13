package liki

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
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

type fakeSubscriptionCheckoutGateway struct {
	request payment.SubscriptionCheckoutRequest
}

type fakeSubscriptionReconciler struct {
	checkoutID string
	event      *payment.WebhookEvent
}

func (r *fakeSubscriptionReconciler) FetchSubscriptionByCheckout(_ context.Context, checkoutID string) (*payment.WebhookEvent, error) {
	r.checkoutID = checkoutID
	return r.event, nil
}

func (g *fakeSubscriptionCheckoutGateway) CreateSubscriptionCheckout(
	_ context.Context, request payment.SubscriptionCheckoutRequest,
) (*payment.CheckoutResult, error) {
	g.request = request
	return &payment.CheckoutResult{
		SessionID:   "checkout_1",
		CheckoutURL: "https://checkout.example/subscription",
	}, nil
}

func TestSubscriptionCheckoutStoresLocalKeyAndOmitsAmount(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	entitlements := NewEntitlementStore(db)
	subscriptions := NewSubscriptionStore(db)
	if err := entitlements.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := subscriptions.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	checkoutGateway := &fakeSubscriptionCheckoutGateway{}
	catalog := NewCommerceCatalog(CommerceCatalogConfig{
		OneTimePriceMinorCNY:      7900,
		OneTimePriceMinorUSD:      2990,
		SubscriptionPriceMinorUSD: 1990,
	})
	handler := NewHandler(nil, entitlements, nil, nil, nil,
		WithCommerceCatalog(catalog),
		WithSubscriptionCommerce(subscriptions, checkoutGateway, &fakeSubscriptionGateway{}, "https://liki.hk"),
	)

	request := httptest.NewRequest(http.MethodPost, "/api/chat/products/liki-chat/checkout",
		strings.NewReader(`{"billing_mode":"subscription","locale":"en"}`))
	request.Header.Set("CF-IPCountry", "US")
	request.SetPathValue("id", string(ProductChat))
	request = request.WithContext(auth.WithAuth(request.Context(), auth.User{
		ID: "user-1", Email: "user@example.com",
	}, auth.AuthSession{}, ""))
	recorder := httptest.NewRecorder()
	handler.HandleCheckout(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			SubscriptionID string `json:"subscription_id"`
			CheckoutURL    string `json:"checkout_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.SubscriptionID == "" || response.Data.CheckoutURL != "https://checkout.example/subscription" {
		t.Fatalf("response = %+v", response.Data)
	}
	if checkoutGateway.request.UserID != "user-1" ||
		checkoutGateway.request.SubscriptionKey != response.Data.SubscriptionID ||
		checkoutGateway.request.Currency != "USD" {
		t.Fatalf("gateway request = %+v", checkoutGateway.request)
	}
	stored, err := subscriptions.GetByUser(ctx, "user-1", response.Data.SubscriptionID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != SubscriptionPending || stored.ProviderCheckoutID != "checkout_1" {
		t.Fatalf("stored = %+v", stored)
	}
}

func TestSubscriptionRoutesListAndCancel(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	entitlements := NewEntitlementStore(db)
	subscriptions := NewSubscriptionStore(db)
	if err := entitlements.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := subscriptions.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	pending, err := subscriptions.CreatePending(ctx, "user-1", "user@example.com", ProductChat, CurrencyUSD)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
UPDATE chat_subscriptions
SET provider_subscription_id='sub_1', status='active'
WHERE id=?`, pending.ID); err != nil {
		t.Fatal(err)
	}
	cancellationGateway := &fakeSubscriptionGateway{}
	handler := NewHandler(nil, entitlements, nil, nil, nil,
		WithSubscriptionCommerce(subscriptions, &fakeSubscriptionCheckoutGateway{}, cancellationGateway, "https://liki.hk"),
	)
	request := httptest.NewRequest(http.MethodGet, "/api/me/subscriptions", nil)
	request = request.WithContext(auth.WithAuth(request.Context(), auth.User{
		ID: "user-1", Email: "user@example.com",
	}, auth.AuthSession{}, ""))
	recorder := httptest.NewRecorder()
	handler.HandleSubscriptions(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	cancel := httptest.NewRequest(http.MethodPost, "/api/me/subscriptions/"+pending.ID+"/cancellation", nil)
	cancel.SetPathValue("id", pending.ID)
	cancel = cancel.WithContext(auth.WithAuth(cancel.Context(), auth.User{
		ID: "user-1", Email: "user@example.com",
	}, auth.AuthSession{}, ""))
	recorder = httptest.NewRecorder()
	handler.HandleSetCancellation(recorder, cancel)
	if recorder.Code != http.StatusOK {
		t.Fatalf("cancel status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !cancellationGateway.cancelled {
		t.Fatal("cancellation gateway was not called")
	}
}

func TestSubscriptionPaymentsRouteReturnsRefundState(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	entitlements := NewEntitlementStore(db)
	subscriptions := NewSubscriptionStore(db)
	for _, migrate := range []func(context.Context) error{entitlements.Migrate, subscriptions.Migrate} {
		if err := migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	pending, err := subscriptions.CreatePending(ctx, "user-1", "user@example.com", ProductChat, CurrencyUSD)
	if err != nil {
		t.Fatal(err)
	}
	if err := subscriptions.AttachCheckout(ctx, "user-1", pending.ID, "checkout-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE chat_subscriptions SET provider_subscription_id='sub_1' WHERE id=?`, pending.ID); err != nil {
		t.Fatal(err)
	}
	paid := payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     payment.EventPaymentSucceeded,
		Data: payment.WebhookEventData{
			SubscriptionID: "sub_1", PaymentID: "pay_1", Amount: 1990, Currency: "USD",
			Metadata: map[string]string{"liki_user_id": "user-1"},
		},
	}
	if err := subscriptions.RecordPayment(ctx, paid, nil); err != nil {
		t.Fatal(err)
	}
	refund := payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     payment.EventPaymentRefunded,
		Data: payment.WebhookEventData{
			PaymentID: "refund_1", RefundedPaymentID: "pay_1", Amount: 1990, Currency: "USD",
			Metadata: map[string]string{"liki_user_id": "user-1"},
		},
	}
	if err := subscriptions.RecordPaymentRefund(ctx, refund, nil); err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(nil, entitlements, nil, nil, nil,
		WithSubscriptionCommerce(subscriptions, &fakeSubscriptionCheckoutGateway{}, &fakeSubscriptionGateway{}, "https://liki.hk"),
	)
	request := httptest.NewRequest(http.MethodGet, "/api/me/subscription-payments", nil)
	request = request.WithContext(auth.WithAuth(request.Context(), auth.User{
		ID: "user-1", Email: "user@example.com",
	}, auth.AuthSession{}, ""))
	recorder := httptest.NewRecorder()
	handler.HandleSubscriptionPayments(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			Payments []SubscriptionPayment `json:"payments"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data.Payments) != 1 ||
		response.Data.Payments[0].Status != "refunded" ||
		response.Data.Payments[0].RefundedMinorUnits != 1990 {
		t.Fatalf("payments = %+v", response.Data.Payments)
	}
}

func TestSubscriptionListReconcilesPendingCheckout(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	entitlements := NewEntitlementStore(db)
	subscriptions := NewSubscriptionStore(db)
	for _, migrate := range []func(context.Context) error{entitlements.Migrate, subscriptions.Migrate} {
		if err := migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	pending, err := subscriptions.CreatePending(ctx, "user-1", "user@example.com", ProductChat, CurrencyUSD)
	if err != nil {
		t.Fatal(err)
	}
	if err := subscriptions.AttachCheckout(ctx, "user-1", pending.ID, "checkout_1"); err != nil {
		t.Fatal(err)
	}
	periodStart := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)
	reconciler := &fakeSubscriptionReconciler{event: &payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     payment.EventSubscriptionActive,
		Data: payment.WebhookEventData{
			SubscriptionID: "sub_1", Product: "pdt_subscription", Status: "active",
			Currency: "USD", PeriodStart: &periodStart, PeriodEnd: &periodEnd,
			Metadata: map[string]string{"liki_subscription_key": pending.ID},
		},
	}}
	handler := NewHandler(nil, entitlements, nil, nil, nil,
		WithSubscriptionCommerce(subscriptions, &fakeSubscriptionCheckoutGateway{}, &fakeSubscriptionGateway{}, "https://liki.hk"),
		WithSubscriptionReconciliation(reconciler, func(ctx context.Context, event payment.WebhookEvent) error {
			return subscriptions.ApplyWebhook(ctx, entitlements, event)
		}),
	)
	request := httptest.NewRequest(http.MethodGet, "/api/me/subscriptions", nil)
	request = request.WithContext(auth.WithAuth(request.Context(), auth.User{
		ID: "user-1", Email: "user@example.com",
	}, auth.AuthSession{}, ""))
	recorder := httptest.NewRecorder()
	handler.HandleSubscriptions(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if reconciler.checkoutID != "checkout_1" {
		t.Fatalf("reconciler checkout = %q", reconciler.checkoutID)
	}
	stored, err := subscriptions.GetByUser(ctx, "user-1", pending.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != SubscriptionActive || stored.ProviderSubscriptionID != "sub_1" {
		t.Fatalf("stored = %+v", stored)
	}
}

func TestSubscriptionCheckoutRejectsActiveSubscription(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	entitlements := NewEntitlementStore(db)
	subscriptions := NewSubscriptionStore(db)
	for _, migrate := range []func(context.Context) error{entitlements.Migrate, subscriptions.Migrate} {
		if err := migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	active, err := subscriptions.CreatePending(ctx, "user-1", "user@example.com", ProductChat, CurrencyUSD)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE chat_subscriptions SET status='active' WHERE id=?`, active.ID); err != nil {
		t.Fatal(err)
	}
	checkoutGateway := &fakeSubscriptionCheckoutGateway{}
	catalog := NewCommerceCatalog(CommerceCatalogConfig{
		OneTimePriceMinorUSD:      2990,
		SubscriptionPriceMinorUSD: 1990,
	})
	handler := NewHandler(nil, entitlements, nil, nil, nil,
		WithCommerceCatalog(catalog),
		WithSubscriptionCommerce(subscriptions, checkoutGateway, &fakeSubscriptionGateway{}, "https://liki.hk"),
	)
	request := httptest.NewRequest(http.MethodPost, "/api/chat/products/liki-chat/checkout",
		strings.NewReader(`{"billing_mode":"subscription","locale":"en"}`))
	request.Header.Set("CF-IPCountry", "US")
	request.SetPathValue("id", string(ProductChat))
	request = request.WithContext(auth.WithAuth(request.Context(), auth.User{
		ID: "user-1", Email: "user@example.com",
	}, auth.AuthSession{}, ""))
	recorder := httptest.NewRecorder()
	handler.HandleCheckout(recorder, request)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s, want 409", recorder.Code, recorder.Body.String())
	}
	var subscriptionsCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM chat_subscriptions WHERE user_id='user-1'`).Scan(&subscriptionsCount); err != nil {
		t.Fatal(err)
	}
	if subscriptionsCount != 1 {
		t.Fatalf("subscription count = %d, want existing row only", subscriptionsCount)
	}
}
