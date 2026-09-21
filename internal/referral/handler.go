package referral

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"liki-web/internal/auth"
	"liki-web/internal/httputil"
)

type RequireUserFunc func(http.HandlerFunc) http.HandlerFunc
type RateLimiterFunc func(http.HandlerFunc) http.HandlerFunc

type Handler struct {
	store        *Store
	requireUser  RequireUserFunc
	secureCookie bool
}

func NewHandler(store *Store, requireUser RequireUserFunc, secureCookie bool) *Handler {
	return &Handler{store: store, requireUser: requireUser, secureCookie: secureCookie}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, limit RateLimiterFunc) {
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("POST /api/referral/register", h.requireUser(limit(h.HandleRegister)))

	mux.HandleFunc("GET /api/kol/status", h.requireUser(h.HandleKOLStatus))
	mux.HandleFunc("GET /api/kol/session", h.requireUser(h.requireKOL(h.HandleKOLSession)))
	mux.HandleFunc("GET /api/kol/overview", h.requireUser(h.requireKOL(h.HandleKOLSession)))
	mux.HandleFunc("GET /api/kol/commissions", h.requireUser(h.requireKOL(h.HandleKOLCommissions)))
	mux.HandleFunc("GET /api/kol/payouts", h.requireUser(h.requireKOL(h.HandleKOLPayouts)))
	mux.HandleFunc("POST /api/kol/payouts", h.requireUser(h.requireKOL(h.HandleCreateKOLPayout)))

	mux.HandleFunc("GET /api/scout/session", h.requireUser(h.requireScout(h.HandleScoutSession)))
	mux.HandleFunc("GET /api/scout/overview", h.requireUser(h.requireScout(h.HandleScoutSession)))
	mux.HandleFunc("GET /api/scout/recruits", h.requireUser(h.requireScout(h.HandleScoutRecruits)))
	mux.HandleFunc("GET /api/scout/commissions", h.requireUser(h.requireScout(h.HandleScoutCommissions)))
	mux.HandleFunc("GET /api/scout/payouts", h.requireUser(h.requireScout(h.HandleScoutPayouts)))
	mux.HandleFunc("POST /api/scout/payouts", h.requireUser(h.requireScout(h.HandleCreateScoutPayout)))
	mux.HandleFunc("GET /r/{code}", h.HandleLink)
}

// PostLoginHook is called by auth after a successful email login. Referral
// attribution applies to freshly-created accounts only; the 24h window lets a
// transient attribution failure heal on the account's first retry.
func (h *Handler) PostLoginHook(w http.ResponseWriter, r *http.Request, user auth.User, isNewUser bool) {
	if !isNewUser || user.CreatedAt.IsZero() {
		return
	}
	cookie, err := r.Cookie(AttributionCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return
	}
	if _, err := h.store.Attribute(r.Context(), user.ID, cookie.Value); err != nil {
		if errors.Is(err, ErrInvalidCode) || errors.Is(err, ErrSelfReferral) {
			return
		}
		// Do not fail an already-created authenticated session. The cookie
		// remains available for retry within the 24h new-account window.
		slog.Warn("referral: attribute new account", "user_id", user.ID, "err", err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: AttributionCookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: h.secureCookie, SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.RespondAppError(w, httputil.ErrUnauthorized)
		return
	}
	item, err := h.store.RegisterReferrer(r.Context(), user.ID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotEligible):
			httputil.RespondError(w, http.StatusForbidden, "referral_not_eligible", "需要先购买并激活 Liki Chat")
		case errors.Is(err, ErrSelfReferral):
			httputil.RespondError(w, http.StatusBadRequest, "self_referral", "不能通过自己的链接开通")
		default:
			slog.Error("referral: register", "user_id", user.ID, "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		}
		return
	}
	stats, err := h.store.PartnerStats(r.Context(), item.UserID)
	if err != nil {
		slog.Error("referral: register stats", "user_id", item.UserID, "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"partner": stats})
}

// HandleLink is the only public referral entry point. It validates the link
// identifier before writing the HttpOnly attribution cookie.
func (h *Handler) HandleLink(w http.ResponseWriter, r *http.Request) {
	if _, err := h.store.ReferrerByCode(r.Context(), r.PathValue("code")); err == nil {
		http.SetCookie(w, &http.Cookie{
			Name:     AttributionCookieName,
			Value:    strings.ToUpper(strings.TrimSpace(r.PathValue("code"))),
			Path:     "/",
			MaxAge:   int(AttributionCookieTTL.Seconds()),
			Expires:  time.Now().UTC().Add(AttributionCookieTTL),
			HttpOnly: true,
			Secure:   h.secureCookie,
			SameSite: http.SameSiteLaxMode,
		})
	}
	http.Redirect(w, r, "/zh-Hans/", http.StatusFound)
}

func (h *Handler) HandleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.RespondAppError(w, httputil.ErrUnauthorized)
		return
	}
	item, err := h.store.ReferrerByUser(r.Context(), user.ID)
	if err == nil {
		stats, statsErr := h.store.PartnerStats(r.Context(), item.UserID)
		if statsErr != nil {
			slog.Error("referral: stats", "user_id", item.UserID, "err", statsErr)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]any{"partner": stats})
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		slog.Error("referral: me", "user_id", user.ID, "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	eligible, err := h.store.paidCheck(r.Context(), user.ID)
	if err != nil {
		slog.Error("referral: eligibility", "user_id", user.ID, "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{
		"registered": false, "eligible": eligible,
	})
}

func (h *Handler) HandleCommissions(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.RespondAppError(w, httputil.ErrUnauthorized)
		return
	}
	stats, err := h.store.PartnerStats(r.Context(), user.ID)
	if errors.Is(err, sql.ErrNoRows) {
		httputil.RespondError(w, http.StatusForbidden, "referral_not_registered", "尚未开通推广资格")
		return
	}
	if err != nil {
		slog.Error("referral: commissions", "user_id", user.ID, "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"partner": stats})
}

type payoutRequest struct {
	Method  string `json:"method"`
	Account string `json:"account"`
}

func (r payoutRequest) Validate() error { return nil }

func maskPayoutAccount(account string) string {
	if at := strings.Index(account, "@"); at > 0 {
		return account[:1] + "***" + account[at:]
	}
	if len(account) > 7 {
		return account[:2] + "****" + account[len(account)-4:]
	}
	return "****"
}

func (h *Handler) createPayout(w http.ResponseWriter, r *http.Request, user auth.User, role PartnerRole) {
	req, ok := httputil.DecodeAndValidate[payoutRequest](w, r)
	if !ok {
		return
	}
	item, err := h.store.CreatePayoutForRole(r.Context(), user.ID, role, PayoutMethod(req.Method), req.Account)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidPayoutRequest):
			httputil.RespondError(w, http.StatusUnprocessableEntity, "validation_error", "提现方式或账号无效")
		case errors.Is(err, ErrPartnerNotFound):
			httputil.RespondError(w, http.StatusForbidden, "role_required", "当前账号未开通该合作权限")
		case errors.Is(err, ErrNoWithdrawableCommission):
			httputil.RespondError(w, http.StatusConflict, "no_withdrawable_commission", "没有可提现佣金")
		case errors.Is(err, ErrMixedPayoutCurrency):
			httputil.RespondError(w, http.StatusConflict, "mixed_payout_currency", "待提现佣金包含多种币种，请联系运营处理")
		case errors.Is(err, ErrPayoutMinimumNotMet):
			httputil.RespondError(w, http.StatusUnprocessableEntity, "payout_minimum_not_met", "未达到最低提现金额")
		default:
			slog.Error("referral: create payout", "user_id", user.ID, "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		}
		return
	}
	item.Account = maskPayoutAccount(item.Account)
	httputil.RespondJSON(w, http.StatusCreated, map[string]any{"payout": item})
}

func (h *Handler) HandleListPayouts(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.RespondAppError(w, httputil.ErrUnauthorized)
		return
	}
	items, err := h.store.ListPayoutsForUser(r.Context(), user.ID, 100)
	if err != nil {
		slog.Error("referral: list payouts", "user_id", user.ID, "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	for i := range items {
		items[i].Account = maskPayoutAccount(items[i].Account)
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"payouts": items})
}
