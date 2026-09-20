package liki

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"liki-web/internal/agent"
	"liki-web/internal/auth"
	"liki-web/internal/payment"
	"liki-web/internal/profile"
	"liki-web/internal/sse"
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
	limit        LimitFunc
}

func NewHandler(orders *OrderStore, entitlements *EntitlementStore, service *Service, runtime *payment.Runtime, requireUser RequireUserFunc) *Handler {
	return &Handler{orders: orders, entitlements: entitlements, service: service, runtime: runtime, requireUser: requireUser}
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
	mux.HandleFunc("POST /api/chat/sessions/{id}/messages", h.requireUser(limit(h.HandleSendMessage)))
}

// ── Entitlement ──

func (h *Handler) HandleEntitlements(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	items, err := h.entitlements.ListActiveByUser(r.Context(), user.ID)
	if err != nil {
		internalError(w)
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
	responseOK(w, map[string]any{
		"entitlements":   items,
		"allowed_skills": out,
	})
}

// ── Commerce ──

func (h *Handler) HandleProducts(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	allowed, err := h.entitlements.AllowedSkills(r.Context(), user.ID)
	if err != nil {
		internalError(w)
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
			internalError(w)
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
		purchased := len(skills) > 0
		for _, skill := range skills {
			if !allowedSet[skill] {
				purchased = false
				break
			}
		}
		items = append(items, map[string]any{
			"id":          string(commercial.ID),
			"name":        commercial.Name,
			"description": commercial.Description,
			"skills":      skills,
			"prices":      prices,
			"purchased":   purchased,
		})
	}
	responseOK(w, map[string]any{"products": items})
}

type checkoutRequest struct {
	Locale string `json:"locale"`
}

func (h *Handler) HandleCheckout(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	var req checkoutRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	productID := Product(r.PathValue("id"))
	if _, err := DefinitionFor(productID); err != nil {
		writeError(w, http.StatusNotFound, "product_not_found", "产品不存在")
		return
	}
	currency := DetectCurrency(r.Header.Get("CF-IPCountry"), r.Header.Get("Geo-Country"))
	order, err := h.orders.CreateOrder(r.Context(), user.ID, user.Email, productID, currency)
	if err != nil {
		writeError(w, http.StatusBadRequest, "order_failed", "创建订单失败")
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
			responseOK(w, map[string]any{
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
	writeError(w, http.StatusBadGateway, "checkout_failed", "发起支付失败")
}

func (h *Handler) HandleOrders(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	orders, err := h.orders.ListOrdersForUser(r.Context(), user.ID, 50)
	if err != nil {
		internalError(w)
		return
	}
	responseOK(w, map[string]any{"orders": orders})
}

func localeOrDefault(locale string) string {
	if strings.HasPrefix(strings.TrimSpace(locale), "en") {
		return "en"
	}
	return "zh-Hant"
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

type sendMessageRequest struct {
	Content string `json:"content"`
}

func (r sendMessageRequest) Validate() error {
	if strings.TrimSpace(r.Content) == "" {
		return errors.New("content is required")
	}
	return nil
}

func (h *Handler) HandleSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.service.store.ListSessions(r.Context(), userID(r))
	if err != nil {
		internalError(w)
		return
	}
	responseOK(w, sessions)
}

func (h *Handler) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[createSessionRequest](w, r)
	if !ok {
		return
	}
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	session, err := h.service.CreateSession(r.Context(), user.ID, req.Skill, req.Title, req.ProfileID)
	if err != nil {
		if errors.Is(err, ErrSkillForbidden) {
			writeError(w, http.StatusForbidden, "skill_forbidden", "当前权益不可使用该 skill")
			return
		}
		if errors.Is(err, profile.ErrNotFound) {
			notFound(w)
			return
		}
		slog.Error("liki: create session", "user_id", user.ID, "err", err)
		internalError(w)
		return
	}
	responseOK(w, session)
}

func (h *Handler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
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
		internalError(w)
		return
	}
	responseOK(w, messages)
}

func (h *Handler) HandleSession(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	session, err := h.service.Session(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		writeSessionError(w, err)
		return
	}
	responseOK(w, session)
}

func (h *Handler) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[sendMessageRequest](w, r)
	if !ok {
		return
	}
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	sessionID := r.PathValue("id")

	if _, err := h.service.Session(r.Context(), user.ID, sessionID); err != nil {
		writeSessionError(w, err)
		return
	}

	flusher, ok := sse.Begin(w)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器不支持流式响应")
		return
	}

	streamCtx, cancelStream := context.WithCancel(r.Context())
	defer cancelStream()
	var writeErr error
	userMessage, assistantMessage, session, err := h.service.SendMessage(
		streamCtx, user.ID, sessionID, req.Content,
		func(ev agent.ChatEvent) {
			if writeErr != nil {
				return
			}
			if err := sse.Write(w, flusher, ev); err != nil {
				writeErr = err
				cancelStream()
				slog.Warn("liki: write sse event", "err", err)
			}
		},
	)
	if err != nil && writeErr == nil && !errors.Is(err, context.Canceled) {
		slog.Error("liki: send message", "session_id", sessionID, "err", err)
	}
	if err != nil {
		if writeErr == nil && !errors.Is(err, context.Canceled) {
			_ = sse.Write(w, flusher, agent.ChatEvent{Type: agent.EventError, Content: "服务暂时不可用，请稍后重试"})
		}
		return
	}
	if writeErr != nil {
		return
	}

	_ = sse.Write(w, flusher, agent.ChatEvent{
		Type: agent.EventDone,
		Data: map[string]any{
			"session":           session,
			"active_skill":      session.Skill,
			"user_message":      userMessage,
			"assistant_message": assistantMessage,
		},
	})
}

// ── Shared HTTP helpers ──

func userID(r *http.Request) string {
	user, _ := auth.UserFromContext(r.Context())
	return user.ID
}

type validator interface {
	Validate() error
}

func responseOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func unauthorized(w http.ResponseWriter) {
	writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
}

func notFound(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, "not_found", "会话不存在")
}

func forbidden(w http.ResponseWriter) {
	writeError(w, http.StatusForbidden, "skill_forbidden", "当前权益不可使用该 skill")
}

func internalError(w http.ResponseWriter) {
	writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
}

func writeSessionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSkillForbidden):
		forbidden(w)
	case errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrRunnerUnavailable):
		notFound(w)
	default:
		slog.Error("liki: access session", "err", err)
		internalError(w)
	}
}

func decode[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return req, false
	}
	if v, ok := any(req).(validator); ok {
		if err := v.Validate(); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "Invalid request parameters")
			return req, false
		}
	}
	return req, true
}
