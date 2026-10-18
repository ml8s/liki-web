package sales

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

type fakeOrderGateway struct{}

func (g *fakeOrderGateway) CreateCheckout(context.Context, payment.CheckoutRequest) (*payment.CheckoutResult, error) {
	return &payment.CheckoutResult{
		CheckoutSessionID: "session-1",
		CheckoutURL:       "https://pay.example/checkout",
		QRCodeURL:         "https://pay.example/qr.png",
	}, nil
}

func (g *fakeOrderGateway) VerifyWebhook([]byte, http.Header) (*payment.WebhookEvent, error) {
	return nil, nil
}

func (g *fakeOrderGateway) FetchPayment(context.Context, payment.PaymentIntent) (*payment.WebhookEvent, error) {
	return nil, nil
}

func (g *fakeOrderGateway) WebhookAck() []byte { return []byte("ok") }

type fakeOrderFlow struct{}

func (f *fakeOrderFlow) Product() string           { return string(product.Chat) }
func (f *fakeOrderFlow) DisplayName(string) string { return "Liki" }
func (f *fakeOrderFlow) PrepareCheckout(_ context.Context, businessOrderID string) (payment.BusinessOrder, error) {
	return payment.BusinessOrder{
		ID: businessOrderID, UserID: "user-1", Product: string(product.Chat),
		AmountMinor: 7900, Currency: "CNY", Email: "user@example.com", Locale: "zh-Hant",
	}, nil
}
func (f *fakeOrderFlow) OnPaid(context.Context, payment.PaymentIntent) error { return nil }
func (f *fakeOrderFlow) SuccessRedirect(payment.PaymentIntent) string        { return "/app/chat" }
func (f *fakeOrderFlow) FailureRedirect(payment.PaymentIntent) string        { return "/app" }

// newOrderActionsHandler builds a handler whose only live dependency is one
// xunhu-routed payment intent, mirroring the real order lifecycle closely
// enough to exercise cancel/resume mapping.
func newOrderActionsHandler(t *testing.T) (*Handler, *payment.IntentStore, Order, payment.PaymentIntent) {
	t.Helper()
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	orders := NewOrderStoreWithCatalog(db, product.NewCatalog(product.CatalogConfig{
		OneTimePriceMinorCNY: 7900,
		OneTimePriceMinorUSD: 2990,
	}))
	intents := payment.NewIntentStore(db)
	for _, migrate := range []func(context.Context) error{orders.Migrate, intents.Migrate} {
		if err := migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	runtime := payment.NewRuntime(
		map[payment.Provider]payment.Gateway{
			payment.ProviderXunhu: &fakeOrderGateway{},
			payment.ProviderDodo:  &fakeOrderGateway{},
		},
		map[string]payment.ProductFlow{string(product.Chat): &fakeOrderFlow{}},
		intents,
		payment.NewNotificationStore(db),
		payment.NewPaymentCompletionTaskStore(db),
		payment.NewPaymentEventStore(db),
		nil,
		payment.RuntimeOptions{BaseURL: "https://liki.hk"},
	)
	order, err := orders.CreateOrder(ctx, "user-1", "user@example.com", product.Chat, "CNY")
	if err != nil {
		t.Fatal(err)
	}
	intent, err := runtime.CreateIntent(ctx, payment.CreateIntentInput{
		Product: string(product.Chat), BusinessOrderID: order.ID, UserID: "user-1", Locale: "zh-Hant",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := orders.AttachIntent(ctx, "user-1", order.ID, intent.ID); err != nil {
		t.Fatal(err)
	}
	return NewHandler(orders, runtime, nil, nil), intents, order, intent
}

func orderActionRequest(t *testing.T, method, orderID, userID string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(method, "/api/sales/orders/"+orderID, nil)
	request.SetPathValue("id", orderID)
	return request.WithContext(auth.WithAuth(request.Context(), auth.User{ID: userID}, auth.AuthSession{}, ""))
}

func problemType(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var problem struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	return problem.Type
}

func TestCancelOrderMovesPendingToCanceled(t *testing.T) {
	handler, _, order, _ := newOrderActionsHandler(t)
	ctx := context.Background()

	recorder := httptest.NewRecorder()
	handler.HandleCancelOrder(recorder, orderActionRequest(t, http.MethodPost, order.ID, "user-1"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	canceled, err := handler.orders.GetOrderForUser(ctx, "user-1", order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if canceled.Status != OrderCanceled {
		t.Fatalf("status = %s, want canceled", canceled.Status)
	}

	recorder = httptest.NewRecorder()
	handler.HandleCancelOrder(recorder, orderActionRequest(t, http.MethodPost, order.ID, "user-1"))
	if recorder.Code != http.StatusConflict || problemType(t, recorder) != "order_not_pending" {
		t.Fatalf("second cancel = %d %s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.HandleCancelOrder(recorder, orderActionRequest(t, http.MethodPost, order.ID, "user-2"))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("foreign cancel = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestResumeCheckoutReopensPaymentForPendingOrder(t *testing.T) {
	handler, _, order, _ := newOrderActionsHandler(t)

	recorder := httptest.NewRecorder()
	handler.HandleResumeCheckout(recorder, orderActionRequest(t, http.MethodPost, order.ID, "user-1"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			OrderID     string `json:"order_id"`
			CheckoutURL string `json:"checkout_url"`
			QRCodeURL   string `json:"qr_code_url"`
			Status      string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.OrderID != order.ID ||
		response.Data.CheckoutURL != "https://pay.example/checkout" ||
		response.Data.QRCodeURL != "https://pay.example/qr.png" ||
		response.Data.Status != string(payment.IntentCheckoutCreated) {
		t.Fatalf("response = %+v", response.Data)
	}
}

func TestResumeCheckoutRejectsExpiredIntent(t *testing.T) {
	handler, intents, order, _ := newOrderActionsHandler(t)
	ctx := context.Background()
	if n, err := intents.MarkStaleIntentsExpired(ctx, -time.Second); err != nil || n != 1 {
		t.Fatalf("expire stale = %d, %v", n, err)
	}

	recorder := httptest.NewRecorder()
	handler.HandleResumeCheckout(recorder, orderActionRequest(t, http.MethodPost, order.ID, "user-1"))
	if recorder.Code != http.StatusConflict || problemType(t, recorder) != "order_expired" {
		t.Fatalf("resume expired = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestResumeCheckoutRejectsPaidIntentRace(t *testing.T) {
	handler, intents, order, intent := newOrderActionsHandler(t)
	ctx := context.Background()
	// Simulate webhook arrived but the order row has not been updated yet.
	if _, _, err := intents.MarkPaid(ctx, intent.ID, "pay-1"); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler.HandleResumeCheckout(recorder, orderActionRequest(t, http.MethodPost, order.ID, "user-1"))
	if recorder.Code != http.StatusConflict || problemType(t, recorder) != "order_already_paid" {
		t.Fatalf("resume paid race = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestOrderDetailReturnsPaymentFacts(t *testing.T) {
	handler, intents, order, intent := newOrderActionsHandler(t)
	ctx := context.Background()
	if _, _, err := intents.MarkPaid(ctx, intent.ID, "pay-1"); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/sales/orders/"+order.ID, nil)
	request.SetPathValue("id", order.ID)
	request = request.WithContext(auth.WithAuth(request.Context(), auth.User{ID: "user-1"}, auth.AuthSession{}, ""))
	recorder := httptest.NewRecorder()
	handler.HandleOrderDetail(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			Order struct {
				ID           string `json:"id"`
				IntentStatus string `json:"intent_status"`
			} `json:"order"`
			Payment *struct {
				Provider string `json:"provider"`
				Status   string `json:"status"`
			} `json:"payment"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Order.ID != order.ID || response.Data.Order.IntentStatus != string(payment.IntentPaid) {
		t.Fatalf("order = %+v", response.Data.Order)
	}
	if response.Data.Payment == nil ||
		response.Data.Payment.Provider != string(payment.ProviderXunhu) ||
		response.Data.Payment.Status != string(payment.IntentPaid) {
		t.Fatalf("payment = %+v", response.Data.Payment)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/sales/orders/"+order.ID, nil)
	request.SetPathValue("id", order.ID)
	request = request.WithContext(auth.WithAuth(request.Context(), auth.User{ID: "user-2"}, auth.AuthSession{}, ""))
	recorder = httptest.NewRecorder()
	handler.HandleOrderDetail(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("foreign detail = %d", recorder.Code)
	}
}

func TestOrdersListAnnotatesIntentStatus(t *testing.T) {
	handler, intents, order, intent := newOrderActionsHandler(t)
	ctx := context.Background()
	if _, _, err := intents.MarkPaid(ctx, intent.ID, "pay-1"); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/sales/orders", nil)
	request = request.WithContext(auth.WithAuth(request.Context(), auth.User{ID: "user-1"}, auth.AuthSession{}, ""))
	recorder := httptest.NewRecorder()
	handler.HandleOrders(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			Orders []struct {
				ID           string `json:"id"`
				IntentStatus string `json:"intent_status"`
			} `json:"orders"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data.Orders) != 1 ||
		response.Data.Orders[0].ID != order.ID ||
		response.Data.Orders[0].IntentStatus != string(payment.IntentPaid) {
		t.Fatalf("orders = %+v", response.Data.Orders)
	}
}
