package xunhu

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/ml8s/liki-web/internal/payment"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func jsonResponse(t *testing.T, value any) *http.Response {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(raw)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func queryResponse(data map[string]any) map[string]any {
	return map[string]any{"errcode": 0, "errmsg": "success!", "data": data}
}

func TestFetchPaymentQueriesOfficialContract(t *testing.T) {
	var form url.Values
	var requestURL string
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestURL = request.URL.String()
		if err := request.ParseForm(); err != nil {
			return nil, err
		}
		form = request.Form
		return jsonResponse(t, queryResponse(map[string]any{
			"out_trade_order": "pi-1", "total_amount": "9.90",
			"status": "OD", "transaction_id": "pay-1",
		})), nil
	})}

	c := &Client{appID: "app", appSecret: "secret", queryURL: "https://api.example/payment/query.html", httpClient: client}
	event, err := c.FetchPayment(context.Background(), payment.PaymentIntent{
		ID: "pi-1", Product: "liki-chat", AmountMinor: 990, Currency: "CNY",
	})
	if err != nil {
		t.Fatal(err)
	}
	if requestURL != "https://api.example/payment/query.html" {
		t.Fatalf("query URL = %q", requestURL)
	}
	if form.Get("out_trade_order") != "pi-1" || form.Get("appid") != "app" {
		t.Fatalf("query form = %#v", form)
	}
	if form.Get("hash") == "" {
		t.Fatal("query request is not signed")
	}
	if event == nil || event.Type != payment.EventPaymentSucceeded || event.Data.Amount != 990 ||
		event.Data.PaymentID != "pay-1" || event.Data.OrderID != "pi-1" {
		t.Fatalf("event = %+v", event)
	}
	if event.Data.Metadata["amount_verified"] != "true" {
		t.Fatalf("event = %+v", event)
	}
}

func TestFetchPaymentAcceptsOfficialPendingState(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return jsonResponse(t, queryResponse(map[string]any{
			"out_trade_order": "pi-1", "total_amount": "9.90",
			"status": "WP",
		})), nil
	})}

	c := &Client{appID: "app", appSecret: "secret", queryURL: "https://api.example/payment/query.html", httpClient: client}
	event, err := c.FetchPayment(context.Background(), payment.PaymentIntent{
		ID: "pi-1", AmountMinor: 990, Currency: "CNY",
	})
	if err != nil {
		t.Fatal(err)
	}
	if event != nil {
		t.Fatalf("pending should not produce a payment event, got %+v", event)
	}
}

func TestFetchPaymentRejectsPaidResponseWithoutVerifiedAmount(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return jsonResponse(t, queryResponse(map[string]any{
			"out_trade_order": "pi-1", "status": "OD", "transaction_id": "pay-1",
		})), nil
	})}

	c := &Client{appID: "app", appSecret: "secret", queryURL: "https://api.example/payment/query.html", httpClient: client}
	event, err := c.FetchPayment(context.Background(), payment.PaymentIntent{
		ID: "pi-1", AmountMinor: 990, Currency: "CNY",
	})
	if err == nil || event != nil {
		t.Fatalf("expected error without amount, got event=%+v err=%v", event, err)
	}
}

func TestFetchPaymentRejectsAmountMismatch(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return jsonResponse(t, queryResponse(map[string]any{
			"out_trade_order": "pi-1", "total_amount": "0.01",
			"status": "OD", "transaction_id": "pay-1",
		})), nil
	})}

	c := &Client{appID: "app", appSecret: "secret", queryURL: "https://api.example/payment/query.html", httpClient: client}
	event, err := c.FetchPayment(context.Background(), payment.PaymentIntent{
		ID: "pi-1", AmountMinor: 990, Currency: "CNY",
	})
	if err == nil || event != nil {
		t.Fatalf("expected amount mismatch, got event=%+v err=%v", event, err)
	}
}

func TestXunhuWebhookAckUsesOfficialSuccessBody(t *testing.T) {
	c := &Client{}
	if got := string(c.WebhookAck()); got != "success" {
		t.Fatalf("WebhookAck = %q, want success", got)
	}
}
