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
	mux.HandleFunc("GET /api/referral/me", h.requireUser(h.HandleMe))
	mux.HandleFunc("POST /api/referral/register", h.requireUser(limit(h.HandleRegister)))
	mux.HandleFunc("GET /api/referral/me/commissions", h.requireUser(h.HandleCommissions))
}

// CaptureClick records the landing touch as a 30-day HttpOnly cookie. The code
// is validated again from referrers when a new account is created.
func (h *Handler) CaptureClick(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("ref")))
		if len(code) >= 4 && len(code) <= 32 {
			http.SetCookie(w, &http.Cookie{
				Name:     AttributionCookieName,
				Value:    code,
				Path:     "/",
				MaxAge:   int(AttributionCookieTTL.Seconds()),
				Expires:  time.Now().UTC().Add(AttributionCookieTTL),
				HttpOnly: true,
				Secure:   h.secureCookie,
				SameSite: http.SameSiteLaxMode,
			})
		}
		next.ServeHTTP(w, r)
	})
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

type registerRequest struct {
	RecruiterCode string `json:"recruiter_code"`
}

func (r registerRequest) Validate() error { return nil }

func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.RespondAppError(w, httputil.ErrUnauthorized)
		return
	}
	req, ok := httputil.DecodeAndValidate[registerRequest](w, r)
	if !ok {
		return
	}
	item, err := h.store.RegisterReferrer(r.Context(), user.ID, req.RecruiterCode)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotEligible):
			httputil.RespondError(w, http.StatusForbidden, "referral_not_eligible", "需要先购买并激活 Liki Chat")
		case errors.Is(err, ErrInvalidCode):
			httputil.RespondError(w, http.StatusUnprocessableEntity, "invalid_referral_code", "招商码无效")
		case errors.Is(err, ErrSelfReferral):
			httputil.RespondError(w, http.StatusBadRequest, "self_referral", "不能使用自己的招商码")
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
