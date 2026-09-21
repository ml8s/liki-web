package referral

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"liki-web/internal/auth"
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

func actorFromRequest(r *http.Request) string {
	user, ok := auth.UserFromContext(r.Context())
	if ok && user.ID != "" {
		return user.ID
	}
	if ok && user.Email != "" {
		return user.Email
	}
	return "admin"
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
		if err := h.store.DecideApplication(r.Context(), id, actorFromRequest(r), req.Approve, req.Reason); err != nil {
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
	Reason string `json:"reason"`
}

func (r partnerStatusRequest) Validate() error {
	if strings.TrimSpace(r.Reason) == "" {
		return errors.New("reason is required")
	}
	return nil
}

func (h *AdminHandler) HandlePartnerFreeze() http.HandlerFunc {
	return h.handlePartnerStatus(PartnerFrozen)
}

func (h *AdminHandler) HandlePartnerUnfreeze() http.HandlerFunc {
	return h.handlePartnerStatus(PartnerActive)
}

func (h *AdminHandler) handlePartnerStatus(status PartnerStatus) http.HandlerFunc {
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
		if err := h.store.SetPartnerStatus(r.Context(), userID, status, actorFromRequest(r), req.Reason); err != nil {
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

func (h *AdminHandler) HandleListPayouts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := h.store.ListPayouts(r.Context(), PayoutStatus(r.URL.Query().Get("status")), limit)
		if err != nil {
			slog.Error("admin referral: list payouts", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "查询提现失败")
			return
		}
		for i := range items {
			items[i].Account = maskPayoutAccount(items[i].Account)
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]any{"payouts": items})
	}
}

type payoutActionRequest struct {
	Reason           string `json:"reason"`
	PaymentReference string `json:"payment_reference"`
}

func (r payoutActionRequest) Validate() error {
	if strings.TrimSpace(r.Reason) == "" && r.PaymentReference == "" {
		return errors.New("reason or payment reference is required")
	}
	return nil
}

func (h *AdminHandler) HandlePayoutAction() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		action := r.PathValue("action")
		req, ok := httputil.DecodeAndValidate[payoutActionRequest](w, r)
		if !ok {
			return
		}
		var err error
		switch action {
		case "approve":
			err = h.store.ApprovePayout(r.Context(), id, actorFromRequest(r))
		case "reject":
			err = h.store.DecidePayout(r.Context(), id, actorFromRequest(r), false, req.Reason)
		case "mark-paid":
			err = h.store.MarkPayoutPaid(r.Context(), id, actorFromRequest(r), req.PaymentReference)
		default:
			httputil.RespondError(w, http.StatusNotFound, "not_found", "未知操作")
			return
		}
		if err != nil {
			switch {
			case errors.Is(err, ErrPayoutNotFound):
				httputil.RespondError(w, http.StatusNotFound, "payout_not_found", "提现不存在")
			case errors.Is(err, ErrPayoutInvalidTransition):
				httputil.RespondError(w, http.StatusConflict, "payout_invalid_transition", "提现状态已变化，请刷新后重试")
			default:
				slog.Error("admin referral: payout action", "id", id, "action", action, "err", err)
				httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "处理提现失败")
			}
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"result": "ok"})
	}
}

type promotePartnerRequest struct {
	Reason string `json:"reason"`
}

func (r promotePartnerRequest) Validate() error {
	if strings.TrimSpace(r.Reason) == "" {
		return errors.New("reason is required")
	}
	return nil
}

func (h *AdminHandler) HandlePromotePartner() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("userId")
		req, ok := httputil.DecodeAndValidate[promotePartnerRequest](w, r)
		if !ok {
			return
		}
		if err := h.store.PromoteToScout(r.Context(), userID, actorFromRequest(r), req.Reason); err != nil {
			switch {
			case errors.Is(err, ErrPartnerNotFound):
				httputil.RespondError(w, http.StatusNotFound, "partner_not_found", "合作伙伴不存在")
			default:
				slog.Error("admin referral: promote partner", "user_id", userID, "err", err)
				httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "升级合作伙伴失败")
			}
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"result": "ok"})
	}
}

func (h *AdminHandler) HandleListFinanceTasks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := h.store.ListFinanceTasks(r.Context(), FinanceTaskStatus(r.URL.Query().Get("status")), limit)
		if err != nil {
			slog.Error("admin referral: list finance tasks", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "查询财务任务失败")
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]any{"tasks": items})
	}
}

type financeTaskRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

func (r financeTaskRequest) Validate() error {
	if r.Status != string(FinanceTaskResolved) && r.Status != string(FinanceTaskRejected) {
		return errors.New("invalid status")
	}
	if strings.TrimSpace(r.Note) == "" {
		return errors.New("note is required")
	}
	return nil
}

func (h *AdminHandler) HandleResolveFinanceTask() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		req, ok := httputil.DecodeAndValidate[financeTaskRequest](w, r)
		if !ok {
			return
		}
		err := h.store.ResolveFinanceTask(r.Context(), id, actorFromRequest(r), req.Status, req.Note)
		if err != nil {
			switch {
			case errors.Is(err, ErrFinanceTaskNotFound):
				httputil.RespondError(w, http.StatusNotFound, "finance_task_not_found", "财务任务不存在")
			case errors.Is(err, ErrFinanceTaskInvalidTransition):
				httputil.RespondError(w, http.StatusConflict, "finance_task_invalid_transition", "财务任务状态已变化，请刷新后重试")
			default:
				slog.Error("admin referral: resolve finance task", "id", id, "err", err)
				httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "处理财务任务失败")
			}
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"result": "ok"})
	}
}

type revealPayoutRequest struct {
	Reason string `json:"reason"`
}

func (r revealPayoutRequest) Validate() error {
	if strings.TrimSpace(r.Reason) == "" {
		return errors.New("reason is required")
	}
	return nil
}

func (h *AdminHandler) HandleRevealPayoutAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		req, ok := httputil.DecodeAndValidate[revealPayoutRequest](w, r)
		if !ok {
			return
		}
		account, err := h.store.RevealPayoutAccount(r.Context(), id, actorFromRequest(r), req.Reason)
		if err != nil {
			switch {
			case errors.Is(err, ErrPayoutNotFound):
				httputil.RespondError(w, http.StatusNotFound, "payout_not_found", "提现不存在")
			default:
				slog.Error("admin referral: reveal payout account", "id", id, "err", err)
				httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "读取收款账号失败")
			}
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"account": account})
	}
}
