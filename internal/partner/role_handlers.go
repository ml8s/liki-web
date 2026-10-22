package partner

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
)

func (h *Handler) requireKOL(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.UserFromContext(r.Context())
		if !ok {
			httputil.RespondAppError(w, httputil.ErrUnauthorized)
			return
		}
		partner, err := h.store.KOLByID(r.Context(), user.ID)
		if errors.Is(err, sql.ErrNoRows) {
			httputil.RespondError(w, http.StatusForbidden, "role_required", "当前账号未开通该合作权限")
			return
		}
		if err != nil {
			slog.ErrorContext(r.Context(), "partner: load kol", "user_id", user.ID, "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
			return
		}
		next(w, r)
		_ = partner
	}
}

func (h *Handler) requireScout(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.UserFromContext(r.Context())
		if !ok {
			httputil.RespondAppError(w, httputil.ErrUnauthorized)
			return
		}
		partner, err := h.store.ScoutByID(r.Context(), user.ID)
		if errors.Is(err, sql.ErrNoRows) {
			httputil.RespondError(w, http.StatusNotFound, "not_found", "页面不存在")
			return
		}
		if err != nil {
			slog.ErrorContext(r.Context(), "partner: load scout", "user_id", user.ID, "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
			return
		}
		next(w, r)
		_ = partner
	}
}

func (h *Handler) HandleKOLSession(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	partner, err := h.store.KOLByID(r.Context(), user.ID)
	if errors.Is(err, sql.ErrNoRows) {
		httputil.RespondError(w, http.StatusForbidden, "role_required", "当前账号未开通该合作权限")
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "partner: kol session", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	overview, err := h.store.KOLOverview(r.Context(), user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "partner: kol overview", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{
		"email": user.Email, "partner": partner, "overview": overview,
	})
}

func (h *Handler) HandleKOLCommissions(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	limit, _ := parseIntQuery(r, "limit", 100)
	items, balances, err := h.store.ListKOLCommissions(r.Context(), user.ID, limit)
	if err != nil {
		slog.ErrorContext(r.Context(), "partner: kol commissions", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"commissions": items, "balances": balances})
}

func (h *Handler) HandleKOLPayouts(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if _, err := h.store.PartnerForPayout(r.Context(), user.ID, RoleKOL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.RespondError(w, http.StatusForbidden, "role_required", "当前账号未开通该合作权限")
			return
		}
		slog.ErrorContext(r.Context(), "partner: kol payouts", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	items, err := h.store.ListPayoutsForRole(r.Context(), user.ID, RoleKOL, 100)
	if err != nil {
		slog.ErrorContext(r.Context(), "partner: kol payout list", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"payouts": items})
}

func (h *Handler) HandleKOLSettlement(w http.ResponseWriter, r *http.Request) {
	h.settlement(w, r, RoleKOL)
}

func (h *Handler) HandleCreateKOLPayout(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if _, err := h.store.PartnerForPayout(r.Context(), user.ID, RoleKOL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.RespondError(w, http.StatusForbidden, "role_required", "当前账号未开通该合作权限")
			return
		}
		slog.ErrorContext(r.Context(), "partner: create kol payout", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	h.createPayout(w, r, user, RoleKOL)
}

func (h *Handler) HandleScoutSession(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	overview, err := h.store.ScoutOverview(r.Context(), user.ID)
	if errors.Is(err, sql.ErrNoRows) {
		httputil.RespondError(w, http.StatusNotFound, "not_found", "页面不存在")
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "partner: scout overview", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{
		"email": user.Email, "overview": overview,
	})
}

func (h *Handler) HandleScoutRecruits(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	items, err := h.store.ScoutRecruits(r.Context(), user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "partner: scout recruits", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"recruits": items})
}

func (h *Handler) HandleScoutCommissions(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	limit, _ := parseIntQuery(r, "limit", 100)
	items, balances, err := h.store.ListScoutCommissions(r.Context(), user.ID, limit)
	if err != nil {
		slog.ErrorContext(r.Context(), "partner: scout commissions", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"sales": items, "balances": balances})
}

func (h *Handler) HandleScoutPayouts(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if _, err := h.store.PartnerForPayout(r.Context(), user.ID, RoleScout); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.RespondError(w, http.StatusNotFound, "not_found", "页面不存在")
			return
		}
		slog.ErrorContext(r.Context(), "partner: scout payouts", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	items, err := h.store.ListPayoutsForRole(r.Context(), user.ID, RoleScout, 100)
	if err != nil {
		slog.ErrorContext(r.Context(), "partner: scout payout list", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"payouts": items})
}

func (h *Handler) HandleScoutSettlement(w http.ResponseWriter, r *http.Request) {
	h.settlement(w, r, RoleScout)
}

func (h *Handler) settlement(w http.ResponseWriter, r *http.Request, role PartnerRole) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.RespondAppError(w, httputil.ErrUnauthorized)
		return
	}
	item, err := h.store.SettlementForRole(r.Context(), user.ID, role)
	if errors.Is(err, sql.ErrNoRows) {
		httputil.RespondError(w, http.StatusForbidden, "role_required", "当前账号没有该角色结算权限")
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "partner: settlement", "user_id", user.ID, "role", role, "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, item)
}

func (h *Handler) HandleCreateScoutPayout(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if _, err := h.store.PartnerForPayout(r.Context(), user.ID, RoleScout); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.RespondError(w, http.StatusNotFound, "not_found", "页面不存在")
			return
		}
		slog.ErrorContext(r.Context(), "partner: create scout payout", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	h.createPayout(w, r, user, RoleScout)
}

func (h *Handler) createPayout(w http.ResponseWriter, r *http.Request, user auth.User, role PartnerRole) {
	req, ok := httputil.DecodeAndValidate[payoutRequest](w, r)
	if !ok {
		return
	}
	payout, err := h.store.CreatePayoutForRole(r.Context(), user.ID, role, req.Method, req.Account)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidPayoutRequest):
			httputil.RespondError(w, http.StatusUnprocessableEntity, "validation_error", "提现方式或账号无效")
		case errors.Is(err, ErrNoWithdrawableCommission):
			httputil.RespondError(w, http.StatusConflict, "no_withdrawable_commission", "没有可提现佣金")
		case errors.Is(err, ErrMixedPayoutCurrency):
			httputil.RespondError(w, http.StatusConflict, "mixed_currency", "待提现佣金包含多种货币，请联系客服")
		case errors.Is(err, ErrPayoutMinimumNotMet):
			httputil.RespondError(w, http.StatusUnprocessableEntity, "payout_minimum_not_met", "未达到最低提现金额")
		default:
			slog.ErrorContext(r.Context(), "partner: create payout", "user_id", user.ID, "role", role, "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "创建提现失败")
		}
		return
	}
	payout.Account = maskPayoutAccount(payout.Account)
	httputil.RespondJSON(w, http.StatusCreated, map[string]any{"payout": payout})
}

type payoutRequest struct {
	Method  PayoutMethod `json:"method"`
	Account string       `json:"account"`
}

func (r payoutRequest) Validate() error {
	if strings.TrimSpace(r.Account) == "" {
		return errors.New("account is required")
	}
	return nil
}

func maskPayoutAccount(account string) string {
	if at := strings.Index(account, "@"); at > 0 {
		return account[:1] + "***" + account[at:]
	}
	if len(account) > 7 {
		return account[:2] + "****" + account[len(account)-4:]
	}
	return "****"
}

func parseIntQuery(r *http.Request, name string, fallback int) (int, error) {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return fallback, nil
	}
	n := 0
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0, errors.New("invalid integer")
		}
		n = n*10 + int(ch-'0')
		if n > 10000 {
			return 0, errors.New("integer too large")
		}
	}
	if n <= 0 {
		return fallback, nil
	}
	return n, nil
}
