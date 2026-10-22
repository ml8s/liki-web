package sales

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

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
	mux.HandleFunc("GET /api/sales/orders/{id}", h.requireUser(h.HandleOrderDetail))
	mux.HandleFunc("POST /api/sales/orders/{id}/cancel", h.requireUser(h.limit(h.HandleCancelOrder)))
	mux.HandleFunc("POST /api/sales/orders/{id}/checkout", h.requireUser(h.limit(h.HandleResumeCheckout)))
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
			httputil.RespondJSON(w, http.StatusCreated, checkoutView{
				OrderID:         order.ID,
				PaymentIntentID: intent.ID,
				Provider:        string(intent.Provider),
				Status:          string(intent.Status),
				CheckoutURL:     result.CheckoutURL,
				QRCodeURL:       result.QRCodeURL,
				StatusToken:     result.StatusToken,
			})
			return
		}
	}
	if cancelErr := h.orders.CancelOrder(r.Context(), user.ID, order.ID); cancelErr != nil {
		slog.ErrorContext(r.Context(), "sales: cancel failed checkout order", "order_id", order.ID, "err", cancelErr)
	}
	slog.ErrorContext(r.Context(), "sales: start checkout", "order_id", order.ID, "err", err)
	httputil.RespondError(w, http.StatusBadGateway, "checkout_failed", "发起支付失败")
}

// checkoutView is the browser-safe checkout resource. Provider session and
// order internals stay behind the BFF.
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
