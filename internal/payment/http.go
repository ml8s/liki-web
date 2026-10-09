package payment

import (
	"io"
	"log/slog"
	"net/http"

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

func (runtime *Runtime) HandleFulfillmentStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	intent, err := runtime.GetIntent(r.Context(), id)
	if err != nil || !runtime.authorizeIntentAccess(r, intent) {
		httputil.RespondAppError(w, httputil.ErrOrderNotFound)
		return
	}
	task, found, err := runtime.GetFulfillmentStatus(r.Context(), id)
	if err != nil {
		slog.Error("payment: fulfillment status", "intent_id", id, "err", err)
		httputil.RespondAppError(w, httputil.ErrInternal)
		return
	}
	status := string(FulfillmentQueued)
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
			slog.Error("payment: return", "intent_id", id, "err", err)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		if intent.Status == IntentPaid && intent.Email != "" {
			if err := httputil.SetJWTCookie(w, intent.Email, intent.BusinessOrderID); err != nil {
				slog.Error("payment: return auth cookie", "intent_id", intent.ID, "err", err)
			}
		}
		slog.Info("payment: return redirect", "intent_id", intent.ID, "status", intent.Status, "location", location)
		http.Redirect(w, r, location, http.StatusSeeOther)
	}
}

func (runtime *Runtime) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		slog.Error("payment: read webhook body", "err", err)
		httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "read failed")
		return
	}
	ack, err := runtime.HandleWebhookEventWithAck(r.Context(), body, r.Header)
	if err != nil {
		slog.Error("payment: webhook", "err", err)
		httputil.RespondError(w, http.StatusBadRequest, "webhook_failed", "webhook failed")
		return
	}
	if len(ack) > 0 {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(ack)
		return
	}
	w.WriteHeader(http.StatusOK)
}
