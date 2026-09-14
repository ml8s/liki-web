package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"liki-web/internal/naming"
	"liki-web/internal/payment"

	"liki-web/internal/paymentflow"
	"liki-web/internal/product"
)

type fakeGateway struct {
	event payment.WebhookEvent
}

func (g *fakeGateway) CreateCheckout(ctx context.Context, req payment.CheckoutRequest) (*payment.CheckoutResult, error) {
	return &payment.CheckoutResult{SessionID: "session-1", CheckoutURL: "https://pay.example/checkout"}, nil
}

func (g *fakeGateway) VerifyWebhook(raw []byte, header http.Header) (*payment.WebhookEvent, error) {
	return &g.event, nil
}

func (g *fakeGateway) FetchPayment(ctx context.Context, intent payment.PaymentIntent) (*payment.WebhookEvent, error) {
	return nil, nil
}

func (g *fakeGateway) WebhookAck() []byte { return []byte("success") }

func newPaymentRuntime(t *testing.T) (*payment.Runtime, *fakeGateway) {
	t.Helper()
	ctx := context.Background()
	mainDB, err := sql.Open("sqlite", t.TempDir()+"/main.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mainDB.Close() })
	namingDB, err := sql.Open("sqlite", t.TempDir()+"/naming.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = namingDB.Close() })

	intents := payment.NewIntentStore(mainDB)
	notifications := payment.NewNotificationStore(mainDB)
	if err := intents.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := notifications.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	fulfillments := payment.NewFulfillmentTaskStore(mainDB)
	events := payment.NewPaymentEventStore(mainDB)
	if err := fulfillments.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := events.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	namingStore := naming.NewStore(namingDB)
	if err := namingStore.CreateOrder(ctx, "biz-1", "naming", 990, "CNY", "user@example.com", "zh-Hant"); err != nil {
		t.Fatal(err)
	}
	gateway := &fakeGateway{}
	return payment.NewRuntime(
		map[payment.Provider]payment.Gateway{payment.ProviderXunhu: gateway},
		map[string]payment.ProductFlow{
			string(product.ProductNaming): paymentflow.NewNamingFlow(namingStore, 15),
		},
		intents,
		notifications,
		fulfillments,
		events,
		&fakeEmailSender{},
		payment.RuntimeOptions{BaseURL: "https://liki.hk", AdminEmail: "admin@example.com"},
	), gateway
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
	runtime, gateway := newPaymentRuntime(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, ServerDeps{PaymentRuntime: runtime}, "test", "test", NewRateLimiter(), NewRateLimiter())

	rec := postJSON(t, mux, "/api/payments/intents", `{"product":"naming","business_order_id":"biz-1","provider":"xunhu"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("create intent = %d %s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Data payment.PaymentIntent `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	intentID := envelope.Data.ID

	rec = postJSON(t, mux, "/api/payments/intents/"+intentID+"/checkout", `{}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("checkout = %d %s", rec.Code, rec.Body.String())
	}

	var checkoutEnvelope struct {
		Data struct {
			CheckoutURL string `json:"checkout_url"`
			QRURL       string `json:"qr_url"`
			StatusToken string `json:"status_token"`
			SuccessURL  string `json:"success_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &checkoutEnvelope); err != nil {
		t.Fatal(err)
	}
	if checkoutEnvelope.Data.CheckoutURL != "https://pay.example/checkout" ||
		checkoutEnvelope.Data.SuccessURL != "/zh-Hant/naming/chat?order_id=biz-1" {
		t.Fatalf("checkout response = %+v", checkoutEnvelope.Data)
	}

	if checkoutEnvelope.Data.StatusToken == "" {
		t.Fatal("checkout response did not include status token")
	}
	statusReq := httptest.NewRequest(http.MethodGet, "/api/payments/intents/"+intentID+"/status?status_token="+checkoutEnvelope.Data.StatusToken, nil)
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
			OrderID: intentID, Amount: 990, Currency: "CNY", PaymentID: "pay-1",
		},
	}
	rec = postJSON(t, mux, "/api/payments/webhook", `signature-body`)
	if rec.Code != http.StatusOK {
		t.Fatalf("webhook = %d %s", rec.Code, rec.Body.String())
	}
	if _, err := runtime.ProcessFulfillmentTasks(context.Background(), 10); err != nil {
		t.Fatal(err)
	}

	returnReq := httptest.NewRequest(http.MethodGet, "/api/payments/return/"+intentID, nil)
	returnReq.SetPathValue("id", intentID)
	returnRec := httptest.NewRecorder()
	mux.ServeHTTP(returnRec, returnReq)
	if returnRec.Code != http.StatusSeeOther {
		t.Fatalf("return = %d, want 303", returnRec.Code)
	}
	if got := returnRec.Header().Get("Location"); !strings.Contains(got, "/zh-Hant/naming/chat?order_id=biz-1") {
		t.Fatalf("redirect = %q", got)
	}
	cookies := returnRec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Value == "" {
		t.Fatal("paid return did not set auth cookie")
	}
}

func TestSharedPaymentWebhookReturnsProviderAck(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-32-chars-long-enough")
	runtime, gateway := newPaymentRuntime(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, ServerDeps{PaymentRuntime: runtime}, "test", "test", NewRateLimiter(), NewRateLimiter())

	rec := postJSON(t, mux, "/api/payments/intents", `{"product":"naming","business_order_id":"biz-1","provider":"xunhu"}`)
	var envelope struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	gateway.event = payment.WebhookEvent{
		Provider: payment.ProviderXunhu,
		Type:     payment.EventPaymentSucceeded,
		Data: payment.WebhookEventData{
			OrderID: envelope.Data.ID, Amount: 990, PaymentID: "pay-ack",
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
	runtime, _ := newPaymentRuntime(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, ServerDeps{PaymentRuntime: runtime}, "test", "test", NewRateLimiter(), NewRateLimiter())
	for _, path := range []string{
		"/api/naming/webhook", "/api/mingshu/webhook", "/api/hepan/webhook",
	} {
		rec := postJSON(t, mux, path, `{}`)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s = %d, want 404", path, rec.Code)
		}
	}
}
