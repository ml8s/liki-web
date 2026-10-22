package payment

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ml8s/liki-web/internal/metrics"
)

type PaymentStatus string

const (
	IntentCreated         PaymentStatus = "created"
	IntentCheckoutCreated PaymentStatus = "checkout_created"
	IntentPaid            PaymentStatus = "paid"
	PaymentStatusRefunded PaymentStatus = "refunded"
	IntentExpired         PaymentStatus = "expired"
)

type BusinessOrder struct {
	ID          string
	UserID      string
	Product     string
	AmountMinor int64
	Currency    string
	Email       string
	Locale      string
	Status      string
}

type PaymentIntent struct {
	ID                        string
	BusinessOrderID           string
	UserID                    string
	Product                   string
	Provider                  Provider
	Status                    PaymentStatus
	AmountMinor               int64
	Currency                  string
	Email                     string
	Locale                    string
	Metadata                  map[string]string
	ProviderCheckoutSessionID string
	ProviderPaymentID         string
	CheckoutURL               string
	QRCodeURL                 string
	ReturnURL                 string
	SuccessURL                string
	FailureURL                string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
	PaidAt                    *time.Time
	RefundedAt                *time.Time
	RefundedAmountMinor       int64
}

type CheckoutRequest struct {
	IntentID        string
	BusinessOrderID string
	Product         string
	AmountMinor     int64
	Currency        string
	Email           string
	Locale          string
	Title           string
	Metadata        map[string]string
	ReturnURL       string
	CancelURL       string
}

// validateURLs rejects what the providers reject. A relative cancel URL makes
// Dodo answer 400 INVALID_REQUEST_PARAMETERS ("cancel_url: must be a valid
// URL") and fails the whole checkout, so the invariant is enforced here instead
// of being rediscovered from a provider response. Empty URLs mean "omit".
func (r CheckoutRequest) validateURLs() error {
	for _, field := range []struct{ name, raw string }{
		{"return_url", r.ReturnURL},
		{"cancel_url", r.CancelURL},
	} {
		if field.raw == "" {
			continue
		}
		if !strings.HasPrefix(field.raw, "http://") && !strings.HasPrefix(field.raw, "https://") {
			return fmt.Errorf("payment: checkout %s must be an absolute URL, got %q", field.name, field.raw)
		}
	}
	return nil
}

type Gateway interface {
	CreateCheckout(ctx context.Context, req CheckoutRequest) (*CheckoutResult, error)
	VerifyWebhook(raw []byte, header http.Header) (*WebhookEvent, error)
	// FetchPayment asks the provider for the current state of one checkout intent.
	// A nil event means the provider has no matching succeeded payment.
	FetchPayment(ctx context.Context, intent PaymentIntent) (*WebhookEvent, error)
	// WebhookAck returns the provider-required successful HTTP response body.
	WebhookAck() []byte
}

type ProductFlow interface {
	Product() string
	// DisplayName is the human-facing product name used in payment receipts.
	DisplayName(locale string) string
	PrepareCheckout(ctx context.Context, businessOrderID string) (BusinessOrder, error)
	OnPaid(ctx context.Context, intent PaymentIntent) error
	SuccessRedirect(intent PaymentIntent) string
	FailureRedirect(intent PaymentIntent) string
}

type CreateIntentInput struct {
	Product         string
	BusinessOrderID string
	UserID          string
	// Provider is a client hint only. The server route policy is authoritative.
	Provider Provider
	Locale   string
}

type RuntimeOptions struct {
	BaseURL    string
	AdminEmail string
}

// PaymentCompletionObserver is a non-critical notification emitted after a product
// flow's idempotent OnPaid succeeds and the fulfillment task is terminal.
// Observers must be idempotent and must not block or fail product fulfillment.
type PaymentCompletionObserver func(ctx context.Context, intent PaymentIntent)

// SubscriptionEventProcessor owns the recurring-payment lifecycle. It is called
// before intent lookup so provider subscription events cannot be mistaken for
// one-time order events.
type SubscriptionEventProcessor interface {
	HandleSubscriptionEvent(ctx context.Context, event WebhookEvent) error
}

type Runtime struct {
	gateways              map[Provider]Gateway
	flows                 map[string]ProductFlow
	intents               *IntentStore
	notifications         *NotificationStore
	fulfillments          *PaymentCompletionTaskStore
	events                *PaymentEventStore
	routePolicy           ProviderRoutePolicy
	sender                EmailSender
	adminEmail            string
	baseURL               string
	fulfillmentObserver   PaymentCompletionObserver
	refundObserver        func(ctx context.Context, event WebhookEvent, intent PaymentIntent) error
	subscriptionProcessor SubscriptionEventProcessor
}

func NewRuntime(
	gateways map[Provider]Gateway,
	flows map[string]ProductFlow,
	intents *IntentStore,
	notifications *NotificationStore,
	fulfillments *PaymentCompletionTaskStore,
	events *PaymentEventStore,
	sender EmailSender,
	options RuntimeOptions,
) *Runtime {
	return &Runtime{
		gateways:      gateways,
		flows:         flows,
		intents:       intents,
		notifications: notifications,
		fulfillments:  fulfillments,
		events:        events,
		routePolicy:   CurrencyRoutePolicy{},
		sender:        sender,
		adminEmail:    options.AdminEmail,
		baseURL:       options.BaseURL,
	}
}

func (r *Runtime) SetPaymentCompletionObserver(observer PaymentCompletionObserver) {
	r.fulfillmentObserver = observer
}

func (r *Runtime) SetRefundObserver(observer func(ctx context.Context, event WebhookEvent, intent PaymentIntent) error) {
	r.refundObserver = observer
}

func (r *Runtime) SetSubscriptionProcessor(processor SubscriptionEventProcessor) {
	r.subscriptionProcessor = processor
}

func (r *Runtime) ReconcilePendingIntents(ctx context.Context, limit int) error {
	if limit <= 0 {
		limit = 20
	}
	intents, err := r.intents.ListIntentsForReconciliation(ctx, limit)
	if err != nil {
		return err
	}
	for _, intent := range intents {
		if _, changed, err := r.reconcileIntent(ctx, intent); err != nil {
			slog.ErrorContext(ctx, "payment: reconcile intent", "intent_id", intent.ID, "err", err)
		} else if changed {
			metrics.Default.Inc("payment_reconcile_total", map[string]string{"result": "captured"})
		}
	}
	return nil
}

// reconcileIntent asks the provider for the current state of one pending
// intent and captures it if the provider reports a successful payment.
func (r *Runtime) reconcileIntent(ctx context.Context, intent PaymentIntent) (PaymentIntent, bool, error) {
	gateway, err := r.gateway(intent.Provider)
	if err != nil {
		return intent, false, err
	}
	event, err := gateway.FetchPayment(ctx, intent)
	if err != nil {
		return intent, false, err
	}
	if event == nil || event.Type != EventPaymentSucceeded {
		return intent, false, nil
	}
	if _, err := r.events.Record(ctx, intent.ID, PaymentEventReconciliation, *event); err != nil {
		return intent, false, err
	}
	if err := r.capture(ctx, event, intent); err != nil {
		return intent, false, err
	}
	updated, err := r.intents.GetIntent(ctx, intent.ID)
	return updated, true, err
}

func (r *Runtime) captureRefund(ctx context.Context, event *WebhookEvent, intent PaymentIntent) error {
	if intent.Provider != event.Provider {
		return fmt.Errorf("payment: refund provider %q does not match intent %q", event.Provider, intent.Provider)
	}
	if intent.Status != IntentPaid && intent.Status != PaymentStatusRefunded {
		return fmt.Errorf("payment: refund for intent %s in state %q", intent.ID, intent.Status)
	}
	if intent.Status == PaymentStatusRefunded && intent.RefundedAmountMinor >= intent.AmountMinor {
		return nil
	}
	if event.Data.Amount != 0 && event.Data.Amount > intent.AmountMinor-intent.RefundedAmountMinor {
		return fmt.Errorf("payment: refund exceeds remaining amount for intent %s: got %d remaining %d",
			intent.ID, event.Data.Amount, intent.AmountMinor-intent.RefundedAmountMinor)
	}
	updated, record, err := r.intents.RecordRefund(ctx, intent, event.Data.RefundReference(), event.Data.Amount, event.Data.Currency)
	if err != nil {
		return err
	}
	// Normalize the observer value: a zero provider amount means
	// "refund the remainder", which RecordRefund has already resolved to the
	// recorded amount. Downstream ledgers (referral) require a positive value.
	if event.Data.Amount == 0 {
		event.Data.Amount = record.Amount
	}
	if (record.New || !record.Notified) && r.refundObserver != nil {
		if err := r.refundObserver(context.WithoutCancel(ctx), *event, updated); err != nil {
			return err
		}
		if err := r.intents.MarkRefundNotified(context.WithoutCancel(ctx), record.ID); err != nil {
			return err
		}
	}
	metrics.Default.Inc("payment_refund_total", map[string]string{"provider": string(updated.Provider)})
	return nil
}

func (r *Runtime) capture(ctx context.Context, event *WebhookEvent, intent PaymentIntent) error {
	if intent.Provider != event.Provider {
		return fmt.Errorf("payment: webhook provider %q does not match intent %q", event.Provider, intent.Provider)
	}
	if event.Data.Amount != intent.AmountMinor {
		return fmt.Errorf("payment: amount mismatch for intent %s: got %d want %d", intent.ID, event.Data.Amount, intent.AmountMinor)
	}
	if intent.Currency != "" && event.Data.Currency != "" && intent.Currency != event.Data.Currency {
		return fmt.Errorf("payment: currency mismatch for intent %s: got %s want %s", intent.ID, event.Data.Currency, intent.Currency)
	}

	updated, _, err := r.intents.MarkPaid(ctx, intent.ID, event.Data.PaymentID)
	if err != nil {
		return err
	}
	// A duplicate webhook heals either missing durable stream, but never resets
	// a queued, running, succeeded, or dead fulfillment task.
	if err := r.fulfillments.EnsureQueued(ctx, updated); err != nil {
		return err
	}
	// Notifications and product fulfillment are separate durable work streams so
	// a slow or failing generator cannot block the provider acknowledgement.
	return r.queuePaidNotifications(ctx, updated)
}

func (r *Runtime) availableProviders() []Provider {
	providers := make([]Provider, 0, len(r.gateways))
	for provider := range r.gateways {
		providers = append(providers, provider)
	}
	return providers
}

// ProcessPaymentCompletionTasks executes at most limit queued tasks. Claiming marks a
// task running before OnPaid starts; product flows must be idempotent.
