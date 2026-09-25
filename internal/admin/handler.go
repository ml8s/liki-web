package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/naming"
	"github.com/ml8s/liki-web/internal/product"
)

type AdminService struct {
	svc    *naming.AdminService
	onPaid func(ctx context.Context, orderID string)
}

func NewHandler(svc *naming.AdminService, onPaid func(ctx context.Context, orderID string)) *AdminService {
	return &AdminService{svc: svc, onPaid: onPaid}
}

// ── HandleListOrders ──

func (h *AdminService) HandleSession(user auth.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"email": user.Email})
	}
}

func (h *AdminService) HandleListOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("email")
		if email == "" {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "email 参数不能为空")
			return
		}

		orders, err := h.svc.ListOrders(r.Context(), email)
		if err != nil {
			slog.Error("admin: list orders", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "查询订单失败")
			return
		}

		httputil.RespondJSON(w, http.StatusOK, map[string]any{"orders": orders})
	}
}

// ── HandleCreateOrder ──

type createOrderRequest struct {
	Email     string `json:"email"`
	Product   string `json:"product"`
	Currency  string `json:"currency"`
	ExpiresAt string `json:"expires_at"`
	Lang      string `json:"lang,omitempty"`
}

func (h *AdminService) HandleCreateOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}
		if req.Email == "" || req.ExpiresAt == "" {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "email 和 expires_at 不能为空")
			return
		}

		productID := product.NormalizeProduct(product.Product(req.Product))
		if productID == "" {
			productID = product.ProductNaming
		}
		if !product.IsValidProduct(productID) {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "invalid product")
			return
		}
		currency := product.Currency(strings.ToUpper(req.Currency))
		if currency == "" {
			currency = product.CNY
		}
		amount, err := product.AmountMinor(productID, currency)
		if err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "unsupported currency")
			return
		}

		locale := req.Lang
		if locale == "" {
			locale = "zh-Hans"
		}

		orderID, orderErr := h.svc.CreatePaidOrder(r.Context(), req.Email, string(productID), amount, string(currency), req.ExpiresAt, locale)
		if orderErr != nil {
			slog.Error("admin: create order", "err", orderErr)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "创建订单失败")
			return
		}

		if h.onPaid != nil {
			go h.onPaid(context.WithoutCancel(r.Context()), orderID)
		}

		httputil.RespondJSON(w, http.StatusOK, map[string]string{"order_id": orderID})
	}
}

// ── HandleUpdateExpire ──

type updateExpireRequest struct {
	ExpiresAt string `json:"expires_at"`
}

func (h *AdminService) HandleUpdateExpire() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID := r.PathValue("id")
		if orderID == "" {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "order id 不能为空")
			return
		}

		var req updateExpireRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}
		if req.ExpiresAt == "" {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "expires_at 不能为空")
			return
		}

		if err := h.svc.UpdateExpire(r.Context(), orderID, req.ExpiresAt); err != nil {
			slog.Error("admin: update expire", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "更新失败")
			return
		}

		httputil.RespondJSON(w, http.StatusOK, map[string]string{"result": "ok"})
	}
}

// ── HandleGenerateCodes ──

type generateCodesRequest struct {
	Count int `json:"count"`
}

func (h *AdminService) HandleGenerateCodes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req generateCodesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}
		if req.Count <= 0 {
			req.Count = 1
		}
		if req.Count > 100 {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "count 不能超过 100")
			return
		}

		createdBy := "admin"
		codes, err := h.svc.BatchGenerateVerificationCodes(r.Context(), req.Count, createdBy)
		if err != nil {
			slog.Error("admin: generate codes", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "生成激活码失败")
			return
		}

		type codeItem struct {
			Code string `json:"code"`
		}
		items := make([]codeItem, len(codes))
		for i, c := range codes {
			items[i] = codeItem{Code: c}
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]any{"codes": items})
	}
}

// ── HandleListCodes ──

func (h *AdminService) HandleListCodes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		codes, err := h.svc.ListVerificationCodes(r.Context())
		if err != nil {
			slog.Error("admin: list codes", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "查询激活码失败")
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]any{"codes": codes})
	}
}
