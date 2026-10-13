package dodo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dodopayments "github.com/dodopayments/dodopayments-go"
	"github.com/dodopayments/dodopayments-go/option"

	"github.com/ml8s/liki-web/internal/payment"
)

func TestCreateSubscriptionCheckoutOmitsAmount(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"session_id": "checkout_1", "checkout_url": "https://checkout.example/subscription",
		})
	}))
	t.Cleanup(server.Close)
	client := &Client{
		checkoutSvc: dodopayments.NewCheckoutSessionService(option.WithBearerToken("test"), option.WithBaseURL(server.URL)),
		products: map[string]string{
			"liki-chat:one_time":     "pdt_one_time",
			"liki-chat:subscription": "pdt_subscription",
		},
	}

	result, err := client.CreateSubscriptionCheckout(context.Background(), payment.SubscriptionCheckoutRequest{
		Product:         "liki-chat",
		SubscriptionKey: "local-subscription",
		UserID:          "user-1",
		Email:           "user@example.com",
		Currency:        "USD",
		ReturnURL:       "https://liki.hk/app/chat",
		CancelURL:       "https://liki.hk/app/chat",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.CheckoutURL != "https://checkout.example/subscription" {
		t.Fatalf("result = %+v", result)
	}
	cart := body["product_cart"].([]any)[0].(map[string]any)
	if cart["product_id"] != "pdt_subscription" {
		t.Fatalf("product cart = %#v", cart)
	}
	if _, exists := cart["amount"]; exists {
		t.Fatalf("subscription cart contains amount: %#v", cart)
	}
	metadata := body["metadata"].(map[string]any)
	if metadata["liki_subscription_key"] != "local-subscription" || metadata["liki_user_id"] != "user-1" {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestOneTimeCheckoutUsesModeQualifiedProductKey(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"session_id": "checkout_one_time", "checkout_url": "https://checkout.example/pay",
		})
	}))
	t.Cleanup(server.Close)
	client := &Client{
		checkoutSvc: dodopayments.NewCheckoutSessionService(option.WithBearerToken("test"), option.WithBaseURL(server.URL)),
		products: map[string]string{
			"liki-chat:one_time":     "pdt_one_time",
			"liki-chat:subscription": "pdt_subscription",
		},
	}
	_, err := client.CreateCheckout(context.Background(), payment.CheckoutRequest{
		Product: "liki-chat", AmountMinor: 2990, Currency: "USD",
	})
	if err != nil {
		t.Fatal(err)
	}
	cart := body["product_cart"].([]any)[0].(map[string]any)
	if cart["product_id"] != "pdt_one_time" || cart["amount"] != float64(2990) {
		t.Fatalf("cart = %#v", cart)
	}
}

func TestVerifySubscriptionWebhookNormalizesLifecycle(t *testing.T) {
	timestamp := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	payload := map[string]any{
		"type":      "subscription.renewed",
		"timestamp": timestamp.Format(time.RFC3339),
		"data": map[string]any{
			"subscription_id":             "sub_1",
			"product_id":                  "pdt_subscription",
			"status":                      "active",
			"currency":                    "USD",
			"customer":                    map[string]any{"customer_id": "cus_1", "email": "user@example.com"},
			"metadata":                    map[string]string{"liki_subscription_key": "local-subscription", "liki_user_id": "user-1"},
			"previous_billing_date":       timestamp.AddDate(0, -1, 0).Format(time.RFC3339),
			"next_billing_date":           timestamp.AddDate(0, 1, 0).Format(time.RFC3339),
			"cancel_at_next_billing_date": false,
			"subscription_period_count":   2,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	// Verification is delegated to the SDK's Unwrap method in production; this
	// test pins the normalization contract through that same public API.
	var envelope dodopayments.UnwrapWebhookEvent
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	union := envelope.AsUnion()
	event, ok := union.(dodopayments.SubscriptionRenewedWebhookEvent)
	if !ok {
		t.Fatalf("event type = %T, want SubscriptionRenewedWebhookEvent", union)
	}
	normalized := normalizeSubscription(string(event.Type), event.Data, timestamp)
	if normalized.Type != payment.EventSubscriptionRenewed ||
		normalized.Data.SubscriptionID != "sub_1" ||
		normalized.Data.ProviderPeriodCount != 2 ||
		normalized.Data.PeriodEnd == nil ||
		normalized.Data.ProviderEventTime == nil ||
		!normalized.Data.ProviderEventTime.Equal(timestamp) {
		t.Fatalf("normalized = %#v", normalized)
	}
}

func TestVerifyRefundWebhookPreservesOriginalPayment(t *testing.T) {
	timestamp := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	payload := map[string]any{
		"type":      "refund.succeeded",
		"timestamp": timestamp.Format(time.RFC3339),
		"data": map[string]any{
			"refund_id":  "refund_1",
			"payment_id": "pay_1",
			"amount":     1990,
			"currency":   "USD",
			"customer":   map[string]any{"customer_id": "cus_1", "email": "user@example.com"},
			"metadata":   map[string]string{"liki_user_id": "user-1"},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var envelope dodopayments.UnwrapWebhookEvent
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	union := envelope.AsUnion()
	event, ok := union.(dodopayments.RefundSucceededWebhookEvent)
	if !ok {
		t.Fatalf("event type = %T, want RefundSucceededWebhookEvent", union)
	}
	normalized := normalizeRefund(event.Type, event.Data, timestamp)
	if normalized.Type != payment.EventPaymentRefunded ||
		normalized.Data.PaymentID != "refund_1" ||
		normalized.Data.RefundID != "refund_1" ||
		normalized.Data.RefundedPaymentID != "pay_1" ||
		normalized.Data.Amount != 1990 ||
		normalized.Data.ProviderEventTime == nil ||
		!normalized.Data.ProviderEventTime.Equal(timestamp) {
		t.Fatalf("normalized = %#v", normalized)
	}
}

func TestFetchSubscriptionByCheckoutReconcilesLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/checkouts/checkout_1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "checkout_1", "payment_id": "pay_1", "payment_status": "succeeded",
			})
		case "/payments/pay_1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "pay_1", "status": "succeeded", "subscription_id": "sub_1",
			})
		case "/subscriptions/sub_1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"subscription_id": "sub_1",
				"product_id":      "pdt_subscription",
				"status":          "active",
				"currency":        "USD",
				"customer":        map[string]any{"customer_id": "cus_1", "email": "user@example.com"},
				"metadata": map[string]any{
					"liki_subscription_key": "local-subscription",
					"liki_user_id":          "user-1",
					"liki_product":          "liki-chat",
				},
				"previous_billing_date":     "2026-10-01T00:00:00Z",
				"next_billing_date":         "2026-10-31T00:00:00Z",
				"subscription_period_count": 1,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	client := &Client{
		checkoutSvc:     dodopayments.NewCheckoutSessionService(option.WithBearerToken("test"), option.WithBaseURL(server.URL)),
		subscriptionSvc: dodopayments.NewSubscriptionService(option.WithBearerToken("test"), option.WithBaseURL(server.URL)),
		paymentsSvc:     dodopayments.NewPaymentService(option.WithBearerToken("test"), option.WithBaseURL(server.URL)),
	}
	event, err := client.FetchSubscriptionByCheckout(context.Background(), "checkout_1")
	if err != nil {
		t.Fatal(err)
	}
	if event == nil || event.Type != payment.EventSubscriptionActive ||
		event.Data.SubscriptionID != "sub_1" ||
		event.Data.ProviderProductID != "pdt_subscription" ||
		event.Data.Metadata["liki_subscription_key"] != "local-subscription" {
		t.Fatalf("event = %#v", event)
	}
}
