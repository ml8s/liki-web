package donation

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
		GoodsName:       "Liki 捐赠",
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

func postRedeem(t *testing.T, handler *Handler, proof string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/donations/redeem", nil)
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

func TestRedeemDisabledReturnsServiceUnavailable(t *testing.T) {
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
	if got := postRedeem(t, handler, ""); got.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", got.Code)
	}
}

func TestRedeemWithoutProofReturnsSignedPaymentNeeded(t *testing.T) {
	handler, gateway, _ := newTestHandler(t)
	gateway.needed = base64.RawURLEncoding.EncodeToString([]byte(
		`{"protocol":{"resource_id":"` + ResourceID + `","currency":"CNY"}}`))

	response := postRedeem(t, handler, "")
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

func TestRedeemInvalidOrRejectedProofReturnsPaymentRequired(t *testing.T) {
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
			if got := postRedeem(t, handler, test.proof); got.Code != http.StatusPaymentRequired {
				t.Fatalf("status = %d, want 402; body=%q", got.Code, got.Body.String())
			}
		})
	}
}

func TestRedeemSuccessfulProofReturnsReceiptAndConfirms(t *testing.T) {
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
	first := postRedeem(t, handler, "")
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

	response := postRedeem(t, handler, proofJSON(t, "outer-proof", "trade-1"))
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
	if body.ResourceID != ResourceID || body.Content.Kind != "donation-receipt" {
		t.Fatalf("body = %#v", body)
	}
	if body.Content.SchemaVersion != "liki-donation-v1" || body.Content.Status != "received" {
		t.Fatalf("receipt identity = %#v", body.Content)
	}
	if body.Content.Amount != "1.00" || body.CredentialPath != "~/.liki/donation.json" {
		t.Fatalf("receipt = %#v", body)
	}
	if !body.FulfillmentConfirmed || gateway.confirms != 1 {
		t.Fatalf("confirmed=%v gateway.confirm calls=%d", body.FulfillmentConfirmed, gateway.confirms)
	}
}

func TestRedeemSameProofDoesNotConfirmTwice(t *testing.T) {
	handler, gateway, _ := newTestHandler(t)
	first := postRedeem(t, handler, "")
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

	firstRedeem := postRedeem(t, handler, proof)
	secondRedeem := postRedeem(t, handler, proof)
	if firstRedeem.Code != http.StatusOK || secondRedeem.Code != http.StatusOK {
		t.Fatalf("statuses = %d/%d", firstRedeem.Code, secondRedeem.Code)
	}
	if gateway.confirms != 1 {
		t.Fatalf("confirm calls = %d, want 1", gateway.confirms)
	}
}

func TestRedeemConfirmationFailureCanBeRetriedWithSameProof(t *testing.T) {
	handler, gateway, _ := newTestHandler(t)
	if response := postRedeem(t, handler, ""); response.Code != http.StatusPaymentRequired {
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

	failed := postRedeem(t, handler, proof)
	if failed.Code != http.StatusBadGateway {
		t.Fatalf("failed status = %d; body=%q", failed.Code, failed.Body.String())
	}
	gateway.confirmErr = nil
	retried := postRedeem(t, handler, proof)
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
	order, err := store.LatestOrder(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return order
}
