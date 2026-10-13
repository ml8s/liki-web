package liki

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

type fakeSubscriptionCheckoutGateway struct {
	request payment.SubscriptionCheckoutRequest
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
