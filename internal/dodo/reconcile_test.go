package dodo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dodopayments "github.com/dodopayments/dodopayments-go"
	"github.com/dodopayments/dodopayments-go/option"

	"github.com/ml8s/liki-web/internal/payment"
)

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encode JSON: %v", err)
	}
}

func newReconcileClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	options := []option.RequestOption{
		option.WithBearerToken("test"),
		option.WithBaseURL(server.URL),
	}
	return &Client{
		checkoutSvc: dodopayments.NewCheckoutSessionService(options...),
		paymentsSvc: dodopayments.NewPaymentService(options...),
	}
}

func successfulCheckoutSession(paymentID any) map[string]any {
	return map[string]any{
		"id":             "sess-1",
		"payment_id":     paymentID,
		"payment_status": "succeeded",
		"customer_email": "session@example.com",
		"created_at":     "2026-09-13T00:00:00Z",
	}
}

func successfulPayment(metadata map[string]string) map[string]any {
	return map[string]any{
		"payment_id":          "pay-1",
		"status":              "succeeded",
		"currency":            "USD",
		"total_amount":        990,
		"checkout_session_id": "sess-1",
		"customer": map[string]any{
			"customer_id": "cus-1",
			"email":       "user@example.com",
			"name":        "User",
		},
		"metadata": metadata,
	}
}

func testIntent() payment.PaymentIntent {
	return payment.PaymentIntent{
		ID:                        "pi-1",
		BusinessOrderID:           "biz-1",
		Product:                   "liki-chat",
		AmountMinor:               990,
		Currency:                  "USD",
		ProviderCheckoutSessionID: "sess-1",
	}
}

func TestFetchPaymentUsesOfficialCheckoutSessionReconciliation(t *testing.T) {
	var checkoutCalls, paymentCalls int
	c := newReconcileClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/checkouts/sess-1":
			checkoutCalls++
			if r.Method != http.MethodGet {
				t.Errorf("checkout method = %s, want GET", r.Method)
			}
			writeJSON(t, w, successfulCheckoutSession("pay-1"))
		case "/payments/pay-1":
			paymentCalls++
			if r.Method != http.MethodGet {
				t.Errorf("payment method = %s, want GET", r.Method)
			}
			writeJSON(t, w, successfulPayment(map[string]string{
				"payment_intent_id": "pi-1",
				"business_order_id": "biz-1",
			}))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	})

	event, err := c.FetchPayment(context.Background(), testIntent())
	if err != nil {
		t.Fatal(err)
	}
	if event == nil || event.Type != payment.EventPaymentSucceeded ||
		event.Data.OrderID != "pi-1" || event.Data.PaymentID != "pay-1" ||
		event.Data.Amount != 990 || event.Data.Currency != "USD" ||
		event.Data.Email != "user@example.com" {
		t.Fatalf("event = %+v", event)
	}
	if checkoutCalls != 1 || paymentCalls != 1 {
		t.Fatalf("calls: checkout=%d payment=%d, want one each", checkoutCalls, paymentCalls)
	}
}

func TestFetchPaymentRejectsPaymentBoundToAnotherIntent(t *testing.T) {
	c := newReconcileClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/checkouts/sess-1":
			writeJSON(t, w, successfulCheckoutSession("pay-1"))
		case "/payments/pay-1":
			writeJSON(t, w, successfulPayment(map[string]string{"payment_intent_id": "pi-other"}))
		default:
			http.NotFound(w, r)
		}
	})

	event, err := c.FetchPayment(context.Background(), testIntent())
	if err == nil || event != nil {
		t.Fatalf("event=%v, err=%v, want binding error and no event", event, err)
	}
	if !strings.Contains(err.Error(), "does not match intent") {
		t.Fatalf("error = %v, want intent binding mismatch", err)
	}
}

func TestFetchPaymentRejectsAmountMismatch(t *testing.T) {
	c := newReconcileClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/checkouts/sess-1":
			writeJSON(t, w, successfulCheckoutSession("pay-1"))
		case "/payments/pay-1":
			response := successfulPayment(map[string]string{"payment_intent_id": "pi-1"})
			response["total_amount"] = 1000
			writeJSON(t, w, response)
		default:
			http.NotFound(w, r)
		}
	})

	event, err := c.FetchPayment(context.Background(), testIntent())
	if err == nil || event != nil {
		t.Fatalf("event=%v, err=%v, want amount error and no event", event, err)
	}
	if !strings.Contains(err.Error(), "amount mismatch") {
		t.Fatalf("error = %v, want amount mismatch", err)
	}
}

func TestFetchPaymentRejectsCurrencyMismatch(t *testing.T) {
	c := newReconcileClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/checkouts/sess-1":
			writeJSON(t, w, successfulCheckoutSession("pay-1"))
		case "/payments/pay-1":
			response := successfulPayment(map[string]string{"payment_intent_id": "pi-1"})
			response["currency"] = "TWD"
			writeJSON(t, w, response)
		default:
			http.NotFound(w, r)
		}
	})

	event, err := c.FetchPayment(context.Background(), testIntent())
	if err == nil || event != nil {
		t.Fatalf("event=%v, err=%v, want currency error and no event", event, err)
	}
	if !strings.Contains(err.Error(), "currency mismatch") {
		t.Fatalf("error = %v, want currency mismatch", err)
	}
}

func TestFetchPaymentIgnoresIncompleteOrPendingSession(t *testing.T) {
	statuses := []string{"processing", "requires_customer_action"}
	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			var paymentCalls int
			c := newReconcileClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/checkouts/sess-1" {
					paymentCalls++
					http.NotFound(w, r)
					return
				}
				writeJSON(t, w, map[string]any{
					"id":             "sess-1",
					"payment_id":     nil,
					"payment_status": status,
					"created_at":     "2026-09-13T00:00:00Z",
				})
			})

			event, err := c.FetchPayment(context.Background(), testIntent())
			if err != nil || event != nil {
				t.Fatalf("event=%v, err=%v, want no event", event, err)
			}
			if paymentCalls != 0 {
				t.Fatalf("payment calls = %d, want 0", paymentCalls)
			}
		})
	}
}

func TestFetchPaymentIgnoresSessionWithoutPaymentID(t *testing.T) {
	var paymentCalls int
	c := newReconcileClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/checkouts/sess-1" {
			paymentCalls++
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, successfulCheckoutSession(nil))
	})

	event, err := c.FetchPayment(context.Background(), testIntent())
	if err != nil || event != nil {
		t.Fatalf("event=%v, err=%v, want no event", event, err)
	}
	if paymentCalls != 0 {
		t.Fatalf("payment calls = %d, want 0", paymentCalls)
	}
}

func TestFetchPaymentByKnownProviderPaymentID(t *testing.T) {
	var checkoutCalls int
	c := newReconcileClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/payments/pay-1" {
			checkoutCalls++
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, successfulPayment(map[string]string{"payment_intent_id": "pi-1"}))
	})

	intent := testIntent()
	intent.ProviderPaymentID = "pay-1"
	event, err := c.FetchPayment(context.Background(), intent)
	if err != nil {
		t.Fatal(err)
	}
	if event == nil || event.Data.PaymentID != "pay-1" || event.Data.OrderID != "pi-1" {
		t.Fatalf("event = %+v", event)
	}
	if checkoutCalls != 0 {
		t.Fatalf("checkout calls = %d, want 0", checkoutCalls)
	}
}

func TestFetchPaymentIgnoresKnownPendingProviderPayment(t *testing.T) {
	var checkoutCalls int
	c := newReconcileClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/payments/pay-1" {
			checkoutCalls++
			http.NotFound(w, r)
			return
		}
		response := successfulPayment(map[string]string{"payment_intent_id": "pi-1"})
		response["status"] = "processing"
		writeJSON(t, w, response)
	})

	intent := testIntent()
	intent.ProviderPaymentID = "pay-1"
	event, err := c.FetchPayment(context.Background(), intent)
	if err != nil || event != nil {
		t.Fatalf("event=%v, err=%v, want no event", event, err)
	}
	if checkoutCalls != 0 {
		t.Fatalf("checkout calls = %d, want 0", checkoutCalls)
	}
}

func TestFetchPaymentRequiresSucceededCheckoutPaymentDetail(t *testing.T) {
	var paymentCalls int
	c := newReconcileClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/checkouts/sess-1":
			writeJSON(t, w, successfulCheckoutSession("pay-1"))
		case "/payments/pay-1":
			paymentCalls++
			response := successfulPayment(map[string]string{"payment_intent_id": "pi-1"})
			response["status"] = "processing"
			writeJSON(t, w, response)
		default:
			http.NotFound(w, r)
		}
	})

	event, err := c.FetchPayment(context.Background(), testIntent())
	if err != nil || event != nil {
		t.Fatalf("event=%v, err=%v, want no event", event, err)
	}
	if paymentCalls != 1 {
		t.Fatalf("payment calls = %d, want 1", paymentCalls)
	}
}

func TestFetchPaymentRejectsCheckoutSessionMismatch(t *testing.T) {
	c := newReconcileClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/payments/pay-1" {
			http.NotFound(w, r)
			return
		}
		response := successfulPayment(map[string]string{"payment_intent_id": "pi-1"})
		response["checkout_session_id"] = "sess-other"
		writeJSON(t, w, response)
	})

	intent := testIntent()
	intent.ProviderPaymentID = "pay-1"
	event, err := c.FetchPayment(context.Background(), intent)
	if err == nil || event != nil {
		t.Fatalf("event=%v, err=%v, want checkout session error", event, err)
	}
	if !strings.Contains(err.Error(), "checkout session") || !strings.Contains(err.Error(), "does not match intent") {
		t.Fatalf("error = %v, want checkout session mismatch", err)
	}
}

func TestFetchPaymentRequiresIntentMetadata(t *testing.T) {
	c := newReconcileClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/payments/pay-1" {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, successfulPayment(nil))
	})

	intent := testIntent()
	intent.ProviderPaymentID = "pay-1"
	event, err := c.FetchPayment(context.Background(), intent)
	if err == nil || event != nil {
		t.Fatalf("event=%v, err=%v, want metadata error", event, err)
	}
	if !strings.Contains(err.Error(), "payment_intent_id") {
		t.Fatalf("error = %v, want missing payment_intent_id", err)
	}
}

func TestDodoWebhookAckIsSuccessfulEmptyBody(t *testing.T) {
	c := &Client{}
	if got := c.WebhookAck(); len(got) != 0 {
		t.Fatalf("WebhookAck = %q, want empty", got)
	}
}
