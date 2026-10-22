package sales

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
)

func (h *Handler) handleSubscriptionCheckout(
	w http.ResponseWriter,
	r *http.Request,
	user auth.User,
	productID product.Product,
	currency product.Currency,
	locale string,
) {
	catalog := h.catalog
	if catalog == nil {
		catalog = product.NewCatalog(product.CatalogConfig{})
	}
	if _, err := catalog.Price(productID, product.BillingModeSubscription, currency); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "subscription_unavailable", "订阅在当前区域不可用")
		return
	}
	if err := h.subscriptions.RequireNoBlockingSubscription(r.Context(), user.ID); err != nil {
		if errors.Is(err, ErrSubscriptionConflict) {
			httputil.RespondError(w, http.StatusConflict, "subscription_conflict", "已有有效订阅")
			return
		}
		slog.ErrorContext(r.Context(), "sales: check blocking subscription", "user_id", user.ID, "err", err)
		httputil.InternalError(w)
		return
	}
	subscription, err := h.subscriptions.CreatePending(r.Context(), user.ID, user.Email, productID, currency)
	if err != nil {
		slog.ErrorContext(r.Context(), "sales: create subscription", "user_id", user.ID, "err", err)
		httputil.RespondError(w, http.StatusBadRequest, "subscription_failed", "创建订阅失败")
		return
	}
	result, err := h.subscriptionCheckout.CreateSubscriptionCheckout(r.Context(), payment.SubscriptionCheckoutRequest{
		Product:         string(productID),
		SubscriptionKey: subscription.ID,
		UserID:          user.ID,
		Email:           user.Email,
		Currency:        string(currency),
		ReturnURL:       h.publicBaseURL + "/app/chat?subscription_id=" + url.QueryEscape(subscription.ID),
		CancelURL:       h.publicBaseURL + "/app/chat?subscription_state=cancelled",
	})
	if err == nil {
		err = h.subscriptions.AttachCheckout(r.Context(), user.ID, subscription.ID, result.CheckoutSessionID)
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "sales: start subscription checkout", "subscription_id", subscription.ID, "locale", LocaleOrDefault(locale), "err", err)
		httputil.RespondError(w, http.StatusBadGateway, "checkout_failed", "发起订阅支付失败")
		return
	}
	httputil.RespondJSON(w, http.StatusCreated, map[string]any{
		"subscription_id": subscription.ID,
		"provider":        string(subscription.Provider),
		"status":          string(subscription.Status),
		"checkout_url":    result.CheckoutURL,
	})
}

func (h *Handler) HandleSubscriptions(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	items, err := h.subscriptions.ListByUser(r.Context(), user.ID, 50)
	if err != nil {
		httputil.InternalError(w)
		return
	}
	if h.subscriptionReconciler != nil && h.applySubscriptionEvent != nil {
		if err := h.reconcilePendingSubscriptions(r.Context(), user.ID); err != nil {
			slog.ErrorContext(r.Context(), "sales: reconcile pending subscriptions", "user_id", user.ID, "err", err)
		}
		items, err = h.subscriptions.ListByUser(r.Context(), user.ID, 50)
		if err != nil {
			httputil.InternalError(w)
			return
		}
	}
	views := make([]subscriptionView, 0, len(items))
	for _, item := range items {
		views = append(views, newSubscriptionView(item))
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"subscriptions": views})
}

// subscriptionView is the owner-facing subscription projection. Provider
// customer/checkout identifiers and sync bookkeeping are not browser facts.
type subscriptionView struct {
	ID                 string              `json:"id"`
	ProductID          product.Product     `json:"product_id"`
	BillingMode        product.BillingMode `json:"billing_mode"`
	Provider           string              `json:"provider"`
	Status             string              `json:"status"`
	CurrencyCode       string              `json:"currency_code"`
	CurrentPeriodStart *time.Time          `json:"current_period_start"`
	CurrentPeriodEnd   *time.Time          `json:"current_period_end"`
	NextBillingAt      *time.Time          `json:"next_billing_at"`
	CancelAtPeriodEnd  bool                `json:"cancel_at_period_end"`
	CancelledAt        *time.Time          `json:"cancelled_at"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

func newSubscriptionView(item Subscription) subscriptionView {
	return subscriptionView{
		ID: item.ID, ProductID: item.ProductID,
		BillingMode: item.BillingMode, Provider: string(item.Provider),
		Status: string(item.Status), CurrencyCode: string(item.CurrencyCode),
		CurrentPeriodStart: item.CurrentPeriodStart,
		CurrentPeriodEnd:   item.CurrentPeriodEnd,
		NextBillingAt:      item.NextBillingAt,
		CancelAtPeriodEnd:  item.CancelAtPeriodEnd,
		CancelledAt:        item.CancelledAt,
		CreatedAt:          item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

// subscriptionPaymentView omits ownership and provider bookkeeping while
// retaining the buyer-visible payment and refund facts.
type subscriptionPaymentView struct {
	ID                 string          `json:"id"`
	SubscriptionID     string          `json:"subscription_id"`
	ProductID          product.Product `json:"product_id"`
	Provider           string          `json:"provider"`
	ProviderPaymentID  string          `json:"provider_payment_id"`
	AmountMinorUnits   int64           `json:"amount_minor_units"`
	RefundedMinorUnits int64           `json:"refunded_minor_units"`
	CurrencyCode       string          `json:"currency_code"`
	Status             string          `json:"status"`
	PeriodStart        *time.Time      `json:"period_start"`
	PeriodEnd          *time.Time      `json:"period_end"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

func newSubscriptionPaymentView(item SubscriptionPayment) subscriptionPaymentView {
	return subscriptionPaymentView{
		ID: item.ID, SubscriptionID: item.SubscriptionID,
		ProductID: item.ProductID, Provider: string(item.Provider),
		ProviderPaymentID:  item.ProviderPaymentID,
		AmountMinorUnits:   item.AmountMinorUnits,
		RefundedMinorUnits: item.RefundedMinorUnits,
		CurrencyCode:       string(item.CurrencyCode), Status: string(item.Status),
		PeriodStart: item.PeriodStart, PeriodEnd: item.PeriodEnd,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func (h *Handler) reconcilePendingSubscriptions(ctx context.Context, userID string) error {
	pending, err := h.subscriptions.ListPendingWithCheckoutByUser(ctx, userID, 20)
	if err != nil {
		return err
	}
	for _, item := range pending {
		if item.ProviderCheckoutSessionID == "" {
			continue
		}
		event, err := h.subscriptionReconciler.FetchSubscriptionByCheckout(ctx, item.ProviderCheckoutSessionID)
		if err != nil {
			slog.ErrorContext(ctx, "sales: fetch pending subscription", "subscription_id", item.ID, "checkout_id", item.ProviderCheckoutSessionID, "err", err)
			continue
		}
		if event == nil {
			continue
		}
		if err := h.applySubscriptionEvent(ctx, *event); err != nil {
			slog.ErrorContext(ctx, "sales: apply reconciled subscription", "subscription_id", item.ID, "err", err)
		}
	}
	return nil
}

func (h *Handler) HandleSubscriptionPayments(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	items, err := h.subscriptions.ListPaymentsByUser(r.Context(), user.ID, 50)
	if err != nil {
		slog.ErrorContext(r.Context(), "sales: list subscription payments", "user_id", user.ID, "err", err)
		httputil.InternalError(w)
		return
	}
	views := make([]subscriptionPaymentView, 0, len(items))
	for _, item := range items {
		views = append(views, newSubscriptionPaymentView(item))
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"payments": views})
}

func (h *Handler) HandleSetCancellation(w http.ResponseWriter, r *http.Request) {
	h.setCancellation(w, r, true)
}

func (h *Handler) HandleDeleteCancellation(w http.ResponseWriter, r *http.Request) {
	h.setCancellation(w, r, false)
}

func (h *Handler) setCancellation(w http.ResponseWriter, r *http.Request, cancel bool) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	if err := h.subscriptions.SetCancellation(r.Context(), user.ID, r.PathValue("id"), cancel, h.subscriptionCancellation); err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			httputil.NotFound(w)
			return
		}
		slog.ErrorContext(r.Context(), "sales: set subscription cancellation", "user_id", user.ID, "subscription_id", r.PathValue("id"), "cancel", cancel, "err", err)
		httputil.RespondError(w, http.StatusBadGateway, "subscription_update_failed", "更新订阅状态失败")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"cancel_at_period_end": cancel})
}
