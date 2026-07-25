package hepan

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"liki-web/internal/checkout"
	"liki-web/internal/httputil"
	"liki-web/internal/payment"
	"liki-web/internal/product"
	"liki-web/internal/testutil"
)

func newTestHandlerStore(t *testing.T) *Store {
	t.Helper()
	return newTestStore(t)
}

// ── HandleCreateOrder ──

func TestHandleCreateOrder_Success(t *testing.T) {
	t.Parallel()
	store := newTestHandlerStore(t)
	h := HandleCreateOrder(store)

	body := `{"person_a":{"name":"张三","birth":"1990-05-20","city":"上海","gender":"male"},"person_b":{"name":"李四","birth":"1992-08-15","city":"北京","gender":"female"}}`
	r := httptest.NewRequest("POST", "/api/hepan/orders", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			OrderID string `json:"order_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.OrderID == "" {
		t.Fatal("empty order_id")
	}
}

func TestHandleCreateOrder_InvalidJSON(t *testing.T) {
	t.Parallel()
	store := newTestHandlerStore(t)
	h := HandleCreateOrder(store)

	r := httptest.NewRequest("POST", "/api/hepan/orders", strings.NewReader(`{invalid`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// ── HandleOrderStatus ──

func TestHandleOrderStatus_Valid(t *testing.T) {
	t.Parallel()
	store := newTestHandlerStore(t)
	orderID := seedOrder(t, store)

	h := checkout.MakeOrderStatus(store)
	r := httptest.NewRequest("GET", "/api/hepan/orders/"+orderID+"/status", nil)
	r.SetPathValue("id", orderID)
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
		t.Errorf("status = %q, want pending", resp.Data.Status)
	}
}

func TestHandleOrderStatus_NotFound(t *testing.T) {
	t.Parallel()
	store := newTestHandlerStore(t)
	h := checkout.MakeOrderStatus(store)

	r := httptest.NewRequest("GET", "/api/hepan/orders/nonexistent/status", nil)
	r.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleOrderStatus_MissingID(t *testing.T) {
	t.Parallel()
	store := newTestHandlerStore(t)
	h := checkout.MakeOrderStatus(store)

	r := httptest.NewRequest("GET", "/api/hepan/orders//status", nil)
	r.SetPathValue("id", "")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// ── HandleGetReport ──

func TestHandleGetReport_Success(t *testing.T) {
	t.Parallel()
	store := newTestHandlerStore(t)
	orderID := seedOrder(t, store)

	report := `{"content":"合盘报告内容"}`
	store.SaveReport(context.Background(), orderID, report) //nolint:errcheck

	h := HandleGetReport(store)
	r := httptest.NewRequest("GET", "/api/hepan/"+orderID, nil)
	r.SetPathValue("id", orderID)
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if !strings.Contains(resp.Data.Content, "合盘报告") {
		t.Errorf("content = %q", resp.Data.Content)
	}
}

func TestHandleGetReport_NotFound(t *testing.T) {
	t.Parallel()
	store := newTestHandlerStore(t)
	h := HandleGetReport(store)

	r := httptest.NewRequest("GET", "/api/hepan/nonexistent", nil)
	r.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleGetReport_EmptyID(t *testing.T) {
	t.Parallel()
	store := newTestHandlerStore(t)
	h := HandleGetReport(store)

	r := httptest.NewRequest("GET", "/api/hepan/", nil)
	r.SetPathValue("id", "")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ── HandleCheckout (limited — needs payment.Service) ──
// Full checkout tests require a payment.Service with mock providers.
// These are covered by internal/http/payment_test.go for the generic flow.

// ── helpers ──

func TestHepanAmount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		curr     string
		wantAmt  int
		wantCurr string
	}{
		{"USD", 2990, "USD"},
		{"usd", 2990, "USD"},
		{"CNY", 19900, "CNY"},
		{"cny", 19900, "CNY"},
		{"", 19900, "CNY"},
		{"HKD", 19900, "CNY"},
	}
	for _, tc := range tests {
		amt, curr := hepanAmount(tc.curr)
		if amt != tc.wantAmt || curr != tc.wantCurr {
			t.Errorf("hepanAmount(%q) = (%d, %q), want (%d, %q)", tc.curr, amt, curr, tc.wantAmt, tc.wantCurr)
		}
	}
}

func TestDetectCurrency(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cf, geo, want string
	}{
		{"CN", "", "CNY"},
		{"", "CN", "CNY"},
		{"US", "", "USD"},
		{"", "US", "USD"},
		{"", "", "CNY"},
	}
	for _, tc := range tests {
		got := string(product.DetectCurrency(tc.cf, tc.geo))
		if got != tc.want {
			t.Errorf("DetectCurrency(%q, %q) = %q, want %q", tc.cf, tc.geo, got, tc.want)
		}
	}
}

func TestNewOrderID_IsUnique(t *testing.T) {
	t.Parallel()
	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id := product.NewOrderID()
		if ids[id] {
			t.Fatal("duplicate order ID")
		}
		ids[id] = true
	}
}

func TestReturnToken_Validates(t *testing.T) {
	t.Parallel()
	oid := product.NewOrderID()
	tok := httputil.ReturnToken("hepan", oid)
	if tok == "" {
		t.Fatal("empty token")
	}
	if !httputil.VerifyReturnToken("hepan", oid, tok) {
		t.Error("verify failed for valid token")
	}
	if httputil.VerifyReturnToken("hepan", oid, "wrong") {
		t.Error("verify passed for wrong token")
	}
	if httputil.VerifyReturnToken("hepan", oid, "") {
		t.Error("verify passed for empty token")
	}
}

func TestLocaleFromAccept(t *testing.T) {
	t.Parallel()
	tests := []struct {
		accept string
		want   string
	}{
		{"zh-CN,zh;q=0.9", "zh-Hans"},
		{"zh-Hans,zh;q=0.9", "zh-Hans"},
		{"zh-TW,zh;q=0.9", "zh-Hant"},
		{"zh-HK,zh;q=0.9", "zh-Hant"},
		{"zh,en;q=0.9", "zh-Hant"},
		{"en,zh;q=0.9", "en"},
		{"fr;q=0.9", "en"},
		{"", "en"},
	}
	for _, tc := range tests {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Accept-Language", tc.accept)
		got := httputil.LocaleFromAccept(r)
		if got != tc.want {
			t.Errorf("httputil.LocaleFromAccept(%q) = %q, want %q", tc.accept, got, tc.want)
		}
	}
}

// ── HandleCheckout ──

func TestHandleCheckout_Success(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	oid := seedOrder(t, store)
	mockProv := &testutil.MockPaymentProvider{
		CreateResult: &payment.CheckoutResult{SessionID: "sess-1", CheckoutURL: "https://pay.example.com/checkout"},
	}
	svc := payment.NewService(mockProv, mockProv, "https://liki.hk")

	cfg := checkout.Config{Product: ProductHepan, ReturnPath: "/api/hepan/payments/return/", RedirectPath: "/hepan/report", LogPrefix: "hepan:"}
	h := checkout.MakeCheckout(store, svc, cfg)
	body := `{"order_id":"` + oid + `","email":"a@b.co","provider":"dodo"}`
	r := httptest.NewRequest("POST", "/api/hepan/checkout", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	var env struct {
		Data payment.CheckoutResult `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Data.SessionID != "sess-1" {
		t.Errorf("session_id = %q", env.Data.SessionID)
	}
}

func TestHandleCheckout_MissingFields(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := payment.NewService(nil, nil, "https://liki.hk")
	cfg := checkout.Config{Product: ProductHepan, ReturnPath: "/api/hepan/payments/return/", RedirectPath: "/hepan/report", LogPrefix: "hepan:"}
	h := checkout.MakeCheckout(store, svc, cfg)

	for _, body := range []string{`{}`, `{"order_id":"x"}`, `{"email":"a@b.co"}`} {
		r := httptest.NewRequest("POST", "/api/hepan/checkout", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h(w, r)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body=%q status=%d want 400", body, w.Code)
		}
	}
}

func TestHandleCheckout_OrderNotFound(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := payment.NewService(nil, nil, "https://liki.hk")
	cfg := checkout.Config{Product: ProductHepan, ReturnPath: "/api/hepan/payments/return/", RedirectPath: "/hepan/report", LogPrefix: "hepan:"}
	h := checkout.MakeCheckout(store, svc, cfg)

	body := `{"order_id":"nonexistent","email":"a@b.co","provider":"dodo"}`
	r := httptest.NewRequest("POST", "/api/hepan/checkout", strings.NewReader(body))
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ── HandleWebhook ──

func TestHandleWebhook_Success(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	oid := seedOrder(t, store)
	mockProv := &testutil.MockPaymentProvider{
		VerifyEvent: &payment.WebhookEvent{
			Type: payment.EventPaymentSucceeded,
			Data: payment.WebhookEventData{OrderID: oid, PaymentID: "pay-1"},
		},
	}
	svc := payment.NewService(mockProv, mockProv, "https://liki.hk")
	h := webhookHandler(store, svc, "hepan", nil, nil, nil)

	body := `{"type":"payment.succeeded","data":{"order_id":"` + oid + `","payment_id":"pay-1"}}`
	r := httptest.NewRequest("POST", "/api/hepan/webhook", strings.NewReader(body))
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var env struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&env) //nolint:errcheck
	if env.Data.Status != "ok" {
		t.Errorf("status = %q", env.Data.Status)
	}

	// Order should be paid now
	o, _ := store.GetOrder(context.Background(), oid) //nolint:errcheck
	if o.Status != "paid" {
		t.Errorf("order status = %q, want paid", o.Status)
	}
}

func TestHandleWebhook_BadSignature(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	mockProv := &testutil.MockPaymentProvider{VerifyErr: errors.New("bad signature")}
	svc := payment.NewService(mockProv, mockProv, "https://liki.hk")
	h := webhookHandler(store, svc, "hepan", nil, nil, nil)

	r := httptest.NewRequest("POST", "/api/hepan/webhook", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleWebhook_NonPaymentEvent(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	oid := seedOrder(t, store)
	mockProv := &testutil.MockPaymentProvider{
		VerifyEvent: &payment.WebhookEvent{Type: "checkout.created", Data: payment.WebhookEventData{OrderID: oid}},
	}
	svc := payment.NewService(mockProv, mockProv, "https://liki.hk")
	h := webhookHandler(store, svc, "hepan", nil, nil, nil)

	r := httptest.NewRequest("POST", "/api/hepan/webhook", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h(w, r)

	// Non-payment events are accepted (ignored)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// ── HandlePaymentReturn ──

func TestHandlePaymentReturn_Success(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	oid := seedOrder(t, store)
	token := httputil.ReturnToken("hepan", oid)

	cfg := checkout.Config{Product: ProductHepan, ReturnPath: "/api/hepan/payments/return/", RedirectPath: "/hepan/report", LogPrefix: "hepan:"}
	h := checkout.MakePaymentReturn(store, cfg)
	r := httptest.NewRequest("GET", "/api/hepan/payments/return/"+oid+"?status=succeeded&t="+token, nil)
	r.SetPathValue("id", oid)
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "/hepan") {
		t.Errorf("redirect = %q, want /hepan/report", loc)
	}

	// Order should be paid
	o, _ := store.GetOrder(context.Background(), oid) //nolint:errcheck
	if o.Status != "paid" {
		t.Errorf("status = %q, want paid", o.Status)
	}
}

func TestHandlePaymentReturn_BadToken(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	oid := seedOrder(t, store)
	cfg := checkout.Config{Product: ProductHepan, ReturnPath: "/api/hepan/payments/return/", RedirectPath: "/hepan/report", LogPrefix: "hepan:"}
	h := checkout.MakePaymentReturn(store, cfg)

	r := httptest.NewRequest("GET", "/api/hepan/payments/return/"+oid+"?status=succeeded&t=wrongtoken", nil)
	r.SetPathValue("id", oid)
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", w.Code)
	}
	// Order should still be pending
	o, _ := store.GetOrder(context.Background(), oid) //nolint:errcheck
	if o.Status != "pending" {
		t.Errorf("status = %q, want pending", o.Status)
	}
}

func TestHandlePaymentReturn_EmptyOrderID(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	cfg := checkout.Config{Product: ProductHepan, ReturnPath: "/api/hepan/payments/return/", RedirectPath: "/hepan/report", LogPrefix: "hepan:"}
	h := checkout.MakePaymentReturn(store, cfg)

	r := httptest.NewRequest("GET", "/api/hepan/payments/return/?status=succeeded&t=token", nil)
	r.SetPathValue("id", "")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", w.Code)
	}
}
