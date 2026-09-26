package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/naming"
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

func (h *AdminService) HandleCreateOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httputil.RespondError(w, http.StatusGone, "product_retired", "独立起名订单已下线，请使用 Liki Chat")
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

func (h *AdminService) HandleGenerateCodes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httputil.RespondError(w, http.StatusGone, "product_retired", "独立起名激活码已下线，请使用 Liki Chat")
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
