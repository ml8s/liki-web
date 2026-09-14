package commerce

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"liki-web/internal/auth"
	"liki-web/internal/entitlement"
	"liki-web/internal/payment"
)

type fakeGateway struct{}

func (fakeGateway) CreateCheckout(context.Context, payment.CheckoutRequest) (*payment.CheckoutResult, error) {
	return &payment.CheckoutResult{SessionID: "session", CheckoutURL: "https://pay.example/checkout", StatusToken: "token"}, nil
}
func (fakeGateway) VerifyWebhook([]byte, http.Header) (*payment.WebhookEvent, error) { return nil, nil }
func (fakeGateway) FetchPayment(context.Context, payment.PaymentIntent) (*payment.WebhookEvent, error) {
	return nil, nil
}
func (fakeGateway) WebhookAck() []byte { return []byte("ok") }

func newHandlerFixture(t *testing.T) (*Handler, *entitlement.Store, string) {
	t.Helper()
	db := openTestDB(t)
	orders := NewStore(db)
	entitlements := entitlement.NewStore(db)
	for _, migrate := range []func(context.Context) error{
		payment.NewIntentStore(db).Migrate,
		payment.NewNotificationStore(db).Migrate,
		payment.NewFulfillmentTaskStore(db).Migrate,
		payment.NewPaymentEventStore(db).Migrate,
	} {
		if err := migrate(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	runtime := payment.NewRuntime(
		map[payment.Provider]payment.Gateway{payment.ProviderXunhu: fakeGateway{}},
		map[string]payment.ProductFlow{string(ProductChat): NewFulfiller(orders, entitlements, ProductChat, 24*time.Hour)},
		payment.NewIntentStore(db), payment.NewNotificationStore(db),
		payment.NewFulfillmentTaskStore(db), payment.NewPaymentEventStore(db),
		fakeSender{}, payment.RuntimeOptions{BaseURL: "https://liki.hk"},
	)
	handler := NewHandler(orders, entitlements, runtime, func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, _ := auth.UserFromContext(r.Context())
			r = r.WithContext(auth.WithAuth(r.Context(), user, auth.AuthSession{}, ""))
			next(w, r)
		}
	})
	return handler, entitlements, "user_1"
}

type fakeSender struct{}

func (fakeSender) Send(context.Context, payment.EmailMessage) error { return nil }

func TestChatCommerceAPI(t *testing.T) {
	handler, _, userID := newHandlerFixture(t)
	user := auth.User{ID: userID, Email: "user@example.com", Status: auth.UserActive}
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, nil)

	request := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("CF-IPCountry", "CN")
		req = req.WithContext(auth.WithAuth(req.Context(), user, auth.AuthSession{}, ""))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	products := request(http.MethodGet, "/api/chat/products", "")
	if products.Code != http.StatusOK || !strings.Contains(products.Body.String(), `"liki-chat"`) {
		t.Fatalf("products = %d %s", products.Code, products.Body.String())
	}
	checkout := request(http.MethodPost, "/api/chat/products/liki-chat/checkout", `{}`)
	if checkout.Code != http.StatusOK || !strings.Contains(checkout.Body.String(), `"checkout_url":"https://pay.example/checkout"`) {
		t.Fatalf("checkout = %d %s", checkout.Code, checkout.Body.String())
	}
	var result struct {
		Data struct {
			OrderID string `json:"order_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(checkout.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Data.OrderID == "" {
		t.Fatal("order id missing")
	}
	orders := request(http.MethodGet, "/api/chat/orders", "")
	if orders.Code != http.StatusOK || !strings.Contains(orders.Body.String(), `"status":"pending"`) {
		t.Fatalf("orders = %d %s", orders.Code, orders.Body.String())
	}
}
