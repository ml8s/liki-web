// Package payment handles the payment lifecycle: orders, checkout, webhooks, and reports.
package payment

import (
	"context"
	"errors"
	"time"
)

// ---------------------------------------------------------------------------
// Checkout
// ---------------------------------------------------------------------------

// CheckoutResult holds the result of creating a checkout session.
type CheckoutResult struct {
	SessionID   string `json:"session_id"`
	CheckoutURL string `json:"checkout_url"`
	QRCodeURL   string `json:"qr_url,omitempty"`
	// StatusToken authorizes short-interval status polling for this intent only.
	StatusToken string `json:"status_token"`
}

// ---------------------------------------------------------------------------
// Webhook
// ---------------------------------------------------------------------------

// Webhook event types from payment providers.
const (
	EventPaymentSucceeded = "payment.succeeded"
	EventPaymentRefunded  = "payment.refunded"

	EventSubscriptionActive                = "subscription.active"
	EventSubscriptionRenewed               = "subscription.renewed"
	EventSubscriptionUpdated               = "subscription.updated"
	EventSubscriptionCancellationScheduled = "subscription.cancellation_scheduled"
	EventSubscriptionOnHold                = "subscription.on_hold"
	EventSubscriptionFailed                = "subscription.failed"
	EventSubscriptionCancelled             = "subscription.cancelled"
	EventSubscriptionExpired               = "subscription.expired"
)

// Provider identifies a payment provider.
type Provider string

// Provider names.
const (
	ProviderDodo  Provider = "dodo"
	ProviderXunhu Provider = "xunhu"
)

// WebhookEvent is a verified webhook event from a payment provider.
type WebhookEvent struct {
	Provider Provider
	Type     string
	Data     WebhookEventData
}

// WebhookEventData holds the extracted fields from a webhook event.
type WebhookEventData struct {
	OrderID             string
	Amount              int64
	Currency            string
	Email               string
	PaymentID           string
	Metadata            map[string]string
	SubscriptionID      string
	CustomerID          string
	Product             string
	Status              string
	PeriodStart         *time.Time
	PeriodEnd           *time.Time
	NextBillingAt       *time.Time
	CancelAtPeriodEnd   bool
	ProviderPeriodCount int64
	ProviderEventTime   *time.Time
}

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	ErrUnknownProvider = errors.New("payment: unknown provider")
	ErrUnknownProduct  = errors.New("payment: unknown product")
)

// IsSubscriptionLifecycle reports whether a webhook changes subscription state.
func IsSubscriptionLifecycle(eventType string) bool {
	switch eventType {
	case EventSubscriptionActive, EventSubscriptionRenewed, EventSubscriptionUpdated,
		EventSubscriptionCancellationScheduled, EventSubscriptionOnHold,
		EventSubscriptionFailed, EventSubscriptionCancelled, EventSubscriptionExpired:
		return true
	default:
		return false
	}
}

// SubscriptionCheckoutRequest is the provider-neutral request for a hosted
// recurring checkout. Providers that own recurring prices must not accept an
// application-supplied amount.
type SubscriptionCheckoutRequest struct {
	Product         string
	SubscriptionKey string
	UserID          string
	Email           string
	Currency        string
	ReturnURL       string
	CancelURL       string
}

// SubscriptionCancellationGateway updates provider-side renewal policy.
type SubscriptionCancellationGateway interface {
	SetSubscriptionCancellation(ctx context.Context, subscriptionID string, cancel bool) error
}
