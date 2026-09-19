package aipay

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type fakeGateway struct {
	needed     string
	neededErr  error
	verifyFunc func(string) (VerifiedPayment, error)
	confirmErr error
	confirms   int
}

func (g *fakeGateway) PaymentNeededHeader(context.Context, Bill) (string, error) {
	return g.needed, g.neededErr
}

func (g *fakeGateway) Verify(_ context.Context, proof string) (VerifiedPayment, error) {
	return g.verifyFunc(proof)
}

func (g *fakeGateway) Confirm(context.Context, string) error {
	g.confirms++
	return g.confirmErr
}

func newTestHandler(t *testing.T) (*Handler, *fakeGateway, *Store) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	store.nowFun = func() time.Time { return time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC) }
	store.idFun = func() string { return "receipt-1" }

	gateway := &fakeGateway{needed: base64.RawURLEncoding.EncodeToString([]byte(`{"protocol":{}}`))}
	handler, err := NewHandler(store, gateway, testConfig())
	if err != nil {
		t.Fatal(err)
	}
	return handler, gateway, store
}

func testConfig() Config {
	return Config{
		AppID:           "app",
		PrivateKey:      "key",
		AlipayPublicKey: "alipay-key",
		Gateway:         "https://example.test/gateway.do",
		SellerID:        "2088",
		SellerName:      "Liki",
		ServiceID:       "service",
		Amount:          "1.00",
		GoodsName:       "Liki 服务支持",
	}
}

func TestNormalizedAmountRemovesLeadingZeros(t *testing.T) {
	config := testConfig()
	config.Amount = "01.10"
	amount, err := config.normalizedAmount()
	if err != nil || amount != "1.10" {
		t.Fatalf("amount = %q, err = %v", amount, err)
	}
}

func postPay(t *testing.T, handler *Handler, proof string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/aipay", nil)
	if proof != "" {
		request.Header.Set("Payment-Proof", proof)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func decodePaymentNeeded(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if response.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402; body=%q", response.Code, response.Body.String())
	}
	encoded := response.Header().Get("Payment-Needed")
	if encoded == "" {
		t.Fatal("Payment-Needed header is empty")
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode bill: %v", err)
	}
	var bill map[string]any
	if err := json.Unmarshal(raw, &bill); err != nil {
		t.Fatalf("unmarshal bill: %v", err)
	}
	return bill
}

func TestPayDisabledReturnsServiceUnavailable(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	store := NewStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(store, &fakeGateway{}, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if got := postPay(t, handler, ""); got.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", got.Code)
	}
}

func TestPayWithoutProofReturnsSignedPaymentNeeded(t *testing.T) {
	handler, gateway, _ := newTestHandler(t)
	gateway.needed = base64.RawURLEncoding.EncodeToString([]byte(
		`{"protocol":{"resource_id":"` + ResourceID + `","currency":"CNY"}}`))

	response := postPay(t, handler, "")
	bill := decodePaymentNeeded(t, response)
	protocol, ok := bill["protocol"].(map[string]any)
	if !ok {
		t.Fatalf("bill protocol missing: %#v", bill)
	}
	if protocol["resource_id"] != ResourceID || protocol["currency"] != Currency {
		t.Fatalf("protocol = %#v", protocol)
	}
	if response.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestPayInvalidOrRejectedProofReturnsPaymentRequired(t *testing.T) {
	handler, gateway, _ := newTestHandler(t)

	tests := []struct {
		name      string
		proof     string
		verify    *VerifiedPayment
		verifyErr error
	}{
		{name: "malformed", proof: "not-base64url-json"},
		{name: "gateway failure", proof: proofJSON(t, "outer-proof", "trade-1"), verifyErr: errors.New("gateway down")},
		{name: "inactive", proof: proofJSON(t, "outer-proof", "trade-1"), verify: &VerifiedPayment{Active: false}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gateway.verifyFunc = func(string) (VerifiedPayment, error) {
				if test.verifyErr != nil {
					return VerifiedPayment{}, test.verifyErr
				}
				return *test.verify, nil
			}
			if got := postPay(t, handler, test.proof); got.Code != http.StatusPaymentRequired {
				t.Fatalf("status = %d, want 402; body=%q", got.Code, got.Body.String())
			}
		})
	}
}

func TestPayRejectsOversizedProof(t *testing.T) {
	handler, gateway, _ := newTestHandler(t)
	gateway.verifyFunc = func(string) (VerifiedPayment, error) {
		t.Error("verify should not be called for oversized proof")
		return VerifiedPayment{}, nil
	}
	oversized := strings.Repeat("A", 64*1024+1)
	if got := postPay(t, handler, oversized); got.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402", got.Code)
	}
}

func TestPaySuccessfulProofReturnsReceiptAndConfirms(t *testing.T) {
	handler, gateway, _ := newTestHandler(t)
	gateway.verifyFunc = func(proof string) (VerifiedPayment, error) {
		if proof != proofJSON(t, "outer-proof", "trade-1") {
			t.Fatalf("proof passed to gateway = %q", proof)
		}
		return VerifiedPayment{
			TradeNo:    "trade-1",
			OutTradeNo: "order-1",
			Amount:     "1.00",
			ResourceID: ResourceID,
			Active:     true,
		}, nil
	}

	// The fake bill is created in the expected gateway sequence by a first
	// request; its generated out_trade_no is then used by the verified proof.
	first := postPay(t, handler, "")
	if first.Code != http.StatusPaymentRequired {
		t.Fatalf("bill status = %d", first.Code)
	}
	order := latestOrder(t, handler.store)
	gateway.verifyFunc = func(string) (VerifiedPayment, error) {
		return VerifiedPayment{
			TradeNo:    "trade-1",
			OutTradeNo: order.OutTradeNo,
			Amount:     "1.00",
			ResourceID: ResourceID,
			Active:     true,
		}, nil
	}

	response := postPay(t, handler, proofJSON(t, "outer-proof", "trade-1"))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%q", response.Code, response.Body.String())
	}
	var body struct {
		ResourceID string `json:"resource_id"`
		Content    struct {
			SchemaVersion string `json:"schema_version"`
			Kind          string `json:"kind"`
			Status        string `json:"status"`
			ReceiptID     string `json:"receipt_id"`
			Amount        string `json:"amount"`
		} `json:"content"`
		CredentialPath       string `json:"credential_path"`
		FulfillmentConfirmed bool   `json:"fulfillment_confirmed"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.ResourceID != ResourceID || body.Content.Kind != "aipay-receipt" {
		t.Fatalf("body = %#v", body)
	}
	if body.Content.SchemaVersion != "liki-aipay-v1" || body.Content.Status != "received" {
		t.Fatalf("receipt identity = %#v", body.Content)
	}
	if body.Content.Amount != "1.00" || body.CredentialPath != "~/.liki/aipay.json" {
		t.Fatalf("receipt = %#v", body)
	}
	if !body.FulfillmentConfirmed || gateway.confirms != 1 {
		t.Fatalf("confirmed=%v gateway.confirm calls=%d", body.FulfillmentConfirmed, gateway.confirms)
	}

	pv := response.Header().Get("Payment-Validation")
	if pv == "" {
		t.Fatal("Payment-Validation header is empty")
	}
	raw, err := base64.RawURLEncoding.DecodeString(pv)
	if err != nil {
		t.Fatalf("decode Payment-Validation: %v", err)
	}
	var validation struct {
		TradeNo    string `json:"trade_no"`
		OutTradeNo string `json:"out_trade_no"`
		Validated  bool   `json:"validated"`
		ResourceID string `json:"resource_id"`
	}
	if err := json.Unmarshal(raw, &validation); err != nil {
		t.Fatalf("unmarshal Payment-Validation: %v", err)
	}
	if !validation.Validated || validation.TradeNo != "trade-1" ||
		validation.ResourceID != ResourceID {
		t.Fatalf("validation = %#v", validation)
	}
}

func TestPaySameProofDoesNotConfirmTwice(t *testing.T) {
	handler, gateway, _ := newTestHandler(t)
	first := postPay(t, handler, "")
	if first.Code != http.StatusPaymentRequired {
		t.Fatalf("bill status = %d", first.Code)
	}
	order := latestOrder(t, handler.store)
	proof := proofJSON(t, "outer-proof", "trade-1")
	gateway.verifyFunc = func(string) (VerifiedPayment, error) {
		return VerifiedPayment{
			TradeNo: "trade-1", OutTradeNo: order.OutTradeNo,
			Amount: "1.00", ResourceID: ResourceID, Active: true,
		}, nil
	}

	firstRedeem := postPay(t, handler, proof)
	secondRedeem := postPay(t, handler, proof)
	if firstRedeem.Code != http.StatusOK || secondRedeem.Code != http.StatusOK {
		t.Fatalf("statuses = %d/%d", firstRedeem.Code, secondRedeem.Code)
	}
	if gateway.confirms != 1 {
		t.Fatalf("confirm calls = %d, want 1", gateway.confirms)
	}
}

func TestPayConfirmationFailureCanBeRetriedWithSameProof(t *testing.T) {
	handler, gateway, _ := newTestHandler(t)
	if response := postPay(t, handler, ""); response.Code != http.StatusPaymentRequired {
		t.Fatalf("bill status = %d", response.Code)
	}
	order := latestOrder(t, handler.store)
	proof := proofJSON(t, "outer-proof", "trade-1")
	gateway.verifyFunc = func(string) (VerifiedPayment, error) {
		return VerifiedPayment{
			TradeNo: "trade-1", OutTradeNo: order.OutTradeNo,
			Amount: "1.00", ResourceID: ResourceID, Active: true,
		}, nil
	}
	gateway.confirmErr = errors.New("alipay unavailable")

	failed := postPay(t, handler, proof)
	if failed.Code != http.StatusBadGateway {
		t.Fatalf("failed status = %d; body=%q", failed.Code, failed.Body.String())
	}
	gateway.confirmErr = nil
	retried := postPay(t, handler, proof)
	if retried.Code != http.StatusOK {
		t.Fatalf("retried status = %d; body=%q", retried.Code, retried.Body.String())
	}
	if gateway.confirms != 2 {
		t.Fatalf("confirm calls = %d, want 2", gateway.confirms)
	}
}

func proofJSON(t *testing.T, paymentProof, tradeNo string) string {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"protocol": map[string]any{
			"payment_proof": paymentProof,
			"trade_no":      tradeNo,
		},
		"method": map[string]any{
			"client_session": "session",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func latestOrder(t *testing.T, store *Store) Order {
	t.Helper()
	order, err := scanOrder(func(dest ...any) error {
		return store.db.QueryRowContext(context.Background(), `
SELECT out_trade_no, trade_no, amount, currency, resource_id, goods_name,
       pay_before, status, fulfillment_status, receipt_id, paid_at
FROM aipay_orders ORDER BY created_at DESC, rowid DESC LIMIT 1`).Scan(dest...)
	})
	if err != nil {
		t.Fatal(err)
	}
	return order
}
