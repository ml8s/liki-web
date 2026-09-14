package xunhu

import (
	"context"
	"encoding/json"
	"liki-web/internal/product"
	"net/http"
	"net/http/httptest"
	"testing"

	"liki-web/internal/payment"
)

func encodeUnifiedResponse(w http.ResponseWriter, value map[string]any) {
	params, _ := flatResponseParams(value)
	value["hash"] = sign(params, "secret")
	_ = json.NewEncoder(w).Encode(value)
}

func TestCreateCheckoutUsesFixedNotifyURL(t *testing.T) {
	var form map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		form = map[string]string{}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		for key, value := range body {
			form[key], _ = value.(string)
		}
		encodeUnifiedResponse(w, map[string]any{
			"errcode": 0, "errmsg": "ok", "url": "https://pay.example/checkout", "url_qrcode": "https://pay.example/qr",
		})
	}))
	defer srv.Close()

	c := &Client{appID: "app", appSecret: "secret", httpClient: srv.Client(), payURL: srv.URL, webhookURL: "https://liki.hk/api/payments/webhook"}
	req := payment.CheckoutRequest{
		IntentID: "pi-1", BusinessOrderID: "biz-1", Product: string(product.ProductNaming),
		AmountMinor: 990, Currency: "CNY", Email: "user@example.com", Locale: "zh-Hant",
		ReturnURL: "https://liki.hk/api/payments/return/pi-1", CancelURL: "https://liki.hk/zh-Hant/naming",
	}
	result, err := c.CreateCheckout(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if form["notify_url"] != "https://liki.hk/api/payments/webhook" {
		t.Fatalf("notify_url = %q", form["notify_url"])
	}
	if form["return_url"] != req.ReturnURL {
		t.Fatalf("return_url = %q", form["return_url"])
	}
	if form["callback_url"] != req.CancelURL {
		t.Fatalf("callback_url = %q, want %q", form["callback_url"], req.CancelURL)
	}
	if form["total_fee"] != "9.90" {
		t.Fatalf("total_fee = %q", form["total_fee"])
	}
	if result.QRCodeURL != "https://pay.example/qr" {
		t.Fatalf("result = %+v", result)
	}
}

func TestVerifyWebhookParsesDecimalAmount(t *testing.T) {
	c := &Client{appSecret: "secret"}
	form := map[string]string{
		"appid": "app", "trade_order_id": "pi-1", "total_fee": "9.90",
		"transaction_id": "pay-1", "status": "OD", "openid": "ignored",
	}
	hash := sign(form, "secret")
	form["hash"] = hash
	raw := []byte("appid=app&trade_order_id=pi-1&total_fee=9.90&transaction_id=pay-1&status=OD&openid=ignored&hash=" + hash)
	event, err := c.VerifyWebhook(raw, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	if event.Data.Amount != 990 {
		t.Fatalf("amount = %d, want 990", event.Data.Amount)
	}
	if event.Provider != payment.ProviderXunhu {
		t.Fatalf("provider = %q", event.Provider)
	}
}
