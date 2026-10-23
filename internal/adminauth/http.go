package adminauth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler { return &Handler{store: store} }

func actorFromContext(r *http.Request) string {
	user, _ := auth.UserFromContext(r.Context())
	return normalizeEmail(user.Email)
}

type createAdminRequest struct {
	Email string `json:"email"`
}

func (r createAdminRequest) Validate() error {
	return validateEmail(r.Email)
}

type capabilityRequest struct {
	Capability string `json:"capability"`
	Reason     string `json:"reason"`
}

func (r capabilityRequest) Validate() error {
	if !IsCapabilityValid(Capability(strings.TrimSpace(r.Capability))) {
		return ErrInvalidCapability
	}
	return validateReason(r.Reason)
}

type adminStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func (r adminStatusRequest) Validate() error {
	status := AdminStatus(strings.TrimSpace(r.Status))
	if status != AdminActive && status != AdminDisabled {
		return ErrInvalidAdminStatus
	}
	return validateReason(r.Reason)
}

func validateEmail(value string) error {
	value = normalizeEmail(value)
	if value == "" || len(value) > 254 || !strings.Contains(value, "@") {
		return errors.New("adminauth: invalid email")
	}
	return nil
}

func validateReason(value string) error {
	if strings.TrimSpace(value) == "" {
		return ErrReasonRequired
	}
	return nil
}

func (h *Handler) HandleListAdmins() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := h.store.ListAdmins(r.Context())
		if err != nil {
			slog.ErrorContext(r.Context(), "adminauth: list admins", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "查询管理员失败")
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]any{"admins": items})
	}
}

func (h *Handler) HandleCreateAdmin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := httputil.DecodeStrict[createAdminRequest](w, r)
		if !ok {
			return
		}
		err := h.store.EnsureAdmin(r.Context(), actorFromContext(r), req.Email, "create administrator")
		if err != nil {
			h.writeError(r.Context(), w, err, "创建管理员失败")
			return
		}
		httputil.RespondJSON(w, http.StatusCreated, map[string]string{"result": "ok"})
	}
}

func (h *Handler) HandleGrantCapability() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := httputil.DecodeStrict[capabilityRequest](w, r)
		if !ok {
			return
		}
		err := h.store.GrantCapability(r.Context(), actorFromContext(r), r.PathValue("email"),
			Capability(req.Capability), req.Reason)
		if err != nil {
			h.writeError(r.Context(), w, err, "授权失败")
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"result": "ok"})
	}
}

func (h *Handler) HandleRevokeCapability() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := httputil.DecodeStrict[capabilityRequest](w, r)
		if !ok {
			return
		}
		err := h.store.RevokeCapability(r.Context(), actorFromContext(r), r.PathValue("email"),
			Capability(req.Capability), req.Reason)
		if err != nil {
			h.writeError(r.Context(), w, err, "撤销授权失败")
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"result": "ok"})
	}
}

func (h *Handler) HandleEnableAdmin() http.HandlerFunc {
	return h.handleStatus(AdminActive)
}

func (h *Handler) HandleDisableAdmin() http.HandlerFunc {
	return h.handleStatus(AdminDisabled)
}

func (h *Handler) handleStatus(status AdminStatus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := httputil.DecodeStrict[adminStatusRequest](w, r)
		if !ok {
			return
		}
		err := h.store.SetAdminStatus(r.Context(), actorFromContext(r), r.PathValue("email"), status, req.Reason)
		if err != nil {
			h.writeError(r.Context(), w, err, "更新管理员状态失败")
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"result": "ok"})
	}
}

func (h *Handler) writeError(ctx context.Context, w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, ErrAdminNotFound):
		httputil.RespondError(w, http.StatusNotFound, "admin_not_found", "管理员不存在")
	case errors.Is(err, ErrAdminAlreadyExists):
		httputil.RespondError(w, http.StatusConflict, "admin_already_exists", "管理员已存在")
	case errors.Is(err, ErrInvalidCapability):
		httputil.RespondError(w, http.StatusUnprocessableEntity, "invalid_capability", "无效能力")
	case errors.Is(err, ErrInvalidAdminStatus):
		httputil.RespondError(w, http.StatusUnprocessableEntity, "validation_error", "无效状态")
	case errors.Is(err, ErrReasonRequired):
		httputil.RespondError(w, http.StatusUnprocessableEntity, "validation_error", "必须填写原因")
	case errors.Is(err, ErrSelfManagementForbidden):
		httputil.RespondError(w, http.StatusConflict, "self_management_forbidden", "不能修改自己的管理员权限")
	case errors.Is(err, ErrLastSuperAdmin):
		httputil.RespondError(w, http.StatusConflict, "last_super_admin", "至少保留一位可用超级管理员")
	case errors.Is(err, ErrSuperRequiresSuper):
		httputil.RespondError(w, http.StatusForbidden, "super_requires_super", "变更超级管理员权限需要超级管理员身份")
	default:
		slog.ErrorContext(ctx, "adminauth: management operation", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", fallback)
	}
}
