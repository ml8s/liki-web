package payment

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"liki-web/internal/httputil"
	"liki-web/internal/product"
)

type createIntentRequest struct {
	Product         string `json:"product"`
	BusinessOrderID string `json:"business_order_id"`
	Provider        string `json:"provider"`
	Locale          string `json:"locale"`
}

func (r createIntentRequest) Validate() error {
	if r.Product == "" || r.BusinessOrderID == "" {
		return errors.New("product and business_order_id are required")
	}
	return nil
}

func (runtime *Runtime) HandleCreateIntent(w http.ResponseWriter, r *http.Request) {
	{
		req, ok := httputil.DecodeAndValidate[createIntentRequest](w, r)
		if !ok {
			return
		}
		if req.BusinessOrderID == "" {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "business_order_id is required")
			return
		}
		intent, err := runtime.CreateIntent(r.Context(), CreateIntentInput{
			Product:         product.Product(req.Product),
			BusinessOrderID: req.BusinessOrderID,
			Provider:        Provider(req.Provider),
			Locale:          req.Locale,
		})
		if err != nil {
			slog.Error("payment: create intent", "err", err)
			httputil.RespondError(w, http.StatusBadRequest, "payment_intent_failed", err.Error())
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]any{
			"id":                intent.ID,
			"business_order_id": intent.BusinessOrderID,
			"product":           intent.Product,
			"provider":          intent.Provider,
			"status":            intent.Status,
			"amount_minor":      intent.AmountMinor,
			"currency":          intent.Currency,
			"success_url":       intent.SuccessURL,
			"cancel_url":        intent.FailureURL,
		})
	}
}

func (runtime *Runtime) HandleStartCheckout(w http.ResponseWriter, r *http.Request) {
	{
		id := r.PathValue("id")
		if id == "" {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "missing payment intent id")
			return
		}
		result, intent, err := runtime.StartCheckout(r.Context(), id)
		if err != nil {
			slog.Error("payment: start checkout", "intent_id", id, "err", err)
			httputil.RespondError(w, http.StatusBadRequest, "checkout_failed", err.Error())
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]any{
			"payment_intent_id": intent.ID,
			"provider":          intent.Provider,
			"status":            intent.Status,
			"checkout_url":      result.CheckoutURL,
			"qr_url":            result.QRCodeURL,
			"status_token":      result.StatusToken,
			"success_url":       intent.SuccessURL,
			"cancel_url":        intent.FailureURL,
		})
	}
}

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
		http.Error(w, "read failed", http.StatusBadRequest)
		return
	}
	ack, err := runtime.HandleWebhookEventWithAck(r.Context(), body, r.Header)
	if err != nil {
		slog.Error("payment: webhook", "err", err)
		http.Error(w, "webhook failed", http.StatusBadRequest)
		return
	}
	if len(ack) > 0 {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(ack)
		return
	}
	w.WriteHeader(http.StatusOK)
}
