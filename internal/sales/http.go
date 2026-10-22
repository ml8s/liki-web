package sales

import (
	"context"
	"net/http"

	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
	"strings"
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
