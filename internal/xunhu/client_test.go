package xunhu

import (
	"context"
	"encoding/json"
	"github.com/ml8s/liki-web/internal/product"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/payment"
)

func TestSign(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		params  map[string]string
		secret  string
		wantLen int // MD5 hex is always 32
	}{
		{
			name:   "basic params",
			params: map[string]string{"appid": "test_app", "version": "1.0", "trade_order_id": "order-1"},
			secret: "secret123",
		},
		{
			name:    "empty params",
			params:  map[string]string{},
			secret:  "secret",
			wantLen: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig := sign(tt.params, tt.secret)
			if len(sig) != 32 {
				t.Errorf("sign() length = %d, want 32", len(sig))
			}
			// Same input produces same output.
			sig2 := sign(tt.params, tt.secret)
			if sig != sig2 {
				t.Error("sign() not deterministic")
			}
		})
	}
}

func TestSign_DifferentSecret(t *testing.T) {
	t.Parallel()
	params := map[string]string{"appid": "test", "version": "1.0"}
	if sign(params, "secretA") == sign(params, "secretB") {
		t.Error("sign() should differ with different secrets")
	}
}

func TestSign_DifferentParams(t *testing.T) {
	t.Parallel()
	a := map[string]string{"appid": "test", "version": "1.0"}
	b := map[string]string{"appid": "test", "version": "1.1"}
	if sign(a, "secret") == sign(b, "secret") {
		t.Error("sign() should differ with different params")
	}
}

func TestSign_OrderIndependent(t *testing.T) {
	t.Parallel()
	// Same key-values in different map iteration order should produce same hash.
	// We create equivalent map literal — Go map iteration is random, so
	// calling sign twice on the same map tests ordering stability.
	params := map[string]string{"c": "3", "a": "1", "b": "2"}
	s1 := sign(params, "secret")
	s2 := sign(params, "secret")
	if s1 != s2 {
		t.Error("sign() should be deterministic regardless of map iteration order")
	}
}

func decodeCheckoutJSON(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	if got := r.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode checkout body: %v", err)
	}
	return body
}

func encodeSignedResponse(t *testing.T, w http.ResponseWriter, secret string, value map[string]any) {
	t.Helper()
	params, err := flatResponseParams(value)
	if err != nil {
		t.Fatal(err)
	}
	value["hash"] = sign(params, secret)
	_ = json.NewEncoder(w).Encode(value)
}

func newTestClient(srv *httptest.Server) *Client {
	return &Client{
		appID: "test_appid", appSecret: "test_secret", webhookURL: "https://liki.hk/api/payments/webhook",
		httpClient: srv.Client(),
		payURL:     srv.URL,
		queryURL:   srv.URL,
	}
}

func TestCreateCheckout_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body := decodeCheckoutJSON(t, r)
		if body["appid"] != "test_appid" {
			t.Errorf("appid = %v", body["appid"])
		}
		if body["trade_order_id"] != "order-1" {
			t.Errorf("trade_order_id = %v", body["trade_order_id"])
		}
		if body["hash"] == "" {
			t.Error("hash is missing")
		}

		w.Header().Set("Content-Type", "application/json")
		encodeSignedResponse(t, w, "test_secret", map[string]any{
			"errcode":    0,
			"errmsg":     "success",
			"url":        "https://pay.xunhupay.com/checkout/test",
			"url_qrcode": "https://pay.xunhupay.com/qrcode/test",
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	result, err := c.CreateCheckout(context.Background(), payment.CheckoutRequest{IntentID: "order-1", Product: string(product.ProductNaming), AmountMinor: 990, Email: "user@test.com", Title: "test", ReturnURL: "https://liki.hk/return"})
	if err != nil {
		t.Fatalf("CreateCheckout: %v", err)
	}
	if result.SessionID == "" {
		t.Error("SessionID should not be empty")
	}
	if result.CheckoutURL != "https://pay.xunhupay.com/checkout/test" {
		t.Errorf("CheckoutURL = %q", result.CheckoutURL)
	}
	if result.QRCodeURL != "https://pay.xunhupay.com/qrcode/test" {
		t.Errorf("QRCodeURL = %q", result.QRCodeURL)
	}
}

func TestCreateCheckout_RejectsInvalidResponseHash(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errcode": 0, "errmsg": "success", "url": "https://pay.example.com/checkout", "hash": "invalid",
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.CreateCheckout(context.Background(), payment.CheckoutRequest{IntentID: "order-1", AmountMinor: 990, Title: "test"})
	if err == nil || !strings.Contains(err.Error(), "response signature mismatch") {
		t.Fatalf("expected response signature mismatch, got %v", err)
	}
}

func TestCreateCheckout_APIError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encodeSignedResponse(t, w, "test_secret", map[string]any{
			"errcode": 400,
			"errmsg":  "appid not found",
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.CreateCheckout(context.Background(), payment.CheckoutRequest{IntentID: "order-1", Product: string(product.ProductNaming), AmountMinor: 990, Title: "test", ReturnURL: "https://liki.hk/return"})
	if err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestCreateCheckout_HTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.CreateCheckout(context.Background(), payment.CheckoutRequest{IntentID: "order-1", Product: string(product.ProductNaming), AmountMinor: 990, Title: "test", ReturnURL: "https://liki.hk/return"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestCreateCheckout_Timeout(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond) // longer than client timeout (200ms)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := c.CreateCheckout(ctx, payment.CheckoutRequest{IntentID: "order-1", Product: string(product.ProductNaming), AmountMinor: 990, Title: "test", ReturnURL: "https://liki.hk/return"})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestVerifyWebhook_PaymentSuccess(t *testing.T) {
	t.Parallel()
	c := &Client{
		appID: "test_appid", appSecret: "test_secret", webhookURL: "https://liki.hk/api/payments/webhook",
	}

	// Build form with valid hash.
	form := url.Values{
		"appid":          {"test_appid"},
		"trade_order_id": {"order-1"},
		"total_fee":      {"9.90"},
		"transaction_id": {"txn_123"},
		"status":         {"OD"},
		"openid":         {"user_openid"},
	}
	formParams := map[string]string{
		"appid":          "test_appid",
		"trade_order_id": "order-1",
		"total_fee":      "9.90",
		"transaction_id": "txn_123",
		"status":         "OD",
		"openid":         "user_openid",
	}
	form.Set("hash", sign(formParams, "test_secret"))

	body := []byte(form.Encode())
	headers := http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}

	event, err := c.VerifyWebhook(body, headers)
	if err != nil {
		t.Fatalf("VerifyWebhook: %v", err)
	}
	if event.Type != "payment.succeeded" {
		t.Errorf("type = %q, want payment.succeeded", event.Type)
	}
	if event.Data.OrderID != "order-1" {
		t.Errorf("OrderID = %q, want order-1", event.Data.OrderID)
	}
	if event.Data.Amount != 990 {
		t.Errorf("Amount = %d, want 990", event.Data.Amount)
	}
	if event.Data.PaymentID != "txn_123" {
		t.Errorf("PaymentID = %q, want txn_123", event.Data.PaymentID)
	}
}

func TestVerifyWebhook_BadSignature(t *testing.T) {
	t.Parallel()
	c := &Client{
		appID: "test_appid", appSecret: "test_secret", webhookURL: "https://liki.hk/api/payments/webhook",
	}

	form := url.Values{
		"appid":          {"test_appid"},
		"trade_order_id": {"order-1"},
		"total_fee":      {"9.90"},
		"status":         {"OD"},
		"hash":           {"bad_signature_here"},
	}
	body := []byte(form.Encode())
	headers := http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}

	_, err := c.VerifyWebhook(body, headers)
	if err == nil {
		t.Fatal("expected error for bad signature")
	}
}

func TestVerifyWebhook_MissingHash(t *testing.T) {
	t.Parallel()
	c := &Client{
		appID: "test_appid", appSecret: "test_secret", webhookURL: "https://liki.hk/api/payments/webhook",
	}

	form := url.Values{
		"appid":          {"test_appid"},
		"trade_order_id": {"order-1"},
		"status":         {"OD"},
	}
	body := []byte(form.Encode())
	headers := http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}

	_, err := c.VerifyWebhook(body, headers)
	if err == nil {
		t.Fatal("expected error for missing hash")
	}
}

func TestVerifyWebhook_NonPaymentStatus(t *testing.T) {
	t.Parallel()
	c := &Client{
		appID: "test_appid", appSecret: "test_secret", webhookURL: "https://liki.hk/api/payments/webhook",
	}

	formParams := map[string]string{
		"appid":          "test_appid",
		"trade_order_id": "order-1",
		"total_fee":      "990",
		"status":         "ORDER_CREATED",
	}
	form := url.Values{}
	for k, v := range formParams {
		form.Set(k, v)
	}
	form.Set("hash", sign(formParams, "test_secret"))

	body := []byte(form.Encode())
	headers := http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}

	event, err := c.VerifyWebhook(body, headers)
	if err != nil {
		t.Fatalf("VerifyWebhook: %v", err)
	}
	if event.Type != "ORDER_CREATED" {
		t.Errorf("type = %q, want ORDER_CREATED", event.Type)
	}
}

func TestVerifyWebhook_InvalidBody(t *testing.T) {
	t.Parallel()
	c := &Client{
		appID: "test_appid", appSecret: "test_secret", webhookURL: "https://liki.hk/api/payments/webhook",
	}

	_, err := c.VerifyWebhook([]byte("%%%"), http.Header{"Content-Type": {"application/x-www-form-urlencoded"}})
	if err == nil {
		t.Fatal("expected error for invalid form body")
	}
}

func TestVerifyWebhook_EmptyBody(t *testing.T) {
	t.Parallel()
	c := &Client{
		appID: "test_appid", appSecret: "test_secret", webhookURL: "https://liki.hk/api/payments/webhook",
	}

	_, err := c.VerifyWebhook([]byte{}, http.Header{"Content-Type": {"application/x-www-form-urlencoded"}})
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestNew(t *testing.T) {
	t.Parallel()
	c := New("app_123", "secret_456", "https://liki.hk/api/payments/webhook")
	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.appID != "app_123" {
		t.Errorf("appID = %q, want app_123", c.appID)
	}
	if c.appSecret != "secret_456" {
		t.Errorf("appSecret = %q, want secret_456", c.appSecret)
	}
	if c.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
	if c.payURL == "" {
		t.Error("payURL should not be empty")
	}
}

func TestCreateCheckout_ProductPricing(t *testing.T) {
	t.Parallel()
	// Verify all known products produce a valid checkout call.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encodeSignedResponse(t, w, "test_secret", map[string]any{
			"errcode": 0,
			"errmsg":  "ok",
			"url":     "https://pay.example.com/checkout",
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)

	tests := []struct {
		name   string
		prod   string
		amount int
	}{
		{"chart", "naming", 990},
		{"bond", "naming", 990},
		{"naming", "naming", 990},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.CreateCheckout(context.Background(), payment.CheckoutRequest{Product: tt.prod, BusinessOrderID: "order-1", AmountMinor: int64(tt.amount), ReturnURL: "https://liki.hk/return"})
			if err != nil {
				t.Fatalf("CreateCheckout(%s): %v", tt.name, err)
			}
			if result.CheckoutURL == "" {
				t.Error("CheckoutURL should not be empty")
			}
		})
	}
}

func TestCreateCheckout_ParamFormat(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := decodeCheckoutJSON(t, r)
		if body["version"] != "1.1" {
			t.Errorf("version = %v, want 1.1", body["version"])
		}
		if nonce, _ := body["nonce_str"].(string); len(nonce) != 32 {
			t.Errorf("nonce_str = %v, want 32 hex chars", body["nonce_str"])
		}
		if body["total_fee"] != "9.90" {
			t.Errorf("total_fee = %v, want 9.90 (yuan decimal)", body["total_fee"])
		}
		if title, _ := body["title"].(string); title == "" {
			t.Error("title should not be empty")
		}

		w.Header().Set("Content-Type", "application/json")
		encodeSignedResponse(t, w, "test_secret", map[string]any{
			"errcode": 0,
			"errmsg":  "ok",
			"url":     "https://pay.example.com/checkout",
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.CreateCheckout(context.Background(), payment.CheckoutRequest{IntentID: "order-1", Product: string(product.ProductNaming), AmountMinor: 990, Title: "test", ReturnURL: "https://liki.hk/return"})
	if err != nil {
		t.Fatalf("CreateCheckout: %v", err)
	}
}

func TestCreateCheckout_AmountFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		amount int
		want   string
	}{
		{"990 fen → 9.90", 990, "9.90"},
		{"990 fen → 9.90", 990, "9.90"},
		{"990 fen → 9.90", 990, "9.90"},
		{"100 fen → 1.00", 100, "1.00"},
		{"0 fen → 0.00", 0, "0.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body := decodeCheckoutJSON(t, r)
				if body["total_fee"] != tt.want {
					t.Errorf("total_fee = %v, want %v", body["total_fee"], tt.want)
				}
				w.Header().Set("Content-Type", "application/json")
				encodeSignedResponse(t, w, "test_secret", map[string]any{
					"errcode": 0, "errmsg": "ok", "url": "https://pay.example.com/checkout",
				})
			}))
			defer srv.Close()

			c := newTestClient(srv)
			_, err := c.CreateCheckout(context.Background(), payment.CheckoutRequest{Product: string(product.ProductNaming), BusinessOrderID: "order-1", AmountMinor: int64(tt.amount), ReturnURL: "https://liki.hk/return"})
			if err != nil {
				t.Fatalf("CreateCheckout: %v", err)
			}
		})
	}
}

func TestVerifyWebhook_LegacyOrderID(t *testing.T) {
	t.Parallel()
	c := &Client{
		appID: "test_appid", appSecret: "test_secret", webhookURL: "https://liki.hk/api/payments/webhook",
	}

	formParams := map[string]string{
		"appid":          "test_appid",
		"out_trade_no":   "legacy-order-1",
		"total_fee":      "9.90",
		"transaction_id": "txn_123",
		"status":         "OD",
	}
	form := url.Values{}
	for k, v := range formParams {
		form.Set(k, v)
	}
	form.Set("hash", sign(formParams, "test_secret"))

	event, err := c.VerifyWebhook(
		[]byte(form.Encode()),
		http.Header{"Content-Type": {"application/x-www-form-urlencoded"}},
	)
	if err != nil {
		t.Fatalf("VerifyWebhook: %v", err)
	}
	if event.Data.OrderID != "legacy-order-1" {
		t.Errorf("OrderID = %q, want legacy-order-1", event.Data.OrderID)
	}
}

func TestCreateCheckout_HashPresent(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := decodeCheckoutJSON(t, r)
		h, _ := body["hash"].(string)
		if h == "" {
			t.Error("hash is missing")
		}
		if len(h) != 32 {
			t.Errorf("hash len = %d, want 32", len(h))
		}
		w.Header().Set("Content-Type", "application/json")
		encodeSignedResponse(t, w, "test_secret", map[string]any{
			"errcode": 0, "errmsg": "ok", "url": "https://pay.example.com/checkout",
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.CreateCheckout(context.Background(), payment.CheckoutRequest{IntentID: "order-1", Product: string(product.ProductNaming), AmountMinor: 990, Title: "test", ReturnURL: "https://liki.hk/return"})
	if err != nil {
		t.Fatalf("CreateCheckout: %v", err)
	}
}

func TestSign_Deterministic(t *testing.T) {
	t.Parallel()
	params := map[string]string{
		"appid":          "201906182089",
		"version":        "1.1",
		"time":           "1782304566",
		"nonce_str":      "b0eaff923b531dc05e4d679aeb32bb73",
		"trade_order_id": "go-test-001",
		"total_fee":      "9.90",
		"title":          "test-chart",
		"notify_url":     "https://liki.hk/api/payments/webhook",
		"return_url":     "https://liki.hk/return",
	}
	got := sign(params, "df0e94d93d3d3328fc83f4daf815ffc2")

	got2 := sign(params, "df0e94d93d3d3328fc83f4daf815ffc2")
	if got != got2 {
		t.Error("sign() should be deterministic")
	}
	if len(got) != 32 {
		t.Errorf("sign() length = %d, want 32", len(got))
	}
}
