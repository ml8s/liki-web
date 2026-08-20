package mingshu

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

func lbSeedOrder(t *testing.T, s *Store) string {
	t.Helper()
	oid := uuid.New().String()
	birth := "张三 | 1990-05-20 12:00 | 上海 | male"
	if err := s.CreateOrder(context.Background(), oid, "mingshu", 990, "CNY", "test@t.com", birth, "zh-Hans"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return oid
}

func lbSeedPaidOrder(t *testing.T, s *Store) string {
	t.Helper()
	oid := lbSeedOrder(t, s)
	s.MarkPaidIdempotent(context.Background(), oid, "pay-1") //nolint:errcheck
	s.SaveReport(context.Background(), oid, `{"content":"test report"}`) //nolint:errcheck
	return oid
}

// ── HandleCreateOrder ──

func TestHandleCreateOrder_Success(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	h := HandleCreateOrder(s)

	body := `{"birth":{"name":"张三","year":1990,"month":5,"day":20,"hour":12,"minute":0,"city":"上海","gender":"male"},"email":"a@b.co"}`
	r := httptest.NewRequest("POST", "/api/mingshu/orders", strings.NewReader(body))
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

func TestHandleCreateOrder_InvalidJSON(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	h := HandleCreateOrder(s)

	r := httptest.NewRequest("POST", "/api/mingshu/orders", strings.NewReader(`{invalid`))
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// ── HandleCheckout ──

func TestLifebookHandleCheckout_Success(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := lbSeedOrder(t, s)
	mockProv := &testutil.MockPaymentProvider{CreateResult: &payment.CheckoutResult{SessionID: "lb-1", CheckoutURL: "https://pay.example.com/lb"}}
	svc := payment.NewService(mockProv, mockProv, "https://liki.hk")

	cfg := checkout.Config{Product: product.ProductMingShu, ReturnPath: "/api/mingshu/payments/return/", RedirectPath: "/mingshu/report", LogPrefix: "mingshu:"}
	h := checkout.MakeCheckout(s, svc, cfg)
	body := `{"order_id":"` + oid + `","email":"a@b.co","provider":"dodo"}`
	r := httptest.NewRequest("POST", "/api/mingshu/checkout", strings.NewReader(body))
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
	if env.Data.SessionID != "lb-1" {
		t.Errorf("session_id = %q", env.Data.SessionID)
	}
}

func TestLifebookHandleCheckout_NotFound(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	svc := payment.NewService(nil, nil, "https://liki.hk")
	cfg := checkout.Config{Product: product.ProductMingShu, ReturnPath: "/api/mingshu/payments/return/", RedirectPath: "/mingshu/report", LogPrefix: "mingshu:"}
	h := checkout.MakeCheckout(s, svc, cfg)

	r := httptest.NewRequest("POST", "/api/mingshu/checkout", strings.NewReader(`{"order_id":"no-such","email":"a@b.co"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ── HandlePaymentReturn ──

func TestLifebookHandlePaymentReturn_Success(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := lbSeedOrder(t, s)
	tok := httputil.ReturnToken("lifebook", oid)

	cfg := checkout.Config{Product: product.ProductMingShu, ReturnPath: "/api/mingshu/payments/return/", RedirectPath: "/mingshu/report", LogPrefix: "mingshu:"}
	h := checkout.MakePaymentReturn(s, cfg)
	r := httptest.NewRequest("GET", "/api/mingshu/payments/return/"+oid+"?status=succeeded&t="+tok, nil)
	r.SetPathValue("id", oid)
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "/mingshu") {
		t.Errorf("redirect = %q, want /mingshu", loc)
	}
}

func TestLifebookHandlePaymentReturn_BadToken(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := lbSeedOrder(t, s)

	cfg := checkout.Config{Product: product.ProductMingShu, ReturnPath: "/api/mingshu/payments/return/", RedirectPath: "/mingshu/report", LogPrefix: "mingshu:"}
	h := checkout.MakePaymentReturn(s, cfg)
	r := httptest.NewRequest("GET", "/api/mingshu/payments/return/"+oid+"?status=succeeded&t=bad", nil)
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

func TestLifebookHandleOrderStatus_Valid(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := lbSeedOrder(t, s)

	h := checkout.MakeOrderStatus(s)
	r := httptest.NewRequest("GET", "/api/mingshu/orders/"+oid+"/status", nil)
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

func TestLifebookHandleOrderStatus_NotFound(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	h := checkout.MakeOrderStatus(s)

	r := httptest.NewRequest("GET", "/api/mingshu/orders/no-such/status", nil)
	r.SetPathValue("id", "no-such")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ── HandleGetReport ──

func TestLifebookHandleGetReport_Success(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := lbSeedPaidOrder(t, s)

	h := HandleGetReport(s)
	r := httptest.NewRequest("GET", "/api/mingshu/orders/"+oid+"/report", nil)
	r.SetPathValue("id", oid)
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.Contains(resp.Data.Content, "test report") {
		t.Errorf("content = %q, want 'test report'", resp.Data.Content)
	}
}

func TestLifebookHandleGetReport_NotFound(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	h := HandleGetReport(s)

	r := httptest.NewRequest("GET", "/api/mingshu/orders/no-such/report", nil)
	r.SetPathValue("id", "no-such")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ── HandleWebhook ──

func TestLifebookHandleWebhook_Success(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := lbSeedOrder(t, s)
	mockProv := &testutil.MockPaymentProvider{
		VerifyEvent: &payment.WebhookEvent{Type: payment.EventPaymentSucceeded, Data: payment.WebhookEventData{OrderID: oid, PaymentID: "pay-1"}},
	}
	svc := payment.NewService(mockProv, mockProv, "https://liki.hk")

	h := webhookHandler(s, svc, "mingshu", nil, nil, nil)
	r := httptest.NewRequest("POST", "/api/mingshu/webhook", strings.NewReader(`{}`))
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

func TestLifebookHandleWebhook_BadSignature(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	mockProv := &testutil.MockPaymentProvider{VerifyErr: errors.New("bad sig")}
	svc := payment.NewService(mockProv, mockProv, "https://liki.hk")

	h := webhookHandler(s, svc, "mingshu", nil, nil, nil)
	r := httptest.NewRequest("POST", "/api/mingshu/webhook", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

