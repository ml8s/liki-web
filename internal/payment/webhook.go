package payment

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"liki-web/internal/httputil"
	"liki-web/internal/product"
)

// WebhookStore is the store interface needed by the shared webhook handler.
type WebhookStore interface {
	MarkPaidIdempotent(ctx context.Context, orderID, paymentID string) (bool, error)
}

// WebhookPipeline is the pipeline interface needed by the shared webhook handler.
type WebhookPipeline interface {
	Run(ctx context.Context, orderID string) error
}

// HandleWebhook returns a shared webhook handler for payment notifications.
// logPrefix is used in log messages (e.g. "hepan", "mingshu", "naming").
// pipeline and onReady are optional (nil for products without async generation).
func HandleWebhook(store WebhookStore, svc *Service, logPrefix string, onPaid product.OnPaid, pipeline WebhookPipeline, onReady product.OnReportReady) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			slog.Warn(logPrefix+": read webhook body", "err", err)
			httputil.RespondError(w, http.StatusBadRequest, "bad_request", "读取失败")
			return
		}

		event, err := svc.VerifyWebhook(body, r.Header)
		if err != nil {
			slog.Warn(logPrefix+": webhook verify", "err", err)
			httputil.RespondError(w, http.StatusBadRequest, "webhook_failed", err.Error())
			return
		}

		if event.Type != EventPaymentSucceeded {
			slog.Info(logPrefix+": non-payment event", "type", event.Type)
			httputil.RespondJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
			return
		}

		if event.Data.OrderID == "" {
			slog.Error(logPrefix + ": webhook empty order_id")
			httputil.RespondError(w, http.StatusBadRequest, "webhook_failed", "empty order_id")
			return
		}

		newPayment, err := store.MarkPaidIdempotent(r.Context(), event.Data.OrderID, event.Data.PaymentID)
		if err != nil {
			slog.Error(logPrefix+": mark paid", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "webhook_failed", "处理失败")
			return
		}

		if newPayment && onPaid != nil {
			onPaid(r.Context(), event.Data.OrderID)
		}

		if newPayment && pipeline != nil {
			go func() {
				orderID := event.Data.OrderID
				bgCtx := context.WithoutCancel(r.Context())
				if err := pipeline.Run(bgCtx, orderID); err != nil {
					slog.Error(logPrefix+": pipeline run", "err", err)
					return
				}
				if onReady != nil {
					onReady(bgCtx, orderID)
				}
			}()
		}

		httputil.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
