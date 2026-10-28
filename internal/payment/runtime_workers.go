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
			metrics.Default.Inc("liki_web_payment_fulfillment_total", map[string]string{"status": "retry_or_failed"})
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
		metrics.Default.Inc("liki_web_payment_fulfillment_total", map[string]string{"status": "succeeded"})
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
			metrics.Default.Inc("liki_web_payment_notification_total", map[string]string{"result": "failed"})
			if markErr := r.notifications.MarkFailed(ctx, message.ID, err.Error()); markErr != nil {
				slog.ErrorContext(ctx, "payment: mark notification failed", "notification_id", message.ID, "err", markErr)
			}
			continue
		}
		metrics.Default.Inc("liki_web_payment_notification_total", map[string]string{"result": "sent"})
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
