package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ml8s/liki-web/internal/metrics"
	"net/http"
)

// HandleWebhookEventWithAck returns the provider-specific acknowledgement body.
// Xunhu requires exactly "success"; an otherwise successful empty response can
// make the provider retry a payment that has already been captured.
// ErrWebhookVerification marks signature/authentication failures. Providers
// must not retry those (the request is hostile or malformed), so the HTTP
// layer maps it to 400; transient failures map to 500 instead.
var ErrWebhookVerification = errors.New("payment: webhook verification failed")

func (r *Runtime) HandleWebhookEventWithAck(ctx context.Context, raw []byte, header http.Header) ([]byte, error) {
	event, err := r.verifyWebhook(raw, header)
	if err != nil {
		metrics.Default.Inc("payment_webhook_total", map[string]string{"result": "verify_failed"})
		return nil, fmt.Errorf("%w: %v", ErrWebhookVerification, err)
	}
	if r.subscriptionProcessor != nil && (IsSubscriptionLifecycle(event.Type) ||
		(event.Type == EventPaymentSucceeded && event.Data.SubscriptionID != "") ||
		(event.Type == EventPaymentRefunded && isSubscriptionRefund(event.Data))) {
		if err := r.subscriptionProcessor.HandleSubscriptionEvent(ctx, *event); err != nil {
			metrics.Default.Inc("payment_webhook_total", map[string]string{"result": "subscription_failed"})
			return nil, err
		}
		metrics.Default.Inc("payment_webhook_total", map[string]string{"result": "subscription_handled"})
		return r.webhookAck(event.Provider)
	}
	if event.Type != EventPaymentSucceeded && event.Type != EventPaymentRefunded {
		metrics.Default.Inc("payment_webhook_total", map[string]string{"result": "ignored"})
		slog.InfoContext(ctx, "payment: ignored webhook", "provider", event.Provider, "type", event.Type)
		return r.webhookAck(event.Provider)
	}
	intent, err := r.intents.GetIntent(ctx, event.Data.OrderID)
	if err != nil {
		return nil, fmt.Errorf("payment: webhook order %q: %w", event.Data.OrderID, err)
	}
	if _, err := r.events.Record(ctx, intent.ID, PaymentEventWebhook, *event); err != nil {
		return nil, err
	}
	if event.Type == EventPaymentRefunded {
		if err := r.captureRefund(ctx, event, intent); err != nil {
			metrics.Default.Inc("payment_webhook_total", map[string]string{"result": "refund_failed"})
			return nil, err
		}
		metrics.Default.Inc("payment_webhook_total", map[string]string{"result": "refunded"})
		metrics.Default.Inc("payment_refunded_total", map[string]string{"provider": string(intent.Provider)})
		return r.webhookAck(event.Provider)
	}
	if err := r.capture(ctx, event, intent); err != nil {
		metrics.Default.Inc("payment_webhook_total", map[string]string{"result": "capture_failed"})
		return nil, err
	}
	metrics.Default.Inc("payment_webhook_total", map[string]string{"result": "captured"})
	return r.webhookAck(event.Provider)
}

func isSubscriptionRefund(data WebhookEventData) bool {
	return data.SubscriptionID != "" ||
		data.Metadata[MetadataLikiBillingMode] == "subscription" ||
		data.Metadata[MetadataLikiSubscriptionKey] != ""
}

func (r *Runtime) webhookAck(provider Provider) ([]byte, error) {
	gateway, err := r.gateway(provider)
	if err != nil {
		return nil, err
	}
	return gateway.WebhookAck(), nil
}

func (r *Runtime) verifyWebhook(raw []byte, header http.Header) (*WebhookEvent, error) {
	var errs []error
	for provider, gateway := range r.gateways {
		event, err := gateway.VerifyWebhook(raw, header)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", provider, err))
			continue
		}
		if event == nil {
			continue
		}
		if event.Provider == "" {
			event.Provider = provider
		}
		return event, nil
	}
	return nil, fmt.Errorf("payment: verify webhook: %w", errors.Join(errs...))
}
