package referral

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"liki-web/internal/httputil"
)

type AdminHandler struct {
	store *Store
}

func NewAdminHandler(store *Store) *AdminHandler {
	return &AdminHandler{store: store}
}

func (h *AdminHandler) HandleListPartners() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := h.store.ListPartners(
			r.Context(),
			PartnerRole(r.URL.Query().Get("role")),
			PartnerStatus(r.URL.Query().Get("status")),
			strings.TrimSpace(r.URL.Query().Get("q")),
			limit,
		)
		if err != nil {
			slog.Error("admin referral: list partners", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "查询合作伙伴失败")
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]any{"partners": items})
	}
}

func (h *AdminHandler) HandleListApplications() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := h.store.ListApplications(
			r.Context(),
			ApplicationStatus(r.URL.Query().Get("status")),
			limit,
		)
		if err != nil {
			slog.Error("admin referral: list applications", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "查询申请失败")
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]any{"applications": items})
	}
}

type applicationDecisionRequest struct {
	Approve bool   `json:"approve"`
	Reason  string `json:"reason"`
}

func (r applicationDecisionRequest) Validate() error {
	if strings.TrimSpace(r.Reason) == "" {
		return errors.New("reason is required")
	}
	return nil
}

func (h *AdminHandler) HandleApplicationDecision() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "application id 不能为空")
			return
		}
		req, ok := httputil.DecodeAndValidate[applicationDecisionRequest](w, r)
		if !ok {
			return
		}
		if err := h.store.DecideApplication(r.Context(), id, "admin", req.Approve, req.Reason); err != nil {
			switch {
			case errors.Is(err, ErrApplicationNotFound):
				httputil.RespondError(w, http.StatusNotFound, "application_not_found", "申请不存在")
			case errors.Is(err, ErrApplicationNotPending):
				httputil.RespondError(w, http.StatusConflict, "application_not_pending", "申请已被处理")
			case errors.Is(err, ErrNotEligible):
				httputil.RespondError(w, http.StatusConflict, "partner_inactive", "推广者已冻结，不能通过")
			default:
				slog.Error("admin referral: decide application", "id", id, "err", err)
				httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "处理申请失败")
			}
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"result": "ok"})
	}
}

type partnerStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func (r partnerStatusRequest) Validate() error {
	if r.Status != string(PartnerActive) && r.Status != string(PartnerFrozen) {
		return errors.New("invalid status")
	}
	if strings.TrimSpace(r.Reason) == "" {
		return errors.New("reason is required")
	}
	return nil
}

func (h *AdminHandler) HandlePartnerStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("userId")
		if userID == "" {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "user id 不能为空")
			return
		}
		req, ok := httputil.DecodeAndValidate[partnerStatusRequest](w, r)
		if !ok {
			return
		}
		if err := h.store.SetPartnerStatus(r.Context(), userID, PartnerStatus(req.Status), "admin", req.Reason); err != nil {
			switch {
			case errors.Is(err, ErrPartnerNotFound):
				httputil.RespondError(w, http.StatusNotFound, "partner_not_found", "合作伙伴不存在")
			case errors.Is(err, ErrInvalidPartnerStatus):
				httputil.RespondError(w, http.StatusUnprocessableEntity, "validation_error", "无效状态")
			default:
				slog.Error("admin referral: set partner status", "user_id", userID, "err", err)
				httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "更新合作伙伴失败")
			}
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"result": "ok"})
	}
}

func (h *AdminHandler) HandleAdminActions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := h.store.ListPartnerAdminActions(r.Context(), limit)
		if err != nil {
			slog.Error("admin referral: list actions", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "查询管理记录失败")
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]any{"actions": items})
	}
}
