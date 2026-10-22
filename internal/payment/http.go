package payment

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ml8s/liki-web/internal/httputil"
)

func (runtime *Runtime) HandleIntentStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	intent, err := runtime.GetIntent(r.Context(), id)
	if err != nil || !runtime.authorizeIntentAccess(r, intent) {
		httputil.RespondAppError(w, httputil.ErrOrderNotFound)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{
		"id":                intent.ID,
		"business_order_id": intent.BusinessOrderID,
		"status":            intent.Status,
		"success_url":       intent.SuccessURL,
		"cancel_url":        intent.FailureURL,
	})
}

func (runtime *Runtime) HandlePaymentCompletionStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	intent, err := runtime.GetIntent(r.Context(), id)
	if err != nil || !runtime.authorizeIntentAccess(r, intent) {
		httputil.RespondAppError(w, httputil.ErrOrderNotFound)
		return
	}
	task, found, err := runtime.GetPaymentCompletionStatus(r.Context(), id)
	if err != nil {
		slog.ErrorContext(r.Context(), "payment: fulfillment status", "intent_id", id, "err", err)
		httputil.RespondAppError(w, httputil.ErrInternal)
		return
	}
	status := string(PaymentCompletionQueued)
	if found {
		status = string(task.Status)
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{
		"id":                 intent.ID,
		"payment_status":     intent.Status,
		"fulfillment_status": status,
	})
}

func (runtime *Runtime) HandlePaymentReturn(w http.ResponseWriter, r *http.Request) {
	{
		id := r.PathValue("id")
		intent, location, err := runtime.HandleReturn(r.Context(), id)
		if err != nil {
			slog.ErrorContext(r.Context(), "payment: return", "intent_id", id, "err", err)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		if intent.Status == IntentPaid && intent.Email != "" {
			if err := httputil.SetJWTCookie(w, intent.Email, intent.BusinessOrderID); err != nil {
				slog.ErrorContext(r.Context(), "payment: return auth cookie", "intent_id", intent.ID, "err", err)
			}
		}
		// Log the path only: the query carries a 1-hour payment-status
		// capability token that must not leak into logs.
		slog.InfoContext(r.Context(), "payment: return redirect", "intent_id", intent.ID, "status", intent.Status,
			"location", strings.SplitN(location, "?", 2)[0])
		http.Redirect(w, r, location, http.StatusSeeOther)
	}
}

func (runtime *Runtime) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		slog.ErrorContext(r.Context(), "payment: read webhook body", "err", err)
		httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "read failed")
		return
	}
	ack, err := runtime.HandleWebhookEventWithAck(r.Context(), body, r.Header)
	if err != nil {
		slog.ErrorContext(r.Context(), "payment: webhook", "err", err)
		// Verification failures and unknown orders are the provider's fault:
		// 400 tells it to stop retrying. Transient failures (DB busy) must be
		// 500 so the provider's at-least-once redelivery kicks in.
		if errors.Is(err, ErrIntentNotFound) || errors.Is(err, ErrWebhookVerification) {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "webhook failed")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "webhook_failed", "webhook failed")
		return
	}
	if len(ack) > 0 {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(ack)
		return
	}
	w.WriteHeader(http.StatusOK)
}
