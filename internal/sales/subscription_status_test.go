package sales

import (
	"testing"

	"github.com/ml8s/liki-web/internal/payment"
)

func TestSubscriptionStatusForEvent(t *testing.T) {
	cases := []struct {
		name  string
		event payment.WebhookEvent
		want  SubscriptionStatus
	}{
		{name: "activation", event: payment.WebhookEvent{Type: payment.EventSubscriptionActive}, want: SubscriptionActive},
		{name: "renewal", event: payment.WebhookEvent{Type: payment.EventSubscriptionRenewed}, want: SubscriptionActive},
		{name: "scheduled cancellation remains active", event: payment.WebhookEvent{Type: payment.EventSubscriptionCancellationScheduled}, want: SubscriptionActive},
		{name: "provider update", event: payment.WebhookEvent{Type: payment.EventSubscriptionUpdated, Data: payment.WebhookEventData{Status: "on_hold"}}, want: SubscriptionOnHold},
		{name: "unknown provider update is held", event: payment.WebhookEvent{Type: payment.EventSubscriptionUpdated, Data: payment.WebhookEventData{Status: "mystery"}}, want: SubscriptionOnHold},
		{name: "cancellation", event: payment.WebhookEvent{Type: payment.EventSubscriptionCancelled}, want: SubscriptionCancelled},
		{name: "expiry", event: payment.WebhookEvent{Type: payment.EventSubscriptionExpired}, want: SubscriptionExpired},
		{name: "unknown event fails closed", event: payment.WebhookEvent{Type: "subscription.mystery"}, want: SubscriptionFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := subscriptionStatusForEvent(tc.event); got != tc.want {
				t.Fatalf("status = %q, want %q", got, tc.want)
			}
		})
	}
}
