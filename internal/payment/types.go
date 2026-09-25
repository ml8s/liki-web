// Package payment handles the payment lifecycle: orders, checkout, webhooks, and reports.
package payment

import (
	"database/sql"
	"errors"

	"github.com/ml8s/liki-web/internal/sqliteutil"
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
	OrderID   string
	Amount    int64
	Currency  string
	Email     string
	PaymentID string
	Metadata  map[string]string
}

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	ErrUnknownProvider = errors.New("payment: unknown provider")
	ErrUnknownProduct  = errors.New("payment: unknown product")
)

// OpenDB opens a SQLite database and creates it if needed.
func OpenDB(path string) (*sql.DB, error) {
	return sqliteutil.Open(path)
}
