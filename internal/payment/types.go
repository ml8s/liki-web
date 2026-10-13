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
	CheckoutSessionID string `json:"session_id"`
	CheckoutURL       string `json:"checkout_url"`
	QRCodeURL         string `json:"qr_url,omitempty"`
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

// Metadata keys form the provider-neutral webhook contract. Values are stable
// wire keys and must not be renamed without a compatibility alias.
const (
	MetadataBusinessOrderID     = "business_order_id"
	MetadataEmail               = "email"
	MetadataLikiBillingMode     = "liki_billing_mode"
	MetadataLikiProduct         = "liki_product"
	MetadataLikiSubscriptionKey = "liki_subscription_key"
	MetadataLikiUserID          = "liki_user_id"
	MetadataPaymentIntentID     = "payment_intent_id"
	MetadataOrderID             = "order_id"
	MetadataProduct             = "product"
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
	RefundID            string
	RefundedPaymentID   string
	Metadata            map[string]string
	SubscriptionID      string
	CustomerID          string
	ProviderProductID   string
	Status              string
	PeriodStart         *time.Time
	PeriodEnd           *time.Time
	NextBillingAt       *time.Time
	CancelAtPeriodEnd   bool
	ProviderPeriodCount int64
	ProviderEventTime   *time.Time
}

// CommissionablePayment is the minimal successful-payment fact consumed by
// revenue sharing. It intentionally omits checkout, redirect, and intent
// lifecycle state that referral must not own.
type CommissionablePayment struct {
	ID              string
	BusinessOrderID string
	UserID          string
	Product         string
	Provider        Provider
	AmountMinor     int64
	Currency        string
	Email           string
}

// RefundReference returns the refund operation ID, with the legacy webhook
// field retained as a compatibility fallback for providers that do not expose
// a distinct refund identifier.
func (d WebhookEventData) RefundReference() string {
	if d.RefundID != "" {
		return d.RefundID
	}
	return d.PaymentID
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
