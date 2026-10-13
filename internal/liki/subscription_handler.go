package liki

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ml8s/liki-web/internal/payment"
)

// SubscriptionEventHandler adapts provider webhooks to the local subscription
// and entitlement bounded contexts. It is registered with the generic payment
// runtime after signature verification.
type SubscriptionEventHandler struct {
	subscriptions         *SubscriptionStore
	entitlements          *EntitlementStore
	subscriptionProductID string
	paymentObserver       func(context.Context, payment.WebhookEvent) error
	paymentRefundObserver func(context.Context, payment.WebhookEvent, payment.PaymentIntent) error
}

func NewSubscriptionEventHandler(
	subscriptions *SubscriptionStore,
	entitlements *EntitlementStore,
	subscriptionProductID string,
	paymentObserver func(context.Context, payment.WebhookEvent) error,
) *SubscriptionEventHandler {
	return &SubscriptionEventHandler{
		subscriptions:         subscriptions,
		entitlements:          entitlements,
		subscriptionProductID: strings.TrimSpace(subscriptionProductID),
		paymentObserver:       paymentObserver,
	}
}

func (h *SubscriptionEventHandler) SetPaymentRefundObserver(
	observer func(context.Context, payment.WebhookEvent, payment.PaymentIntent) error,
) {
	if observer != nil {
		h.paymentRefundObserver = observer
	}
}

func (h *SubscriptionEventHandler) HandleSubscriptionEvent(ctx context.Context, event payment.WebhookEvent) error {
	switch {
	case payment.IsSubscriptionLifecycle(event.Type):
		if h.subscriptionProductID == "" || event.Data.Product != h.subscriptionProductID {
			return fmt.Errorf("liki: subscription webhook product %q is not the configured subscription product", event.Data.Product)
		}
		return h.subscriptions.ApplyWebhook(ctx, h.entitlements, event)
	case event.Type == payment.EventPaymentSucceeded && event.Data.SubscriptionID != "":
		return h.subscriptions.RecordPayment(ctx, event, h.paymentObserver)
	case event.Type == payment.EventPaymentRefunded && event.Data.RefundedPaymentID != "":
		return h.subscriptions.RecordPaymentRefund(ctx, event, h.paymentRefundObserver)
	default:
		// Providers may emit payment notifications with no local money effect.
		slog.Info("liki: ignored subscription payment event", "type", event.Type)
		return nil
	}
}
