package sales

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
)

type checkoutView struct {
	OrderID         string `json:"order_id"`
	PaymentIntentID string `json:"payment_intent_id"`
	Provider        string `json:"provider"`
	Status          string `json:"status"`
	CheckoutURL     string `json:"checkout_url"`
	QRCodeURL       string `json:"qr_code_url,omitempty"`
	StatusToken     string `json:"status_token"`
}

// orderView is the owner-facing order projection. Authentication already proves
// ownership, so user and email identifiers are not part of the response.
type orderView struct {
	ID              string          `json:"id"`
	ProductID       product.Product `json:"product_id"`
	AmountMinor     int64           `json:"amount_minor"`
	Currency        string          `json:"currency"`
	Status          OrderStatus     `json:"status"`
	PaymentIntentID string          `json:"payment_intent_id"`
	IntentStatus    string          `json:"intent_status,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	PaidAt          *time.Time      `json:"paid_at"`
}

func newOrderView(order Order, intentStatus string) orderView {
	return orderView{
		ID: order.ID, ProductID: order.ProductID,
		AmountMinor: order.AmountMinor, Currency: order.Currency,
		Status: order.Status, PaymentIntentID: order.PaymentIntentID,
		IntentStatus: intentStatus,
		CreatedAt:    order.CreatedAt, UpdatedAt: order.UpdatedAt, PaidAt: order.PaidAt,
	}
}

func (h *Handler) HandleOrders(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	orders, err := h.orders.ListOrdersForUser(r.Context(), user.ID, 50)
	if err != nil {
		httputil.InternalError(w)
		return
	}
	intentIDs := make([]string, 0, len(orders))
	for _, order := range orders {
		if order.PaymentIntentID != "" {
			intentIDs = append(intentIDs, order.PaymentIntentID)
		}
	}
	intents, err := h.runtime.GetIntents(r.Context(), intentIDs)
	if err != nil {
		slog.ErrorContext(r.Context(), "sales: annotate orders with intents", "user_id", user.ID, "err", err)
		httputil.InternalError(w)
		return
	}
	items := make([]orderView, 0, len(orders))
	for _, order := range orders {
		intentStatus := ""
		if intent, found := intents[order.PaymentIntentID]; found {
			intentStatus = string(intent.Status)
		}
		items = append(items, newOrderView(order, intentStatus))
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"orders": items})
}

// HandleOrderDetail returns one owned order plus its payment intent facts
// (provider, status, timestamps, refunds) for the order detail drawer.
func (h *Handler) HandleOrderDetail(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	order, err := h.orders.GetOrderForUser(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			httputil.NotFound(w)
			return
		}
		slog.ErrorContext(r.Context(), "sales: load order detail", "order_id", r.PathValue("id"), "err", err)
		httputil.InternalError(w)
		return
	}
	item := newOrderView(order, "")
	payload := map[string]any{"order": item}
	if order.PaymentIntentID != "" {
		intent, err := h.runtime.GetIntent(r.Context(), order.PaymentIntentID)
		switch {
		case err == nil:
			item.IntentStatus = string(intent.Status)
			payload["order"] = item
			payload["payment"] = map[string]any{
				"provider":              string(intent.Provider),
				"status":                string(intent.Status),
				"created_at":            intent.CreatedAt,
				"updated_at":            intent.UpdatedAt,
				"paid_at":               intent.PaidAt,
				"refunded_at":           intent.RefundedAt,
				"refunded_amount_minor": intent.RefundedAmountMinor,
			}
		case errors.Is(err, payment.ErrIntentNotFound):
			// An order without a live intent still renders; payment stays absent.
		default:
			slog.ErrorContext(r.Context(), "sales: load intent for order detail", "order_id", order.ID, "err", err)
			httputil.InternalError(w)
			return
		}
	}
	httputil.RespondJSON(w, http.StatusOK, payload)
}

func (h *Handler) HandleCancelOrder(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	orderID := r.PathValue("id")
	order, err := h.orders.GetOrderForUser(r.Context(), user.ID, orderID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			httputil.NotFound(w)
			return
		}
		slog.ErrorContext(r.Context(), "sales: load order for cancel", "order_id", orderID, "err", err)
		httputil.InternalError(w)
		return
	}
	if order.Status != OrderPending {
		httputil.RespondError(w, http.StatusConflict, "order_not_pending", "订单当前不可取消")
		return
	}
	if err := h.orders.CancelOrder(r.Context(), user.ID, orderID); err != nil {
		// The pre-check passed, so a losing race against the payment webhook is
		// the realistic failure; report it as a conflict rather than 404.
		if errors.Is(err, ErrOrderNotFound) {
			httputil.RespondError(w, http.StatusConflict, "order_not_pending", "订单当前不可取消")
			return
		}
		slog.ErrorContext(r.Context(), "sales: cancel order", "order_id", orderID, "err", err)
		httputil.InternalError(w)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]string{"status": string(OrderCanceled)})
}

// HandleResumeCheckout re-opens payment for a pending order. Providers expire
// checkout sessions, so resume always starts a fresh session for the same
// order and payment intent; the stored intent is what keeps this from creating
// duplicate orders.
func (h *Handler) HandleResumeCheckout(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	orderID := r.PathValue("id")
	order, err := h.orders.GetOrderForUser(r.Context(), user.ID, orderID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			httputil.NotFound(w)
			return
		}
		slog.ErrorContext(r.Context(), "sales: load order for resume", "order_id", orderID, "err", err)
		httputil.InternalError(w)
		return
	}
	if order.Status != OrderPending || order.PaymentIntentID == "" {
		httputil.RespondError(w, http.StatusConflict, "order_not_pending", "订单当前不可支付")
		return
	}
	intent, err := h.runtime.GetIntent(r.Context(), order.PaymentIntentID)
	if err != nil {
		if errors.Is(err, payment.ErrIntentNotFound) {
			httputil.RespondError(w, http.StatusConflict, "order_expired", "订单已过期，请重新下单")
			return
		}
		slog.ErrorContext(r.Context(), "sales: load intent for resume", "order_id", orderID, "err", err)
		httputil.InternalError(w)
		return
	}
	switch intent.Status {
	case payment.IntentExpired:
		httputil.RespondError(w, http.StatusConflict, "order_expired", "订单已过期，请重新下单")
		return
	case payment.IntentPaid:
		httputil.RespondError(w, http.StatusConflict, "order_already_paid", "订单已支付，请刷新后查看")
		return
	case payment.PaymentStatusRefunded:
		httputil.RespondError(w, http.StatusConflict, "order_refunded", "订单已退款")
		return
	}
	result, updated, err := h.runtime.StartCheckout(r.Context(), intent.ID)
	if err != nil {
		// StartCheckout itself guards paid/expired intents; map its guard
		// failures back onto the conflict vocabulary above.
		if errors.Is(err, payment.ErrIntentNotFound) {
			httputil.RespondError(w, http.StatusConflict, "order_expired", "订单已过期，请重新下单")
			return
		}
		slog.ErrorContext(r.Context(), "sales: resume checkout", "order_id", orderID, "err", err)
		httputil.RespondError(w, http.StatusBadGateway, "checkout_failed", "发起支付失败")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, checkoutView{
		OrderID:         order.ID,
		PaymentIntentID: intent.ID,
		Provider:        string(updated.Provider),
		Status:          string(updated.Status),
		CheckoutURL:     result.CheckoutURL,
		QRCodeURL:       result.QRCodeURL,
		StatusToken:     result.StatusToken,
	})
}
