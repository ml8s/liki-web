package referral

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"liki-web/internal/auth"
	"liki-web/internal/httputil"
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
			slog.Error("referral: load kol", "user_id", user.ID, "err", err)
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
			slog.Error("referral: load scout", "user_id", user.ID, "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
			return
		}
		next(w, r)
		_ = partner
	}
}

func (h *Handler) HandleKOLStatus(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	status, err := h.store.KOLStatus(r.Context(), user.ID)
	if err != nil {
		slog.Error("referral: kol status", "user_id", user.ID, "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, status)
}

func (h *Handler) HandleKOLSession(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	partner, err := h.store.KOLByID(r.Context(), user.ID)
	if errors.Is(err, sql.ErrNoRows) {
		httputil.RespondError(w, http.StatusForbidden, "role_required", "当前账号未开通该合作权限")
		return
	}
	if err != nil {
		slog.Error("referral: kol session", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	overview, err := h.store.KOLOverview(r.Context(), user.ID)
	if err != nil {
		slog.Error("referral: kol overview", "err", err)
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
		slog.Error("referral: kol commissions", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"commissions": items, "balances": balances})
}

func (h *Handler) HandleKOLPayouts(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if _, err := h.store.RequireKOL(r.Context(), user.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.RespondError(w, http.StatusForbidden, "role_required", "当前账号未开通该合作权限")
			return
		}
		slog.Error("referral: kol payouts", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	items, err := h.store.ListPayoutsForUser(r.Context(), user.ID, 100)
	if err != nil {
		slog.Error("referral: kol payout list", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"payouts": items})
}

func (h *Handler) HandleCreateKOLPayout(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if _, err := h.store.RequireKOL(r.Context(), user.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.RespondError(w, http.StatusForbidden, "role_required", "当前账号未开通该合作权限")
			return
		}
		slog.Error("referral: create kol payout", "err", err)
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
		slog.Error("referral: scout overview", "err", err)
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
		slog.Error("referral: scout recruits", "err", err)
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
		slog.Error("referral: scout commissions", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"sales": items, "balances": balances})
}

func (h *Handler) HandleScoutPayouts(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if _, err := h.store.RequireScout(r.Context(), user.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.RespondError(w, http.StatusNotFound, "not_found", "页面不存在")
			return
		}
		slog.Error("referral: scout payouts", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	items, err := h.store.ListPayoutsForUser(r.Context(), user.ID, 100)
	if err != nil {
		slog.Error("referral: scout payout list", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"payouts": items})
}

func (h *Handler) HandleCreateScoutPayout(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if _, err := h.store.RequireScout(r.Context(), user.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.RespondError(w, http.StatusNotFound, "not_found", "页面不存在")
			return
		}
		slog.Error("referral: create scout payout", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	h.createPayout(w, r, user, RoleScout)
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
