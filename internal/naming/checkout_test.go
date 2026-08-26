package naming

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"liki-web/internal/checkout"
	"liki-web/internal/httputil"
	"liki-web/internal/payment"
	"liki-web/internal/product"
	"liki-web/internal/testutil"
)

func nmSeedOrder(t *testing.T, s *Store) string {
	t.Helper()
	oid := uuid.New().String()
	if err := s.CreateOrder(context.Background(), oid, "naming", 990, "CNY", "test@t.com", "zh-Hans"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return oid
}

func nmSeedPaidOrder(t *testing.T, s *Store) string {
	t.Helper()
	oid := nmSeedOrder(t, s)
	s.MarkPaidIdempotent(context.Background(), oid, "pay-1")         //nolint:errcheck
	s.OverrideChatExpiresAt(context.Background(), oid, "2099-12-31") //nolint:errcheck
	return oid
}

// ── HandleCreateOrder ──

func TestNamingHandleCreateOrder_Success(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	h := HandleCreateOrder(s)

	r := httptest.NewRequest("POST", "/api/naming/orders", strings.NewReader(`{"product":"naming"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Data struct {
			OrderID string `json:"order_id"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if resp.Data.OrderID == "" {
		t.Error("empty order_id")
	}
}

// ── HandleCheckout ──

func TestNamingHandleCheckout_Success(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	oid := nmSeedOrder(t, s)
	mockProv := &testutil.MockPaymentProvider{CreateResult: &payment.CheckoutResult{SessionID: "nm-1", CheckoutURL: "https://pay.example.com/nm"}}
	svc := payment.NewService(mockProv, mockProv, "https://liki.hk")

	cfg := checkout.Config{Product: product.ProductNaming, ProductFn: namingProductFromOrder, ReturnPath: "/api/naming/payments/return/", RedirectPath: "/naming/chat", LogPrefix: "naming:"}
	h := checkout.MakeCheckout(s, svc, cfg)
	body := `{"order_id":"` + oid + `","email":"a@b.co","provider":"dodo"}`
	r := httptest.NewRequest("POST", "/api/naming/checkout", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var env struct {
		Data payment.CheckoutResult `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&env) //nolint:errcheck
	if env.Data.SessionID != "nm-1" {
		t.Errorf("session_id = %q", env.Data.SessionID)
	}
}

func TestNamingHandleCheckout_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	svc := payment.NewService(nil, nil, "https://liki.hk")
	cfg := checkout.Config{Product: product.ProductNaming, ProductFn: namingProductFromOrder, ReturnPath: "/api/naming/payments/return/", RedirectPath: "/naming/chat", LogPrefix: "naming:"}
	h := checkout.MakeCheckout(s, svc, cfg)

	r := httptest.NewRequest("POST", "/api/naming/checkout", strings.NewReader(`{"order_id":"no-such","email":"a@b.co"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ── HandlePaymentReturn ──

func TestNamingHandlePaymentReturn_Success(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	oid := nmSeedOrder(t, s)
	tok := httputil.ReturnToken("naming", oid)

	cfg := checkout.Config{Product: product.ProductNaming, ProductFn: namingProductFromOrder, ReturnPath: "/api/naming/payments/return/", RedirectPath: "/naming/chat", LogPrefix: "naming:"}
	h := checkout.MakePaymentReturn(s, cfg)
	r := httptest.NewRequest("GET", "/api/naming/payments/return/"+oid+"?status=succeeded&t="+tok, nil)
	r.SetPathValue("id", oid)
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "/naming/chat") {
		t.Errorf("redirect = %q, want /naming/chat", loc)
	}
}

func TestNamingHandlePaymentReturn_BadToken(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	oid := nmSeedOrder(t, s)
	cfg := checkout.Config{Product: product.ProductNaming, ProductFn: namingProductFromOrder, ReturnPath: "/api/naming/payments/return/", RedirectPath: "/naming/chat", LogPrefix: "naming:"}
	h := checkout.MakePaymentReturn(s, cfg)

	r := httptest.NewRequest("GET", "/api/naming/payments/return/"+oid+"?status=succeeded&t=bad", nil)
	r.SetPathValue("id", oid)
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", w.Code)
	}
	o, _ := s.GetOrder(context.Background(), oid) //nolint:errcheck
	if o.Status != "pending" {
		t.Errorf("status = %q, want pending", o.Status)
	}
}

// ── HandleOrderStatus ──

func TestNamingHandleOrderStatus_Valid(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	oid := nmSeedOrder(t, s)
	h := checkout.MakeOrderStatus(s)

	r := httptest.NewRequest("GET", "/api/naming/orders/"+oid+"/status", nil)
	r.SetPathValue("id", oid)
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if resp.Data.Status != "pending" {
		t.Errorf("status = %q", resp.Data.Status)
	}
}

func TestNamingHandleOrderStatus_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	h := checkout.MakeOrderStatus(s)

	r := httptest.NewRequest("GET", "/api/naming/orders/no-such/status", nil)
	r.SetPathValue("id", "no-such")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ── HandleRetryOrder ──

func TestNamingHandleRetryOrder_Success(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	oid := nmSeedPaidOrder(t, s)
	mockProv := &testutil.MockPaymentProvider{CreateResult: &payment.CheckoutResult{SessionID: "retry-1", CheckoutURL: "https://pay.example.com/retry"}}
	svc := payment.NewService(mockProv, mockProv, "https://liki.hk")

	h := HandleRetryOrder(s, svc)
	r := httptest.NewRequest("POST", "/api/naming/orders/"+oid+"/retry", nil)
	r.SetPathValue("id", oid)
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// ── HandleWebhook ──

func TestNamingHandleWebhook_Success(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	oid := nmSeedOrder(t, s)
	mockProv := &testutil.MockPaymentProvider{
		VerifyEvent: &payment.WebhookEvent{Type: payment.EventPaymentSucceeded, Data: payment.WebhookEventData{OrderID: oid, PaymentID: "pay-1"}},
	}
	svc := payment.NewService(mockProv, mockProv, "https://liki.hk")

	h := payment.HandleWebhook(s, svc, "naming", nil, nil, nil)
	r := httptest.NewRequest("POST", "/api/naming/webhook", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	o, _ := s.GetOrder(context.Background(), oid) //nolint:errcheck
	if o.Status != "paid" {
		t.Errorf("status = %q, want paid", o.Status)
	}
}

func TestNamingHandleWebhook_BadSignature(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mockProv := &testutil.MockPaymentProvider{VerifyErr: errors.New("bad sig")}
	svc := payment.NewService(mockProv, mockProv, "https://liki.hk")

	h := payment.HandleWebhook(s, svc, "naming", nil, nil, nil)
	r := httptest.NewRequest("POST", "/api/naming/webhook", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// ── HandleRedeemCode ──

func TestNamingHandleRedeemCode_Success(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	s.GenerateVerificationCode(context.Background(), "admin") //nolint:errcheck

	// List codes to find the generated one
	codes, _ := s.ListVerificationCodes(context.Background()) //nolint:errcheck
	if len(codes) == 0 {
		t.Fatal("no codes generated")
	}
	code := codes[0].Code

	h := HandleRedeemCode(s)
	body := `{"code":"` + code + `","email":"user@t.com"}`
	r := httptest.NewRequest("POST", "/api/naming/verification-codes/redeem", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			OrderID string `json:"order_id"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if resp.Data.OrderID == "" {
		t.Error("empty order_id")
	}
}

func TestNamingHandleRedeemCode_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	h := HandleRedeemCode(s)

	r := httptest.NewRequest("POST", "/api/naming/verification-codes/redeem", strings.NewReader(`{"code":"INVALID","email":"a@b.co"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
