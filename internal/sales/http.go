package sales

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
)

type RequireUserFunc func(http.HandlerFunc) http.HandlerFunc
type LimitFunc func(http.HandlerFunc) http.HandlerFunc

// SubscriptionCheckoutCreator is owned by the provider gateway. Keeping it as
// a local interface avoids coupling sales HTTP to Dodo.
type SubscriptionCheckoutCreator interface {
	CreateSubscriptionCheckout(context.Context, payment.SubscriptionCheckoutRequest) (*payment.CheckoutResult, error)
}

// SubscriptionReconciler heals a lost webhook through the provider's
// authoritative checkout/payment/subscription lookup chain.
type SubscriptionReconciler interface {
	FetchSubscriptionByCheckout(context.Context, string) (*payment.WebhookEvent, error)
}

type SubscriptionEventApplier func(context.Context, payment.WebhookEvent) error

// SkillAuthorizer reports which skills a buyer may currently use.
type SkillAuthorizer interface {
	AllowedSkills(ctx context.Context, userID string) ([]product.Skill, error)
}

// Handler exposes sales routes.
type Handler struct {
	orders                   *OrderStore
	catalog                  *product.Catalog
	runtime                  *payment.Runtime
	requireUser              RequireUserFunc
	limit                    LimitFunc
	currency                 CurrencyResolver
	skills                   SkillAuthorizer
	subscriptions            *SubscriptionStore
	subscriptionCheckout     SubscriptionCheckoutCreator
	subscriptionCancellation payment.SubscriptionCancellationGateway
	subscriptionReconciler   SubscriptionReconciler
	applySubscriptionEvent   SubscriptionEventApplier
	publicBaseURL            string
}

type HandlerOption func(*Handler)

func WithCatalog(catalog *product.Catalog) HandlerOption {
	return func(h *Handler) {
		if catalog != nil {
			h.catalog = catalog
		}
	}
}

func WithCurrencyResolver(resolver CurrencyResolver) HandlerOption {
	return func(h *Handler) {
		if resolver != nil {
			h.currency = resolver
		}
	}
}

func WithSkillAuthorizer(authorizer SkillAuthorizer) HandlerOption {
	return func(h *Handler) {
		if authorizer != nil {
			h.skills = authorizer
		}
	}
}

// WithSubscriptions wires the subscription surface. The checkout creator is
// optional: a deployment without a configured recurring product must still list
// and cancel existing subscriptions, but it must not advertise a new one —
// HandleProducts only offers a subscription while subscriptionCheckout is set.
func WithSubscriptions(
	subscriptions *SubscriptionStore,
	checkout SubscriptionCheckoutCreator,
	cancellation payment.SubscriptionCancellationGateway,
	publicBaseURL string,
) HandlerOption {
	return func(h *Handler) {
		if subscriptions == nil || cancellation == nil {
			return
		}
		h.subscriptions = subscriptions
		h.subscriptionCheckout = checkout
		h.subscriptionCancellation = cancellation
		h.publicBaseURL = strings.TrimRight(publicBaseURL, "/")
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

func NewHandler(
	orders *OrderStore,
	runtime *payment.Runtime,
	requireUser RequireUserFunc,
	limit LimitFunc,
	options ...HandlerOption,
) *Handler {
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	h := &Handler{
		orders:      orders,
		runtime:     runtime,
		requireUser: requireUser,
		limit:       limit,
		currency:    HeaderCurrencyResolver{},
	}
	for _, option := range options {
		option(h)
	}
	return h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/sales/products", h.requireUser(h.HandleProducts))
	mux.HandleFunc("POST /api/sales/checkouts", h.requireUser(h.limit(h.HandleCheckout)))
	mux.HandleFunc("GET /api/sales/orders", h.requireUser(h.HandleOrders))
	if h.subscriptions == nil {
		return
	}
	mux.HandleFunc("GET /api/sales/subscriptions", h.requireUser(h.HandleSubscriptions))
	mux.HandleFunc("GET /api/sales/subscription-payments", h.requireUser(h.HandleSubscriptionPayments))
	mux.HandleFunc("POST /api/sales/subscriptions/{id}/cancellation", h.requireUser(h.HandleSetCancellation))
	mux.HandleFunc("DELETE /api/sales/subscriptions/{id}/cancellation", h.requireUser(h.HandleDeleteCancellation))
}

func (h *Handler) HandleProducts(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	allowed, err := h.skills.AllowedSkills(r.Context(), user.ID)
	if err != nil {
		httputil.InternalError(w)
		return
	}
	allowedSet := map[string]bool{}
	for _, skill := range allowed {
		allowedSet[string(skill)] = true
	}
	catalog := h.catalog
	if catalog == nil {
		catalog = product.NewCatalog(product.CatalogConfig{})
	}
	items := make([]map[string]any, 0, 1)
	for _, commercial := range catalog.List() {
		definition, err := product.DefinitionFor(commercial.ID)
		if err != nil {
			httputil.InternalError(w)
			return
		}
		skills := make([]string, 0, len(definition.Skills))
		for _, skill := range definition.Skills {
			skills = append(skills, string(skill))
		}
		displayCurrency := h.currency.CurrencyFor(r)
		displayPrice, priceErr := catalog.Price(commercial.ID, product.BillingModeOneTime, displayCurrency)
		if priceErr != nil {
			httputil.InternalError(w)
			return
		}
		options := make([]map[string]any, 0, len(commercial.Offers))
		for _, offer := range commercial.Offers {
			if offer.BillingMode == product.BillingModeSubscription && h.subscriptionCheckout == nil {
				continue
			}
			// The catalog holds one offer per billing mode and currency, so
			// matching the offer against the server-selected display currency is
			// the same gate catalog.Price applies to order creation and
			// subscription checkout. Advertising another region's offer (for
			// example the USD-only subscription to a CNY buyer) would only fail
			// closed on click.
			if offer.Price.Currency != string(displayCurrency) {
				continue
			}
			options = append(options, map[string]any{
				"billing_mode": string(offer.BillingMode),
				"price": map[string]any{
					"currency_code": string(offer.Price.Currency),
					"minor_units":   offer.Price.MinorUnits,
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
	ProductID   string              `json:"product_id"`
	BillingMode product.BillingMode `json:"billing_mode"`
	Locale      string              `json:"locale"`
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
		req.BillingMode = product.BillingModeOneTime
	}
	catalog := h.catalog
	if catalog == nil {
		catalog = product.NewCatalog(product.CatalogConfig{})
	}
	productID := product.Product(req.ProductID)
	if _, err := catalog.OfferingFor(productID); err != nil {
		httputil.RespondError(w, http.StatusNotFound, "product_not_found", "产品不存在")
		return
	}
	currency := h.currency.CurrencyFor(r)
	if req.BillingMode == product.BillingModeSubscription {
		h.handleSubscriptionCheckout(w, r, user, productID, currency, req.Locale)
		return
	}
	if req.BillingMode != product.BillingModeOneTime {
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
		Locale: LocaleOrDefault(req.Locale),
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
		slog.Error("sales: cancel failed checkout order", "order_id", order.ID, "err", cancelErr)
	}
	slog.Error("sales: start checkout", "order_id", order.ID, "err", err)
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
		slog.Error("sales: check blocking subscription", "user_id", user.ID, "err", err)
		httputil.InternalError(w)
		return
	}
	subscription, err := h.subscriptions.CreatePending(r.Context(), user.ID, user.Email, productID, currency)
	if err != nil {
		slog.Error("sales: create subscription", "user_id", user.ID, "err", err)
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
		slog.Error("sales: start subscription checkout", "subscription_id", subscription.ID, "locale", LocaleOrDefault(locale), "err", err)
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
			slog.Error("sales: reconcile pending subscriptions", "user_id", user.ID, "err", err)
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
			slog.Error("sales: fetch pending subscription", "subscription_id", item.ID, "checkout_id", item.ProviderCheckoutSessionID, "err", err)
			continue
		}
		if event == nil {
			continue
		}
		if err := h.applySubscriptionEvent(ctx, *event); err != nil {
			slog.Error("sales: apply reconciled subscription", "subscription_id", item.ID, "err", err)
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
		slog.Error("sales: list subscription payments", "user_id", user.ID, "err", err)
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
		slog.Error("sales: set subscription cancellation", "user_id", user.ID, "subscription_id", r.PathValue("id"), "cancel", cancel, "err", err)
		httputil.RespondError(w, http.StatusBadGateway, "subscription_update_failed", "更新订阅状态失败")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"cancel_at_period_end": cancel})
}

func LocaleOrDefault(locale string) string {
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
