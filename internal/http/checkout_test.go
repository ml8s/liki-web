package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"liki-web/internal/checkout"
	"liki-web/internal/product"
)

func TestMakeCheckout_InvalidJSON(t *testing.T) {
	t.Parallel()
	store := newAuthTestStore(t)
	handler := checkout.MakeCheckout(store, nil, checkout.Config{LogPrefix: "test", Product: product.ProductNaming})

	r := httptest.NewRequest("POST", "/api/naming/checkout", strings.NewReader(`{bad`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestMakeCheckout_MissingEmail(t *testing.T) {
	t.Parallel()
	store := newAuthTestStore(t)
	handler := checkout.MakeCheckout(store, nil, checkout.Config{LogPrefix: "test", Product: product.ProductNaming})

	r := httptest.NewRequest("POST", "/api/naming/checkout", strings.NewReader(`{"order_id":"order-1","email":""}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestMakeCheckout_MissingOrderID(t *testing.T) {
	t.Parallel()
	store := newAuthTestStore(t)
	handler := checkout.MakeCheckout(store, nil, checkout.Config{LogPrefix: "test", Product: product.ProductNaming})

	r := httptest.NewRequest("POST", "/api/naming/checkout", strings.NewReader(`{"order_id":"","email":"user@example.com"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestMakeCheckout_OrderNotFound(t *testing.T) {
	t.Parallel()
	store := newAuthTestStore(t)
	handler := checkout.MakeCheckout(store, nil, checkout.Config{LogPrefix: "test", Product: product.ProductNaming})

	r := httptest.NewRequest("POST", "/api/naming/checkout", strings.NewReader(`{"order_id":"nonexistent","email":"user@example.com"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestMakeCheckout_AlreadyPaid(t *testing.T) {
	t.Parallel()
	store := newAuthTestStore(t)
	seedActiveOrder(t, store, "order-1", "user@example.com", product.ProductNaming, "2027-01-01 00:00:00")
	handler := checkout.MakeCheckout(store, nil, checkout.Config{LogPrefix: "test", Product: product.ProductNaming})

	r := httptest.NewRequest("POST", "/api/naming/checkout", strings.NewReader(`{"order_id":"order-1","email":"user@example.com"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestMakePaymentReturn_MissingID(t *testing.T) {
	t.Parallel()
	store := newAuthTestStore(t)
	handler := checkout.MakePaymentReturn(store, checkout.Config{LogPrefix: "test", ReturnPath: "/test/", RedirectPath: "/test"})

	r := httptest.NewRequest("GET", "/api/payment/test/return?status=succeeded&t=valid", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "/" {
		t.Errorf("redirect = %q, want /", loc)
	}
}

func TestMakePaymentReturn_InvalidToken(t *testing.T) {
	t.Parallel()
	store := newAuthTestStore(t)
	handler := checkout.MakePaymentReturn(store, checkout.Config{LogPrefix: "test", ReturnPath: "/test/", RedirectPath: "/test"})

	r := httptest.NewRequest("GET", "/api/payment/test/return/order-1?status=succeeded&t=invalidtoken", nil)
	r.SetPathValue("id", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "/" {
		t.Errorf("redirect = %q, want /", loc)
	}
}

func TestMakePaymentReturn_MissingStatus(t *testing.T) {
	t.Parallel()
	store := newAuthTestStore(t)
	handler := checkout.MakePaymentReturn(store, checkout.Config{LogPrefix: "test", ReturnPath: "/test/", RedirectPath: "/test"})

	r := httptest.NewRequest("GET", "/api/payment/test/return/order-1?t=any", nil)
	r.SetPathValue("id", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", w.Code)
	}
}

func TestMakePaymentReturn_FailedStatus(t *testing.T) {
	t.Parallel()
	store := newAuthTestStore(t)
	handler := checkout.MakePaymentReturn(store, checkout.Config{LogPrefix: "test", ReturnPath: "/test/", RedirectPath: "/test"})

	r := httptest.NewRequest("GET", "/api/payment/test/return/order-1?status=failed&t=any", nil)
	r.SetPathValue("id", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", w.Code)
	}
}

func TestMakePaymentReturn_NonPendingOrder(t *testing.T) {
	t.Parallel()
	store := newAuthTestStore(t)
	seedActiveOrder(t, store, "order-1", "user@example.com", product.ProductNaming, "2027-01-01 00:00:00")
	handler := checkout.MakePaymentReturn(store, checkout.Config{LogPrefix: "test", ReturnPath: "/test/", RedirectPath: "/test"})

	// Order is already paid (seedActiveOrder marks it paid), so callback should redirect to /
	r2 := httptest.NewRequest("GET", "/api/payment/test/return/order-1?status=succeeded&t=anytoken", nil)
	r2.SetPathValue("id", "order-1")
	w2 := httptest.NewRecorder()
	handler(w2, r2)

	if w2.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", w2.Code)
	}
	if w2.Header().Get("Location") != "/" {
		t.Errorf("redirect = %q, want /", w2.Header().Get("Location"))
	}
}

func TestMakeOrderStatus_OrderNotFound(t *testing.T) {
	t.Parallel()
	store := newAuthTestStore(t)
	handler := checkout.MakeOrderStatus(store)

	r := httptest.NewRequest("GET", "/api/orders/nonexistent/status", nil)
	r.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestMakeOrderStatus_MissingID(t *testing.T) {
	t.Parallel()
	store := newAuthTestStore(t)
	handler := checkout.MakeOrderStatus(store)

	r := httptest.NewRequest("GET", "/api/orders//status", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestMakeOrderStatus_Pending(t *testing.T) {
	t.Parallel()
	store := newAuthTestStore(t)
	if err := store.CreateOrder(context.Background(), "order-1", "naming", product.NamingAmountCents(product.CNY), "CNY", "", "zh-Hans"); err != nil {
		t.Fatalf("create order: %v", err)
	}
	handler := checkout.MakeOrderStatus(store)

	r := httptest.NewRequest("GET", "/api/orders/order-1/status", nil)
	r.SetPathValue("id", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "pending") {
		t.Errorf("body should contain pending status: %s", w.Body.String())
	}
}

func TestMakeOrderStatus_Paid(t *testing.T) {
	t.Parallel()
	store := newAuthTestStore(t)
	seedActiveOrder(t, store, "order-1", "user@example.com", product.ProductNaming, "2027-01-01 00:00:00")
	handler := checkout.MakeOrderStatus(store)

	r := httptest.NewRequest("GET", "/api/orders/order-1/status", nil)
	r.SetPathValue("id", "order-1")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "paid") {
		t.Errorf("body should contain paid status: %s", w.Body.String())
	}
}
