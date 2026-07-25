// Package payment handles the payment lifecycle: orders, checkout, webhooks, and reports.
package payment

import (
	"database/sql"
	"errors"
	"fmt"
)

// ---------------------------------------------------------------------------
// Checkout
// ---------------------------------------------------------------------------

// CheckoutResult holds the result of creating a checkout session.
type CheckoutResult struct {
	SessionID   string `json:"session_id"`
	CheckoutURL string `json:"checkout_url"`
	QRCodeURL   string `json:"qrcode_url,omitempty"`
}

// ---------------------------------------------------------------------------
// Webhook
// ---------------------------------------------------------------------------

// Webhook event types from payment providers.
const (
	EventPaymentSucceeded = "payment.succeeded"
)

// Provider names.
const (
	ProviderDodo  = "dodo"
	ProviderXunhu = "xunhu"
)

// WebhookEvent is a verified webhook event from a payment provider.
type WebhookEvent struct {
	Type string
	Data WebhookEventData
}

// WebhookEventData holds the extracted fields from a webhook event.
type WebhookEventData struct {
	OrderID   string
	Amount    int
	Email     string
	PaymentID string
}

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	ErrUnknownProvider = errors.New("payment: unknown provider")
)


// OpenDB opens a SQLite database and creates it if needed.
func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	return db, nil
}
