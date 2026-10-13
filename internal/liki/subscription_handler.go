package liki

import (
	"context"
	"log/slog"

	"github.com/ml8s/liki-web/internal/payment"
)

// SubscriptionEventHandler adapts provider webhooks to the local subscription
// and entitlement bounded contexts. It is registered with the generic payment
// runtime after signature verification.
type SubscriptionEventHandler struct {
	subscriptions   *SubscriptionStore
	entitlements    *EntitlementStore
	paymentObserver func(context.Context, payment.WebhookEvent) error
}

func NewSubscriptionEventHandler(
	subscriptions *SubscriptionStore,
	entitlements *EntitlementStore,
	paymentObserver func(context.Context, payment.WebhookEvent) error,
) *SubscriptionEventHandler {
	return &SubscriptionEventHandler{
		subscriptions:   subscriptions,
		entitlements:    entitlements,
		paymentObserver: paymentObserver,
	}
}

func (h *SubscriptionEventHandler) HandleSubscriptionEvent(ctx context.Context, event payment.WebhookEvent) error {
	switch {
	case payment.IsSubscriptionLifecycle(event.Type):
		return h.subscriptions.ApplyWebhook(ctx, h.entitlements, event)
	case event.Type == payment.EventPaymentSucceeded && event.Data.SubscriptionID != "":
		return h.subscriptions.RecordPayment(ctx, event, h.paymentObserver)
	default:
		// Refund normalization varies by provider payload. Until a provider
		// supplies a subscription payment reference, it is safely ignored here
		// rather than mistakenly voiding a one-time order commission.
		slog.Info("liki: ignored subscription payment event", "type", event.Type)
		return nil
	}
}
