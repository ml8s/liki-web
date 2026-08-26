package checkout

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"liki-web/internal/httputil"
	"liki-web/internal/payment"
	"liki-web/internal/product"
)

// mockCheckoutProvider satisfies payment.Provider (exported in export_test.go).
type mockCheckoutProvider struct{}

func (m *mockCheckoutProvider) CreateCheckout(_ context.Context, _ product.Product, _ int, orderID, _, _ string) (*payment.CheckoutResult, error) {
	return &payment.CheckoutResult{
		SessionID:   "sess_" + orderID,
		CheckoutURL: "https://pay.test/" + orderID,
	}, nil
}

func (m *mockCheckoutProvider) VerifyWebhook(_ []byte, _ http.Header) (*payment.WebhookEvent, error) {
	return &payment.WebhookEvent{
		Type: "payment.succeeded",
		Data: payment.WebhookEventData{OrderID: "oid-1", PaymentID: "pay-1"},
	}, nil
}

// mockOrderStore implements product.OrderStore.
type mockOrderStore struct {
	orders map[string]*product.CheckoutOrder
}

func (m *mockOrderStore) GetCheckoutOrder(_ context.Context, id string) (*product.CheckoutOrder, error) {
	o, ok := m.orders[id]
	if !ok {
		return nil, httputil.ErrOrderNotFound
	}
	return o, nil
}
func (m *mockOrderStore) UpdateEmail(_ context.Context, _, _ string) error    { return nil }
func (m *mockOrderStore) UpdateProvider(_ context.Context, _, _ string) error { return nil }
func (m *mockOrderStore) MarkPaidViaCallback(_ context.Context, _ string) (bool, string, error) {
	return true, "pay-mock", nil
}

func seedOrder(store *mockOrderStore, id, productType, status string) {
	if store.orders == nil {
		store.orders = map[string]*product.CheckoutOrder{}
	}
	store.orders[id] = &product.CheckoutOrder{
		Status:  status,
		Amount:  4999,
		Product: productType,
	}
}

func TestConfig_ResolveProduct_Exact(t *testing.T) {
	cfg := Config{Product: product.Product("naming")}
	got := cfg.resolveProduct(&product.CheckoutOrder{Product: "naming"})
	if got != product.Product("naming") {
		t.Errorf("got %q, want naming", got)
	}
}

func TestConfig_ResolveProduct_Empty(t *testing.T) {
	cfg := Config{Product: product.Product("hepan")}
	got := cfg.resolveProduct(&product.CheckoutOrder{Product: ""})
	if got != product.Product("hepan") {
		t.Errorf("got %q, want hepan", got)
	}
}

func TestMakeCheckout_OrderNotFound(t *testing.T) {
	store := &mockOrderStore{}
	dodo := &mockCheckoutProvider{}
	xunhu := &mockCheckoutProvider{}
	svc := payment.NewService(dodo, xunhu, "https://liki.hk/api")

	h := MakeCheckout(store, svc, Config{Product: product.Product("hepan"), ReturnPath: "/api/hepan/payments/return/", LogPrefix: "test:"})
	r := httptest.NewRequest("POST", "/", bodyReader(`{"order_id":"no-such","email":"t@t.com"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404, body: %s", w.Code, w.Body.String())
	}
}

func TestMakeCheckout_Success(t *testing.T) {
	store := &mockOrderStore{}
	seedOrder(store, "oid-1", "hepan", "pending")
	dodo := &mockCheckoutProvider{}
	xunhu := &mockCheckoutProvider{}
	svc := payment.NewService(dodo, xunhu, "https://liki.hk/api")

	h := MakeCheckout(store, svc, Config{Product: product.Product("hepan"), ReturnPath: "/api/hepan/payments/return/", LogPrefix: "test:"})
	r := httptest.NewRequest("POST", "/", bodyReader(`{"order_id":"oid-1","email":"t@t.com","provider":"dodo"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
}

func TestMakePaymentReturn_Redirects(t *testing.T) {
	store := &mockOrderStore{}
	seedOrder(store, "oid-2", "mingshu", "paid")
	h := MakePaymentReturn(store, Config{Product: product.Product("mingshu"), ReturnPath: "/api/mingshu/payments/return/", RedirectPath: "/mingshu/report", LogPrefix: "ms:"})
	token := httputil.ReturnToken("mingshu", "oid-2")
	r := httptest.NewRequest("GET", "/api/mingshu/payments/return/oid-2?status=succeeded&t="+token, nil)
	r.SetPathValue("id", "oid-2")
	w := httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", w.Code)
	}
}

func TestMakePaymentReturn_BadToken(t *testing.T) {
	store := &mockOrderStore{}
	seedOrder(store, "oid-3", "hepan", "pending")
	h := MakePaymentReturn(store, Config{Product: product.Product("hepan"), ReturnPath: "/api/hepan/payments/return/", LogPrefix: "hp:"})
	r := httptest.NewRequest("GET", "/api/hepan/payments/return/oid-3?status=succeeded&t=bad", nil)
	r.SetPathValue("id", "oid-3")
	w := httptest.NewRecorder()
	h(w, r)
	// Bad token redirects to root, not 404
	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", w.Code)
	}
}

func TestMakeOrderStatus_Found(t *testing.T) {
	store := &mockOrderStore{}
	seedOrder(store, "oid-4", "hepan", "paid")
	h := MakeOrderStatus(store)
	r := httptest.NewRequest("GET", "/api/orders/oid-4/status", nil)
	r.SetPathValue("id", "oid-4")
	w := httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestMakeOrderStatus_NotFound(t *testing.T) {
	store := &mockOrderStore{}
	h := MakeOrderStatus(store)
	r := httptest.NewRequest("GET", "/api/orders/no-such/status", nil)
	r.SetPathValue("id", "no-such")
	w := httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func bodyReader(s string) *strings.Reader { return strings.NewReader(s) }
