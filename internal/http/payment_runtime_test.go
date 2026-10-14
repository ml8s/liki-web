package http

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"time"

	_ "modernc.org/sqlite"

	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sales"
)

type fakeGateway struct {
	event payment.WebhookEvent
}

func (g *fakeGateway) CreateCheckout(ctx context.Context, req payment.CheckoutRequest) (*payment.CheckoutResult, error) {
	return &payment.CheckoutResult{CheckoutSessionID: "session-1", CheckoutURL: "https://pay.example/checkout"}, nil
}

func (g *fakeGateway) VerifyWebhook(raw []byte, header http.Header) (*payment.WebhookEvent, error) {
	return &g.event, nil
}

func (g *fakeGateway) FetchPayment(ctx context.Context, intent payment.PaymentIntent) (*payment.WebhookEvent, error) {
	return nil, nil
}

func (g *fakeGateway) WebhookAck() []byte { return []byte("success") }

func newPaymentRuntime(t *testing.T) (*payment.Runtime, *fakeGateway, *sales.OrderStore, string) {
	t.Helper()
	ctx := context.Background()
	mainDB, err := sql.Open("sqlite", t.TempDir()+"/main.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mainDB.Close() })

	intents := payment.NewIntentStore(mainDB)
	notifications := payment.NewNotificationStore(mainDB)
	if err := intents.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := notifications.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	fulfillments := payment.NewPaymentCompletionTaskStore(mainDB)
	events := payment.NewPaymentEventStore(mainDB)
	if err := fulfillments.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := events.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	orderStore := sales.NewOrderStoreWithCatalog(mainDB, product.NewCatalog(product.CatalogConfig{
		OneTimePriceMinorCNY: 7900, OneTimePriceMinorUSD: 2990, SubscriptionPriceMinorUSD: 1990,
	}))
	entitlementStore := entitlement.NewStore(mainDB)
	if err := orderStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := entitlementStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	order, err := orderStore.CreateOrder(ctx, "user_1", "user@example.com", product.Chat, "CNY")
	if err != nil {
		t.Fatal(err)
	}
	gateway := &fakeGateway{}
	runtime := payment.NewRuntime(
		map[payment.Provider]payment.Gateway{payment.ProviderXunhu: gateway},
		map[string]payment.ProductFlow{
			string(product.Chat): sales.NewProductPaymentFlow(orderStore, entitlementStore, product.Chat, 30*time.Hour),
		},
		intents,
		notifications,
		fulfillments,
		events,
		&fakeEmailSender{},
		payment.RuntimeOptions{BaseURL: "https://liki.hk", AdminEmail: "admin@example.com"},
	)
	return runtime, gateway, orderStore, order.ID
}

type fakeEmailSender struct{}

func (s *fakeEmailSender) Send(ctx context.Context, msg payment.EmailMessage) error { return nil }

func postJSON(t *testing.T, mux *http.ServeMux, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestSharedPaymentHTTPFlow(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-32-chars-long-enough")
	runtime, gateway, orderStore, bizOrderID := newPaymentRuntime(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, ServerDeps{PaymentRuntime: runtime}, "test", "test", NewRateLimiter(), NewRateLimiter())

	intent, err := runtime.CreateIntent(context.Background(), payment.CreateIntentInput{
		Product: string(product.Chat), BusinessOrderID: bizOrderID, Provider: payment.ProviderXunhu,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := orderStore.AttachIntent(context.Background(), "user_1", bizOrderID, intent.ID); err != nil {
		t.Fatal(err)
	}
	intentID := intent.ID
	checkout, _, err := runtime.StartCheckout(context.Background(), intentID)
	if err != nil {
		t.Fatal(err)
	}
	if checkout.CheckoutURL != "https://pay.example/checkout" {
		t.Fatalf("checkout URL = %q", checkout.CheckoutURL)
	}
	if checkout.StatusToken == "" {
		t.Fatal("checkout did not include status token")
	}
	statusReq := httptest.NewRequest(http.MethodGet, "/api/payments/intents/"+intentID+"/status?status_token="+checkout.StatusToken, nil)
	statusReq.SetPathValue("id", intentID)
	statusRec := httptest.NewRecorder()
	mux.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("status = %d %s", statusRec.Code, statusRec.Body.String())
	}

	gateway.event = payment.WebhookEvent{
		Provider: payment.ProviderXunhu,
		Type:     payment.EventPaymentSucceeded,
		Data: payment.WebhookEventData{
			OrderID: intentID, Amount: 7900, Currency: "CNY", PaymentID: "pay-1",
		},
	}
	rec := postJSON(t, mux, "/api/payments/webhook", `signature-body`)
	if rec.Code != http.StatusOK {
		t.Fatalf("webhook = %d %s", rec.Code, rec.Body.String())
	}
	if _, err := runtime.ProcessPaymentCompletionTasks(context.Background(), 10); err != nil {
		t.Fatal(err)
	}

	returnReq := httptest.NewRequest(http.MethodGet, "/api/payments/return/"+intentID, nil)
	returnReq.SetPathValue("id", intentID)
	returnRec := httptest.NewRecorder()
	mux.ServeHTTP(returnRec, returnReq)
	if returnRec.Code != http.StatusSeeOther {
		t.Fatalf("return = %d, want 303", returnRec.Code)
	}
	if got := returnRec.Header().Get("Location"); !strings.Contains(got, "/app/chat?order_id="+bizOrderID+"&status=paid") {
		t.Fatalf("redirect = %q", got)
	}
	cookies := returnRec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Value == "" {
		t.Fatal("paid return did not set auth cookie")
	}
}

func TestSharedPaymentWebhookReturnsProviderAck(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-32-chars-long-enough")
	runtime, gateway, orderStore, bizOrderID := newPaymentRuntime(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, ServerDeps{PaymentRuntime: runtime}, "test", "test", NewRateLimiter(), NewRateLimiter())

	intent, err := runtime.CreateIntent(context.Background(), payment.CreateIntentInput{
		Product: string(product.Chat), BusinessOrderID: bizOrderID, Provider: payment.ProviderXunhu,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := orderStore.AttachIntent(context.Background(), "user_1", bizOrderID, intent.ID); err != nil {
		t.Fatal(err)
	}
	gateway.event = payment.WebhookEvent{
		Provider: payment.ProviderXunhu,
		Type:     payment.EventPaymentSucceeded,
		Data: payment.WebhookEventData{
			OrderID: intent.ID, Amount: 7900, PaymentID: "pay-ack",
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/payments/webhook", strings.NewReader("signature-body"))
	ackRec := httptest.NewRecorder()
	mux.ServeHTTP(ackRec, req)
	if ackRec.Code != http.StatusOK || ackRec.Body.String() != "success" {
		t.Fatalf("webhook ack = %d %q, want 200 success", ackRec.Code, ackRec.Body.String())
	}
}

func TestSharedPaymentWebhookRouteOnly(t *testing.T) {
	runtime, _, _, _ := newPaymentRuntime(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, ServerDeps{PaymentRuntime: runtime}, "test", "test", NewRateLimiter(), NewRateLimiter())
	for _, path := range []string{
		"/api/naming/webhook", "/api/mingshu/webhook", "/api/hepan/webhook",
		"/api/payments/intents", "/api/payments/intents/id/checkout",
	} {
		rec := postJSON(t, mux, path, `{}`)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s = %d, want 404", path, rec.Code)
		}
	}
}
