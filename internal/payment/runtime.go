package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ml8s/liki-web/internal/httputil"
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
func (r *Runtime) ProcessPaymentCompletionTasks(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 1
	}
	processed := 0
	for processed < limit {
		task, err := r.fulfillments.ClaimNext(ctx, time.Now().UTC())
		if errors.Is(err, ErrNoRunnablePaymentCompletion) {
			return processed, nil
		}
		if err != nil {
			return processed, err
		}

		intent, err := r.intents.GetIntent(ctx, task.IntentID)
		if err != nil {
			if markErr := r.fulfillments.MarkFailed(ctx, task.ID, err); markErr != nil {
				return processed, markErr
			}
			continue
		}
		flow, ok := r.flows[task.Product]
		if !ok {
			err = fmt.Errorf("%w: %s", ErrUnknownProduct, task.Product)
			if markErr := r.fulfillments.MarkFailed(ctx, task.ID, err); markErr != nil {
				return processed, markErr
			}
			metrics.Default.Inc("payment_fulfillment_total", map[string]string{"status": "retry_or_failed"})
			continue
		}

		taskCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
		if err := flow.OnPaid(taskCtx, intent); err != nil {
			cancel()
			if markErr := r.fulfillments.MarkFailed(ctx, task.ID, err); markErr != nil {
				return processed, markErr
			}
			continue
		}
		cancel()
		if err := r.fulfillments.MarkSucceeded(ctx, task.ID); err != nil {
			return processed, err
		}
		if r.fulfillmentObserver != nil {
			r.fulfillmentObserver(context.WithoutCancel(ctx), intent)
		}
		metrics.Default.Inc("payment_fulfillment_total", map[string]string{"status": "succeeded"})
		processed++
	}
	return processed, nil
}

func (r *Runtime) StartPaymentCompletionWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if err := r.fulfillments.RecoverRunning(ctx); err != nil {
		slog.ErrorContext(ctx, "payment: recover fulfillment tasks", "err", err)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := r.ProcessPaymentCompletionTasks(ctx, 20); err != nil {
				slog.ErrorContext(ctx, "payment: fulfillment worker", "err", err)
			}
		}
	}
}

func (r *Runtime) HandleReturn(ctx context.Context, intentID string) (PaymentIntent, string, error) {
	intent, err := r.intents.GetIntent(ctx, intentID)
	if err != nil {
		return PaymentIntent{}, "", err
	}
	flow, ok := r.flows[intent.Product]
	if !ok {
		return intent, "", fmt.Errorf("%w: %s", ErrUnknownProduct, intent.Product)
	}
	if intent.Status == IntentPaid {
		location := flow.SuccessRedirect(intent)
		token, err := httputil.NewCapabilityToken("payment-status", intent.ID, time.Now(), time.Hour)
		if err != nil {
			return intent, "", err
		}
		separator := "?"
		if strings.Contains(location, "?") {
			separator = "&"
		}
		location += separator + "payment_intent_id=" + url.QueryEscape(intent.ID) +
			"&status_token=" + url.QueryEscape(token)
		return intent, location, nil
	}

	// The browser return can race the provider webhook. Query the provider once
	// so the redirect reflects an already-completed payment instead of forcing
	// the user to wait for webhook or background reconciliation.
	intent, _, reconcileErr := r.reconcileIntent(ctx, intent)
	if reconcileErr != nil {
		slog.WarnContext(ctx, "payment: reconcile return", "intent_id", intentID, "provider", intent.Provider, "err", reconcileErr)
	}
	if intent.Status == IntentPaid {
		location := flow.SuccessRedirect(intent)
		token, tokenErr := httputil.NewCapabilityToken("payment-status", intent.ID, time.Now(), time.Hour)
		if tokenErr != nil {
			return intent, "", tokenErr
		}
		separator := "?"
		if strings.Contains(location, "?") {
			separator = "&"
		}
		location += separator + "payment_intent_id=" + url.QueryEscape(intent.ID) +
			"&status_token=" + url.QueryEscape(token)
		return intent, location, nil
	}
	return intent, flow.FailureRedirect(intent), nil
}

// RunRetention closes abandoned intents and removes historical sent email rows.
// Audit events and fulfillment terminal rows are intentionally retained.
func (r *Runtime) RunRetention(ctx context.Context, pendingAfter, sentNotificationAfter time.Duration) (int64, int64, error) {
	intents, err := r.intents.MarkStaleIntentsExpired(ctx, pendingAfter)
	if err != nil {
		return 0, 0, err
	}
	notifications, err := r.notifications.DeleteSentBefore(ctx, sentNotificationAfter)
	if err != nil {
		return intents, 0, err
	}
	return intents, notifications, nil
}

func (r *Runtime) GetPaymentCompletionStatus(ctx context.Context, intentID string) (PaymentCompletionTask, bool, error) {
	return r.fulfillments.GetByIntent(ctx, intentID)
}

func (r *Runtime) authorizeIntentAccess(req *http.Request, intent PaymentIntent) bool {
	if token := req.URL.Query().Get("status_token"); token != "" {
		return httputil.VerifyCapabilityToken("payment-status", intent.ID, token, time.Now())
	}
	if email, orderID, ok := httputil.JWTAuth(req); ok {
		return email == intent.Email && orderID == intent.BusinessOrderID
	}
	return false
}

func (r *Runtime) StartNotificationWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.ProcessPendingNotifications(ctx, 100); err != nil {
				slog.ErrorContext(ctx, "payment: notification worker", "err", err)
			}
		}
	}
}

func (r *Runtime) StartReconciliationWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.ReconcilePendingIntents(ctx, 20); err != nil {
				slog.ErrorContext(ctx, "payment: reconciliation worker", "err", err)
			}
		}
	}
}

func (r *Runtime) ProcessPendingNotifications(ctx context.Context, limit int) error {
	messages, err := r.notifications.Pending(ctx, limit)
	if err != nil {
		return err
	}
	for _, message := range messages {
		if err := r.sender.Send(ctx, EmailMessage{To: message.To, Subject: message.Subject, Body: message.Body}); err != nil {
			metrics.Default.Inc("payment_notification_total", map[string]string{"result": "failed"})
			if markErr := r.notifications.MarkFailed(ctx, message.ID, err.Error()); markErr != nil {
				slog.ErrorContext(ctx, "payment: mark notification failed", "notification_id", message.ID, "err", markErr)
			}
			continue
		}
		metrics.Default.Inc("payment_notification_total", map[string]string{"result": "sent"})
		if err := r.notifications.MarkSent(ctx, message.ID); err != nil {
			slog.ErrorContext(ctx, "payment: mark notification sent", "notification_id", message.ID, "err", err)
		}
	}
	return nil
}

func (r *Runtime) queuePaidNotifications(ctx context.Context, intent PaymentIntent) error {
	flow, ok := r.flows[intent.Product]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownProduct, intent.Product)
	}
	notice := newPaidOrderNotice(intent, flow, r.baseURL)
	customer, err := buildCustomerReceipt(notice)
	if err != nil {
		return fmt.Errorf("payment: build customer receipt: %w", err)
	}
	if err := r.notifications.EnqueueOnce(ctx, intent.ID, NotificationCustomerReceipt, customer); err != nil {
		return err
	}
	if r.adminEmail == "" {
		return nil
	}
	admin, err := buildAdminNewOrder(notice, r.adminEmail)
	if err != nil {
		return fmt.Errorf("payment: build admin order notification: %w", err)
	}
	return r.notifications.EnqueueOnce(ctx, intent.ID, NotificationAdminNewOrder, admin)
}

func (r *Runtime) gateway(provider Provider) (Gateway, error) {
	gateway, ok := r.gateways[provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, provider)
	}
	return gateway, nil
}
