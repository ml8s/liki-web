package admin

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ml8s/liki-web/internal/adminauth"
	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/liki"
)

type AdminService struct {
	authorizer           *adminauth.Store
	entitlements         *liki.EntitlementStore
	users                UserByEmailReader
	commerce             *liki.CommerceStores
	referralFinance      ReferralFinanceReader
	compensationNotifier CompensationNotifier
}

func NewHandler(authorizer *adminauth.Store, entitlements *liki.EntitlementStore) *AdminService {
	return &AdminService{authorizer: authorizer, entitlements: entitlements}
}

func (h *AdminService) HandleSession(user auth.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		capabilities, err := h.authorizer.Capabilities(r.Context(), user)
		if err != nil {
			slog.Error("admin: read session capabilities", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "读取管理能力失败")
			return
		}
		values := make([]string, 0, len(capabilities))
		for _, capability := range capabilities {
			values = append(values, string(capability))
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]any{"email": user.Email, "capabilities": values})
	}
}

type entitlementExpireRequest struct {
	ExpiresAt string `json:"expires_at"`
	Reason    string `json:"reason"`
}

func (r entitlementExpireRequest) Validate() error {
	if strings.TrimSpace(r.Reason) == "" {
		return errors.New("reason is required")
	}
	_, err := parseAdminTime(r.ExpiresAt)
	return err
}

type entitlementRevokeRequest struct {
	Reason string `json:"reason"`
}

func (r entitlementRevokeRequest) Validate() error {
	return validateRequiredReason(r.Reason)
}

func (h *AdminService) HandleEntitlements() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("email")))
		status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
		items, err := h.entitlements.ListForAdmin(r.Context(), email, status, 100)
		if err != nil {
			slog.Error("admin: list entitlements", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "查询用户权益失败")
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]any{"entitlements": items})
	}
}

func (h *AdminService) HandleExpireEntitlement() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		req, ok := httputil.DecodeAndValidate[entitlementExpireRequest](w, r)
		if !ok || id == "" {
			return
		}
		expiresAt, err := parseAdminTime(req.ExpiresAt)
		if err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "过期时间格式错误")
			return
		}
		before, after, err := h.entitlements.UpdateExpiration(r.Context(), id, expiresAt)
		if writeEntitlementError(w, err) {
			return
		}
		h.auditEntitlementChange(r, "entitlement.expire", id, before, after, req.Reason)
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"result": "ok"})
	}
}

func (h *AdminService) HandleRevokeEntitlement() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		req, ok := httputil.DecodeAndValidate[entitlementRevokeRequest](w, r)
		if !ok || id == "" {
			return
		}
		before, after, err := h.entitlements.Revoke(r.Context(), id)
		if writeEntitlementError(w, err) {
			return
		}
		h.auditEntitlementChange(r, "entitlement.revoked", id, before, after, req.Reason)
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"result": "ok"})
	}
}

func writeEntitlementError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, liki.ErrEntitlementNotFound):
		httputil.RespondError(w, http.StatusNotFound, "entitlement_not_found", "用户权益不存在")
	case errors.Is(err, liki.ErrRevokedEntitlementImmutable):
		httputil.RespondError(w, http.StatusConflict, "entitlement_revoked", "已撤销的权益不能改期")
	case errors.Is(err, liki.ErrEntitlementAlreadyRevoked):
		httputil.RespondError(w, http.StatusConflict, "entitlement_already_revoked", "权益已撤销")
	default:
		slog.Error("admin: change entitlement", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "更新用户权益失败")
	}
	return true
}

func (h *AdminService) auditEntitlementChange(r *http.Request, action, id, before, after, reason string) {
	actor, _ := auth.UserFromContext(r.Context())
	err := h.authorizer.Audit(r.Context(), adminauth.AuditEntry{
		ActorUserID: actor.ID, ActorEmail: actor.Email,
		Capability: string(adminauth.CapabilityEntitlementManage),
		Action:     action, ObjectType: "entitlement", ObjectID: id,
		Before: before, After: after, Reason: strings.TrimSpace(reason),
	})
	if err != nil {
		slog.Error("admin: audit entitlement", "id", id, "err", err)
	}
}

func parseAdminTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("time is required")
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}
	if parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local); err == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, errors.New("invalid time")
}

func validateRequiredReason(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("reason is required")
	}
	return nil
}
