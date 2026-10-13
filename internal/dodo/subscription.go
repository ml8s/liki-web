package dodo

import (
	"time"

	dodopayments "github.com/dodopayments/dodopayments-go"
	"github.com/ml8s/liki-web/internal/payment"
)

// normalizeSubscription extracts the fields required to maintain local access.
// The provider object—not webhook event name alone—is authoritative.
func normalizeSubscription(eventType string, data dodopayments.Subscription, eventTime time.Time) *payment.WebhookEvent {
	previous := data.PreviousBillingDate
	next := data.NextBillingDate
	return &payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     eventType,
		Data: payment.WebhookEventData{
			SubscriptionID:      data.SubscriptionID,
			CustomerID:          data.Customer.CustomerID,
			Email:               data.Customer.Email,
			Currency:            string(data.Currency),
			ProviderProductID:   data.ProductID,
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

// normalizeRefund keeps PaymentID as the provider refund ID for the existing
// one-time refund flow, while retaining the original payment reference needed
// by subscription payments.
func normalizeRefund(eventType dodopayments.RefundSucceededWebhookEventType, data dodopayments.Refund, eventTime time.Time) *payment.WebhookEvent {
	metadata := data.Metadata
	orderID := metadata[payment.MetadataPaymentIntentID]
	if orderID == "" {
		orderID = metadata[payment.MetadataOrderID]
	}
	email := data.Customer.Email
	if email == "" {
		email = metadata[payment.MetadataEmail]
	}
	return &payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     payment.EventPaymentRefunded,
		Data: payment.WebhookEventData{
			OrderID:           orderID,
			Amount:            data.Amount,
			Currency:          string(data.Currency),
			Email:             email,
			PaymentID:         data.RefundID,
			RefundID:          data.RefundID,
			RefundedPaymentID: data.PaymentID,
			Metadata:          metadata,
			ProviderEventTime: timePtr(eventTime),
		},
	}
}
