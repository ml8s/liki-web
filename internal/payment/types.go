// Package payment handles the payment lifecycle: orders, checkout, webhooks, and reports.
package payment

import (
	"errors"
	"time"

	"liki_web/internal/product"
)

// ---------------------------------------------------------------------------
// Order
// ---------------------------------------------------------------------------

// OrderStatus is the payment status of an order.
type OrderStatus string

const (
	OrderPending OrderStatus = "pending"
	OrderPaid    OrderStatus = "paid"
)

// Order holds the database record for a payment order.
type Order struct {
	OrderID       string
	Product       product.Product
	Amount        int
	Currency      string
	Provider      string
	Email         string
	ChartJSON     string
	LlmJSON       string
	BirthInfo     string
	ChatExpiresAt string
	Status        OrderStatus
	PaymentID     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ChatExpiryDuration is the chat window from payment success.
const ChatExpiryDuration = 7 * 24 * time.Hour

// DefaultChatExpiry returns the expiry time for a newly paid naming order.
func DefaultChatExpiry() string {
	return time.Now().Add(ChatExpiryDuration).UTC().Format(time.DateTime)
}

// ---------------------------------------------------------------------------
// Chat message
// ---------------------------------------------------------------------------

// Role is the chat message role.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ChatMessage holds a single persisted chat message.
type ChatMessage struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

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
	EventPaymentRefunded  = "payment.refunded"
	EventPaymentDisputed  = "payment.disputed"
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
// Report
// ---------------------------------------------------------------------------

// ReportData holds the full report data for a paid order.
type ReportData struct {
	OrderID   string          `json:"order_id"`
	Product   product.Product `json:"product"`
	Status    OrderStatus     `json:"status"`
	Email     string          `json:"email,omitempty"`
	ChartJSON string          `json:"chart_json"`
	LlmJSON   string          `json:"llm_json"`
}

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	ErrOrderNotFound   = errors.New("payment: order not found")
	ErrOrderNotPaid    = errors.New("payment: order not paid")
	ErrWebhookVerify   = errors.New("payment: webhook verification failed")
	ErrUnknownProvider = errors.New("payment: unknown provider")
)
