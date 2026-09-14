package commerce

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"liki-web/internal/auth"
	"liki-web/internal/entitlement"
	"liki-web/internal/payment"
)

type RequireUserFunc func(http.HandlerFunc) http.HandlerFunc

type Handler struct {
	orders       *Store
	entitlements *entitlement.Store
	runtime      *payment.Runtime
	requireUser  RequireUserFunc
}

func NewHandler(orders *Store, entitlements *entitlement.Store, runtime *payment.Runtime, requireUser RequireUserFunc) *Handler {
	return &Handler{orders: orders, entitlements: entitlements, runtime: runtime, requireUser: requireUser}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc) {
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/chat/products", h.requireUser(h.HandleProducts))
	mux.HandleFunc("POST /api/chat/products/{id}/checkout", h.requireUser(limit(h.HandleCheckout)))
	mux.HandleFunc("GET /api/chat/orders", h.requireUser(h.HandleOrders))
}

type checkoutRequest struct {
	Locale string `json:"locale"`
}

func (h *Handler) HandleProducts(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
		return
	}
	allowed, err := h.entitlements.AllowedSkills(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	allowedSet := map[string]bool{}
	for _, skill := range allowed {
		allowedSet[string(skill)] = true
	}
	items := make([]map[string]any, 0, 1)
	for _, commercial := range Catalog() {
		definition, err := entitlement.ProductDefinitionFor(commercial.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
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

func (h *Handler) HandleCheckout(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
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
		slog.Error("commerce: cancel failed checkout order", "order_id", order.ID, "err", cancelErr)
	}
	slog.Error("commerce: start checkout", "order_id", order.ID, "product_id", string(productID), "err", err)
	writeError(w, http.StatusBadGateway, "checkout_failed", "发起支付失败")
}

func (h *Handler) HandleOrders(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请先登录")
		return
	}
	orders, err := h.orders.ListOrdersForUser(r.Context(), user.ID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
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

func responseOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
