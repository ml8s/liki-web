package liki

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ml8s/liki-web/internal/payment"
)

// SubscriptionEventProcessor coordinates provider webhooks across local
// subscription, entitlement, payment, and commission contexts. It is
// registered with the generic payment runtime after signature verification.
type SubscriptionEventProcessor struct {
	subscriptions                 *SubscriptionStore
	entitlements                  *EntitlementStore
	providerSubscriptionProductID string
	commissionObserver            func(context.Context, payment.CommissionablePayment) error
	commissionRefundObserver      func(context.Context, payment.WebhookEvent, payment.CommissionablePayment) error
}

func NewSubscriptionEventProcessor(
	subscriptions *SubscriptionStore,
	entitlements *EntitlementStore,
	providerSubscriptionProductID string,
	commissionObserver func(context.Context, payment.CommissionablePayment) error,
) *SubscriptionEventProcessor {
	return &SubscriptionEventProcessor{
		subscriptions:                 subscriptions,
		entitlements:                  entitlements,
		providerSubscriptionProductID: strings.TrimSpace(providerSubscriptionProductID),
		commissionObserver:            commissionObserver,
	}
}

func (p *SubscriptionEventProcessor) SetCommissionRefundObserver(
	observer func(context.Context, payment.WebhookEvent, payment.CommissionablePayment) error,
) {
	if observer != nil {
		p.commissionRefundObserver = observer
	}
}

func (p *SubscriptionEventProcessor) HandleSubscriptionEvent(ctx context.Context, event payment.WebhookEvent) error {
	switch {
	case payment.IsSubscriptionLifecycle(event.Type):
		if p.providerSubscriptionProductID == "" || event.Data.ProviderProductID != p.providerSubscriptionProductID {
			return fmt.Errorf("liki: subscription webhook product %q is not the configured subscription product", event.Data.ProviderProductID)
		}
		return p.subscriptions.ApplyWebhook(ctx, p.entitlements, event)
	case event.Type == payment.EventPaymentSucceeded && event.Data.SubscriptionID != "":
		return p.subscriptions.RecordPayment(ctx, event, func(ctx context.Context, event payment.WebhookEvent) error {
			if p.commissionObserver == nil {
				return nil
			}
			return p.commissionObserver(ctx, payment.CommissionablePayment{
				ID:              event.Data.PaymentID,
				BusinessOrderID: event.Data.SubscriptionID,
				UserID:          event.Data.Metadata[payment.MetadataLikiUserID],
				Product:         string(ProductChat),
				Provider:        event.Provider,
				AmountMinor:     event.Data.Amount,
				Currency:        event.Data.Currency,
				Email:           event.Data.Email,
			})
		})
	case event.Type == payment.EventPaymentRefunded && event.Data.RefundedPaymentID != "":
		return p.subscriptions.RecordPaymentRefund(ctx, event, p.commissionRefundObserver)
	default:
		// Providers may emit payment notifications with no local money effect.
		slog.Info("liki: ignored subscription payment event", "type", event.Type)
		return nil
	}
}
