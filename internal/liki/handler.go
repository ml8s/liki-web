package liki

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
)

type RequireUserFunc func(http.HandlerFunc) http.HandlerFunc
type LimitFunc func(http.HandlerFunc) http.HandlerFunc

// ErrSubscriptionConflict reports that a blocking subscription already exists.
// SubscriptionCheckoutCreator is owned by the provider gateway. Keeping it as
// a local interface avoids coupling the product HTTP layer to Dodo.
type SubscriptionCheckoutCreator interface {
	CreateSubscriptionCheckout(context.Context, payment.SubscriptionCheckoutRequest) (*payment.CheckoutResult, error)
}

// SubscriptionReconciler heals a lost webhook through the provider's
// authoritative checkout/payment/subscription lookup chain.
type SubscriptionReconciler interface {
	FetchSubscriptionByCheckout(context.Context, string) (*payment.WebhookEvent, error)
}

type SubscriptionEventApplier func(context.Context, payment.WebhookEvent) error

// Handler registers all Liki Chat HTTP routes: entitlement listing,
// product catalog, checkout, orders, and chat sessions/messages.
type Handler struct {
	orders                   *OrderStore
	entitlements             *entitlement.Store
	service                  *Service
	runtime                  *payment.Runtime
	requireUser              RequireUserFunc
	currency                 CurrencyResolver
	catalog                  *Catalog
	subscriptions            *SubscriptionStore
	subscriptionCheckout     SubscriptionCheckoutCreator
	subscriptionCancellation payment.SubscriptionCancellationGateway
	subscriptionReconciler   SubscriptionReconciler
	applySubscriptionEvent   SubscriptionEventApplier
	publicBaseURL            string
}

type HandlerOption func(*Handler)

func WithCurrencyResolver(resolver CurrencyResolver) HandlerOption {
	return func(h *Handler) {
		if resolver != nil {
			h.currency = resolver
		}
	}
}

func WithCommerceCatalog(catalog *Catalog) HandlerOption {
	return func(h *Handler) {
		if catalog != nil {
			h.catalog = catalog
		}
	}
}

func WithSubscriptionCommerce(
	subscriptions *SubscriptionStore,
	checkout SubscriptionCheckoutCreator,
	cancellation payment.SubscriptionCancellationGateway,
	publicBaseURL string,
) HandlerOption {
	return func(h *Handler) {
		if subscriptions != nil && checkout != nil && cancellation != nil {
			h.subscriptions = subscriptions
			h.subscriptionCheckout = checkout
			h.subscriptionCancellation = cancellation
			h.publicBaseURL = strings.TrimRight(publicBaseURL, "/")
		}
	}
}

func WithSubscriptionReconciliation(reconciler SubscriptionReconciler, apply SubscriptionEventApplier) HandlerOption {
	return func(h *Handler) {
		if reconciler != nil && apply != nil {
			h.subscriptionReconciler = reconciler
			h.applySubscriptionEvent = apply
		}
	}
}

func NewHandler(orders *OrderStore, entitlements *entitlement.Store, service *Service, runtime *payment.Runtime, requireUser RequireUserFunc, options ...HandlerOption) *Handler {
	h := &Handler{
		orders:       orders,
		entitlements: entitlements,
		service:      service,
		runtime:      runtime,
		requireUser:  requireUser,
		currency:     HeaderCurrencyResolver{},
	}
	for _, option := range options {
		option(h)
	}
	return h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, limit LimitFunc) {
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	// Entitlement
	mux.HandleFunc("GET /api/me/entitlements", h.requireUser(h.HandleEntitlements))
	// Commerce
	mux.HandleFunc("GET /api/chat/products", h.requireUser(h.HandleProducts))
	mux.HandleFunc("POST /api/chat/products/{id}/checkout", h.requireUser(limit(h.HandleCheckout)))
	mux.HandleFunc("GET /api/chat/orders", h.requireUser(h.HandleOrders))
	if h.subscriptions != nil {
		mux.HandleFunc("GET /api/me/subscriptions", h.requireUser(h.HandleSubscriptions))
		mux.HandleFunc("GET /api/me/subscription-payments", h.requireUser(h.HandleSubscriptionPayments))
		mux.HandleFunc("POST /api/me/subscriptions/{id}/cancellation", h.requireUser(h.HandleSetCancellation))
		mux.HandleFunc("DELETE /api/me/subscriptions/{id}/cancellation", h.requireUser(h.HandleDeleteCancellation))
	}
	// Chat
}

// ── Entitlement ──

func (h *Handler) HandleEntitlements(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	items, err := h.entitlements.ListActiveByUser(r.Context(), user.ID)
	if err != nil {
		httputil.InternalError(w)
		return
	}
	if items == nil {
		items = []Entitlement{}
	}
	allowed := make(map[string]bool)
	out := make([]string, 0)
	for _, item := range items {
		for _, skill := range item.Skills {
			value := string(skill)
			if !allowed[value] {
				allowed[value] = true
				out = append(out, value)
			}
		}
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{
		"entitlements":   items,
		"allowed_skills": out,
	})
}

// ── Commerce ──

func (h *Handler) HandleProducts(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	allowed, err := h.entitlements.AllowedSkills(r.Context(), user.ID)
	if err != nil {
		httputil.InternalError(w)
		return
	}
	allowedSet := map[string]bool{}
	for _, skill := range allowed {
		allowedSet[string(skill)] = true
	}
	items := make([]map[string]any, 0, 1)
	catalog := h.catalog
	if catalog == nil {
		catalog = NewCommerceCatalog(CommerceCatalogConfig{})
	}
	for _, commercial := range catalog.List() {
		definition, err := ProductDefinitionFor(commercial.ID)
		if err != nil {
			httputil.InternalError(w)
			return
		}
		skills := make([]string, 0, len(definition.Skills))
		for _, skill := range definition.Skills {
			skills = append(skills, string(skill))
		}
		displayCurrency := h.currency.CurrencyFor(r)
		displayPrice, priceErr := catalog.Price(commercial.ID, BillingModeOneTime, displayCurrency)
		if priceErr != nil {
			httputil.InternalError(w)
			return
		}
		options := make([]map[string]any, 0, len(commercial.Offers))
		for _, purchaseOption := range commercial.Offers {
			if purchaseOption.BillingMode == product.BillingModeSubscription && h.subscriptionCheckout == nil {
				continue
			}
			options = append(options, map[string]any{
				"billing_mode": string(purchaseOption.BillingMode),
				"price": map[string]any{
					"currency_code": string(purchaseOption.Price.Currency),
					"minor_units":   purchaseOption.Price.MinorUnits,
				},
			})
		}
		purchased := len(skills) > 0
		for _, skill := range skills {
			if !allowedSet[skill] {
				purchased = false
				break
			}
		}
		items = append(items, map[string]any{
			"id":                   string(commercial.ID),
			"name":                 commercial.Name,
			"description":          commercial.Description,
			"skills":               skills,
			"display_currency":     string(displayCurrency),
			"display_amount_minor": displayPrice.MinorUnits,
			"purchase_options":     options,
			"purchased":            purchased,
		})
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"products": items})
}

type checkoutRequest struct {
	BillingMode BillingMode `json:"billing_mode"`
	Locale      string      `json:"locale"`
}

func (h *Handler) HandleCheckout(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	var req checkoutRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.BillingMode == "" {
		req.BillingMode = BillingModeOneTime
	}
	catalog := h.catalog
	if catalog == nil {
		catalog = NewCommerceCatalog(CommerceCatalogConfig{})
	}
	productID := Product(r.PathValue("id"))
	if _, err := catalog.OfferingFor(productID); err != nil {
		httputil.RespondError(w, http.StatusNotFound, "product_not_found", "产品不存在")
		return
	}
	currency := h.currency.CurrencyFor(r)
	if req.BillingMode == product.BillingModeSubscription {
		h.handleSubscriptionCheckout(w, r, user, productID, currency, req.Locale)
		return
	}
	if req.BillingMode != BillingModeOneTime {
		httputil.RespondError(w, http.StatusBadRequest, "invalid_billing_mode", "unsupported billing mode")
		return
	}
	order, err := h.orders.CreateOrder(r.Context(), user.ID, user.Email, productID, string(currency))
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "order_failed", "创建订单失败")
		return
	}
	intent, err := h.runtime.CreateIntent(r.Context(), payment.CreateIntentInput{
		Product: string(productID), BusinessOrderID: order.ID, UserID: user.ID,
		Locale: localeOrDefault(req.Locale),
	})
	if err == nil {
		err = h.orders.AttachIntent(r.Context(), user.ID, order.ID, intent.ID)
	}
	if err == nil {
		var result *payment.CheckoutResult
		result, _, err = h.runtime.StartCheckout(r.Context(), intent.ID)
		if err == nil {
			httputil.RespondJSON(w, http.StatusOK, map[string]any{
				"order_id":                      order.ID,
				payment.MetadataPaymentIntentID: intent.ID,
				"provider":                      intent.Provider,
				"status":                        intent.Status,
				"checkout_url":                  result.CheckoutURL,
				"status_token":                  result.StatusToken,
			})
			return
		}
	}
	if cancelErr := h.orders.CancelOrder(r.Context(), user.ID, order.ID); cancelErr != nil {
		slog.Error("liki: cancel failed checkout order", "order_id", order.ID, "err", cancelErr)
	}
	slog.Error("liki: start checkout", "order_id", order.ID, "product_id", string(productID), "err", err)
	httputil.RespondError(w, http.StatusBadGateway, "checkout_failed", "发起支付失败")
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
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (h *Handler) handleSubscriptionCheckout(
	w http.ResponseWriter,
	r *http.Request,
	user auth.User,
	productID Product,
	currency Currency,
	locale string,
) {
	catalog := h.catalog
	if catalog == nil {
		catalog = NewCommerceCatalog(CommerceCatalogConfig{})
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
		slog.Error("liki: check blocking subscription", "user_id", user.ID, "err", err)
		httputil.InternalError(w)
		return
	}
	subscription, err := h.subscriptions.CreatePending(r.Context(), user.ID, user.Email, productID, currency)
	if err != nil {
		slog.Error("liki: create subscription", "user_id", user.ID, "err", err)
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
		slog.Error("liki: start subscription checkout", "subscription_id", subscription.ID, "locale", localeOrDefault(locale), "err", err)
		httputil.RespondError(w, http.StatusBadGateway, "checkout_failed", "发起订阅支付失败")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{
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
			slog.Error("liki: reconcile pending subscriptions", "user_id", user.ID, "err", err)
		}
		items, err = h.subscriptions.ListByUser(r.Context(), user.ID, 50)
		if err != nil {
			httputil.InternalError(w)
			return
		}
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"subscriptions": items})
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
			slog.Error("liki: fetch pending subscription", "subscription_id", item.ID, "checkout_id", item.ProviderCheckoutSessionID, "err", err)
			continue
		}
		if event == nil {
			continue
		}
		if err := h.applySubscriptionEvent(ctx, *event); err != nil {
			slog.Error("liki: apply reconciled subscription", "subscription_id", item.ID, "err", err)
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
		slog.Error("liki: list subscription payments", "user_id", user.ID, "err", err)
		httputil.InternalError(w)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"payments": items})
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
		slog.Error("liki: set subscription cancellation", "user_id", user.ID, "subscription_id", r.PathValue("id"), "cancel", cancel, "err", err)
		httputil.RespondError(w, http.StatusBadGateway, "subscription_update_failed", "更新订阅状态失败")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"cancel_at_period_end": cancel})
}

func localeOrDefault(locale string) string {
	normalized := strings.ToLower(strings.TrimSpace(locale))
	switch {
	case strings.HasPrefix(normalized, "en"):
		return "en"
	case strings.HasPrefix(normalized, "zh-cn"), strings.HasPrefix(normalized, "zh-sg"), strings.HasPrefix(normalized, "zh-hans"):
		return "zh-Hans"
	default:
		return "zh-Hant"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// ── Chat ──

// ── Shared HTTP helpers ──

func userID(r *http.Request) string {
	user, _ := auth.UserFromContext(r.Context())
	return user.ID
}

// writeSessionError maps liki session domain errors onto the canonical
// HTTP envelope without leaking internal identifiers.
func writeSessionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSkillForbidden):
		httputil.Forbidden(w)
	case errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrRunnerUnavailable):
		httputil.NotFound(w)
	default:
		slog.Error("liki: access session", "err", err)
		httputil.InternalError(w)
	}
}
