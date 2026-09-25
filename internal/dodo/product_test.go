package dodo

import (
	"context"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/payment"
)

func TestCreateCheckoutFailsClosedWithoutLikiChatProduct(t *testing.T) {
	t.Parallel()
	client := New("sk_test", "whsec_test", true, map[string]string{})
	_, err := client.CreateCheckout(context.Background(), payment.CheckoutRequest{
		IntentID:        "pi-1",
		BusinessOrderID: "order-1",
		Product:         "liki-chat",
		AmountMinor:     100,
		Currency:        "USD",
	})
	if err == nil || !strings.Contains(err.Error(), "no product ID configured for liki-chat") {
		t.Fatalf("CreateCheckout() error = %v, want missing liki-chat product", err)
	}
}
