package xunhu

import (
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"testing"
)

// TestSign_ExcludesEmptyValues pins the official Xunhu canonicalization:
// empty values participate neither in ASCII ordering nor in the signature
// material. The expected digest is calculated here from the documented
// canonical string rather than by reusing sign().
func TestSign_ExcludesEmptyValues(t *testing.T) {
	params := map[string]string{
		"a":     "1",
		"b":     "2",
		"empty": "",
	}

	digest := md5.Sum([]byte("a=1&b=2secret"))
	want := hex.EncodeToString(digest[:])
	if got := sign(params, "secret"); got != want {
		t.Fatalf("sign() = %q, want %q", got, want)
	}
}

// Optional callback fields can arrive empty. They must still verify when the
// provider omitted the empty field from its signature material.
func TestVerifyWebhook_EmptyOptionalFieldUsesOfficialSignature(t *testing.T) {
	client := &Client{appSecret: "secret"}
	digest := md5.Sum([]byte("appid=app&email=not-official@example.com&status=OD&total_fee=9.90&trade_order_id=pi-1&transaction_id=pay-1secret"))
	hash := hex.EncodeToString(digest[:])

	raw := []byte("appid=app&attach=&email=not-official@example.com&status=OD&total_fee=9.90&trade_order_id=pi-1&transaction_id=pay-1&hash=" + hash)
	event, err := client.VerifyWebhook(raw, http.Header{})
	if err != nil {
		t.Fatalf("VerifyWebhook() error = %v", err)
	}
	if event == nil || event.Data.OrderID != "pi-1" || event.Data.PaymentID != "pay-1" {
		t.Fatalf("event = %+v", event)
	}
	if event.Data.Email != "" {
		t.Fatalf("email = %q, want empty for non-contract field", event.Data.Email)
	}
	if event.Type != "payment.succeeded" {
		t.Fatalf("event.Type = %q", event.Type)
	}
}
