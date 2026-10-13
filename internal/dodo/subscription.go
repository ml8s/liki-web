package dodo

import (
	"time"

	dodopayments "github.com/dodopayments/dodopayments-go"
	"github.com/ml8s/liki-web/internal/payment"
)

// normalizeSubscription extracts the fields required to maintain local access.
// The provider object—not webhook event name alone—is authoritative.
func normalizeSubscription(eventType string, data dodopayments.Subscription) *payment.WebhookEvent {
	previous := data.PreviousBillingDate
	next := data.NextBillingDate
	eventTime := data.CreatedAt
	return &payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     eventType,
		Data: payment.WebhookEventData{
			SubscriptionID:      data.SubscriptionID,
			CustomerID:          data.Customer.CustomerID,
			Email:               data.Customer.Email,
			Currency:            string(data.Currency),
			Product:             data.ProductID,
			Status:              string(data.Status),
			PeriodStart:         timePtr(previous),
			PeriodEnd:           timePtr(next),
			NextBillingAt:       timePtr(next),
			CancelAtPeriodEnd:   data.CancelAtNextBillingDate,
			ProviderPeriodCount: data.SubscriptionPeriodCount,
			ProviderEventTime:   timePtr(eventTime),
			Metadata:            data.Metadata,
		},
	}
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	copied := value
	return &copied
}
