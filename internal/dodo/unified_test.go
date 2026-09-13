package dodo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	dodopayments "github.com/dodopayments/dodopayments-go"
	"github.com/dodopayments/dodopayments-go/option"

	"liki-web/internal/payment"
	"liki-web/internal/product"
)

func TestCreateCheckoutUnifiedRequest(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"session_id": "sess_unified", "checkout_url": "https://checkout.example/pay",
		})
	}))
	defer srv.Close()

	c := &Client{
		checkoutSvc: dodopayments.NewCheckoutSessionService(option.WithBearerToken("test"), option.WithBaseURL(srv.URL)),
		products: map[product.Product]string{
			product.ProductNaming: "pdt_naming",
		},
	}

	req := payment.CheckoutRequest{
		IntentID:        "pi-1",
		BusinessOrderID: "biz-1",
		Product:         product.ProductNaming,
		AmountMinor:     990,
		Currency:        "USD",
		Email:           "user@example.com",
		Locale:          "zh-Hant",
		ReturnURL:       "https://liki.hk/api/payments/return/pi-1",
		CancelURL:       "https://liki.hk/zh-Hant/naming?payment=cancelled",
		Metadata: map[string]string{
			"payment_intent_id": "pi-1",
		},
	}
	result, err := c.CreateCheckout(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.CheckoutURL != "https://checkout.example/pay" {
		t.Fatalf("result = %+v", result)
	}
	cart := body["product_cart"].([]any)[0].(map[string]any)
	if cart["product_id"] != "pdt_naming" || cart["amount"] != float64(990) {
		t.Fatalf("product cart = %#v", cart["product_cart"])
	}
	metadata := body["metadata"].(map[string]any)
	if metadata["payment_intent_id"] != "pi-1" || metadata["product"] != "naming" {
		t.Fatalf("metadata = %#v", metadata)
	}
	if body["return_url"] != req.ReturnURL {
		t.Fatalf("return_url = %#v", body["return_url"])
	}
	if body["cancel_url"] != req.CancelURL {
		t.Fatalf("cancel_url = %#v, want %q", body["cancel_url"], req.CancelURL)
	}
	if body["billing_currency"] != req.Currency {
		t.Fatalf("billing_currency = %#v, want %q", body["billing_currency"], req.Currency)
	}
}
