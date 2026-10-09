package liki

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/profile"
)

type RequireUserFunc func(http.HandlerFunc) http.HandlerFunc
type LimitFunc func(http.HandlerFunc) http.HandlerFunc

// Handler registers all Liki Chat HTTP routes: entitlement listing,
// product catalog, checkout, orders, and chat sessions/messages.
type Handler struct {
	orders       *OrderStore
	entitlements *EntitlementStore
	service      *Service
	runtime      *payment.Runtime
	requireUser  RequireUserFunc
	currency     CurrencyResolver
}

type HandlerOption func(*Handler)

func WithCurrencyResolver(resolver CurrencyResolver) HandlerOption {
	return func(h *Handler) {
		if resolver != nil {
			h.currency = resolver
		}
	}
}

func NewHandler(orders *OrderStore, entitlements *EntitlementStore, service *Service, runtime *payment.Runtime, requireUser RequireUserFunc, options ...HandlerOption) *Handler {
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
	// Chat
	mux.HandleFunc("GET /api/chat/sessions", h.requireUser(h.HandleSessions))
	mux.HandleFunc("POST /api/chat/sessions", h.requireUser(limit(h.HandleCreateSession)))
	mux.HandleFunc("GET /api/chat/sessions/{id}", h.requireUser(h.HandleSession))
	mux.HandleFunc("GET /api/chat/sessions/{id}/messages", h.requireUser(h.HandleMessages))
	mux.HandleFunc("POST /api/chat/sessions/{id}/ag-ui", h.requireUser(limit(h.HandleAGUIMessage)))
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
	for _, commercial := range Catalog() {
		definition, err := ProductDefinitionFor(commercial.ID)
		if err != nil {
			httputil.InternalError(w)
			return
		}
		skills := make([]string, 0, len(definition.Skills))
		for _, skill := range definition.Skills {
			skills = append(skills, string(skill))
		}
		prices := make(map[string]int64, len(commercial.AmountMinor))
		for currency, amount := range commercial.AmountMinor {
			prices[currency] = amount
		}
		displayCurrency := h.currency.CurrencyFor(r)
		displayAmount := commercial.AmountMinor[string(displayCurrency)]
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
			"prices":               prices,
			"display_currency":     string(displayCurrency),
			"display_amount_minor": displayAmount,
			"purchased":            purchased,
		})
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"products": items})
}

type checkoutRequest struct {
	Locale string `json:"locale"`
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
	productID := Product(r.PathValue("id"))
	if _, err := DefinitionFor(productID); err != nil {
		httputil.RespondError(w, http.StatusNotFound, "product_not_found", "产品不存在")
		return
	}
	currency := string(h.currency.CurrencyFor(r))
	order, err := h.orders.CreateOrder(r.Context(), user.ID, user.Email, productID, currency)
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
				"order_id":          order.ID,
				"payment_intent_id": intent.ID,
				"provider":          intent.Provider,
				"status":            intent.Status,
				"checkout_url":      result.CheckoutURL,
				"status_token":      result.StatusToken,
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

type createSessionRequest struct {
	Skill     string `json:"skill"`
	Title     string `json:"title"`
	ProfileID string `json:"profile_id"`
}

func (r createSessionRequest) Validate() error {
	if strings.TrimSpace(r.Skill) == "" {
		return errors.New("skill is required")
	}
	return nil
}

func (h *Handler) HandleSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.service.store.ListSessions(r.Context(), userID(r))
	if err != nil {
		httputil.InternalError(w)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, sessions)
}

func (h *Handler) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.Decode[createSessionRequest](w, r)
	if !ok {
		return
	}
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	session, err := h.service.CreateSession(r.Context(), user.ID, req.Skill, req.Title, req.ProfileID)
	if err != nil {
		if errors.Is(err, ErrSkillForbidden) {
			httputil.RespondError(w, http.StatusForbidden, "skill_forbidden", "当前权益不可使用该 skill")
			return
		}
		if errors.Is(err, profile.ErrNotFound) {
			httputil.NotFound(w)
			return
		}
		slog.Error("liki: create session", "user_id", user.ID, "err", err)
		httputil.InternalError(w)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, session)
}

func (h *Handler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	sessionID := r.PathValue("id")
	if _, err := h.service.Session(r.Context(), user.ID, sessionID); err != nil {
		writeSessionError(w, err)
		return
	}
	messages, err := h.service.store.ListMessages(r.Context(), user.ID, sessionID)
	if err != nil {
		slog.Error("liki: list messages", "session_id", sessionID, "err", err)
		httputil.InternalError(w)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, messages)
}

func (h *Handler) HandleSession(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	session, err := h.service.Session(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		writeSessionError(w, err)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, session)
}

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
