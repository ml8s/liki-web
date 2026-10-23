package partner

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
)

type RequireUserFunc func(http.HandlerFunc) http.HandlerFunc
type RateLimiterFunc func(http.HandlerFunc) http.HandlerFunc

type Handler struct {
	store          *Store
	requireUser    RequireUserFunc
	secureCookie   bool
	attributionTTL time.Duration
}

func NewHandler(store *Store, requireUser RequireUserFunc, secureCookie bool) *Handler {
	return &Handler{store: store, requireUser: requireUser, secureCookie: secureCookie, attributionTTL: AttributionCookieTTL}
}

// SetAttributionTTL overrides the attribution cookie lifetime (deployment config).
func (h *Handler) SetAttributionTTL(d time.Duration) { h.attributionTTL = d }

func (h *Handler) RegisterRoutes(mux *http.ServeMux, limit RateLimiterFunc) {
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/partner-portal", h.requireUser(h.HandlePartnerStates))
	mux.HandleFunc("GET /api/partner-portal/kol/application", h.requireUser(h.HandleGetKOLApplication))
	mux.HandleFunc("POST /api/partner-portal/kol/application", h.requireUser(limit(h.HandleSubmitKOLApplication)))
	mux.HandleFunc("GET /api/partner-portal/scout/application", h.requireUser(h.HandleGetScoutApplication))
	mux.HandleFunc("POST /api/partner-portal/scout/application", h.requireUser(limit(h.HandleSubmitScoutApplication)))

	mux.HandleFunc("GET /api/partner-portal/kol/session", h.requireUser(h.requireKOL(h.HandleKOLSession)))
	mux.HandleFunc("GET /api/partner-portal/kol/commissions", h.requireUser(h.requireKOL(h.HandleKOLCommissions)))
	mux.HandleFunc("GET /api/partner-portal/kol/payouts", h.requireUser(h.HandleKOLPayouts))
	mux.HandleFunc("POST /api/partner-portal/kol/payouts", h.requireUser(limit(h.HandleCreateKOLPayout)))
	mux.HandleFunc("GET /api/partner-portal/kol/settlement", h.requireUser(h.HandleKOLSettlement))

	mux.HandleFunc("GET /api/partner-portal/scout/session", h.requireUser(h.requireScout(h.HandleScoutSession)))
	mux.HandleFunc("GET /api/partner-portal/scout/recruits", h.requireUser(h.requireScout(h.HandleScoutRecruits)))
	mux.HandleFunc("GET /api/partner-portal/scout/commissions", h.requireUser(h.requireScout(h.HandleScoutCommissions)))
	mux.HandleFunc("GET /api/partner-portal/scout/payouts", h.requireUser(h.HandleScoutPayouts))
	mux.HandleFunc("POST /api/partner-portal/scout/payouts", h.requireUser(limit(h.HandleCreateScoutPayout)))
	mux.HandleFunc("GET /api/partner-portal/scout/settlement", h.requireUser(h.HandleScoutSettlement))
	mux.HandleFunc("GET /r/{code}", h.HandleLink)
}

// PostLoginHook attributes a newly-created account once. A 24-hour retry
// window lets a transient database failure heal without exposing referral
// state or failing an already-created authenticated session.
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
		slog.WarnContext(r.Context(), "partner: attribute new account", "user_id", user.ID, "err", err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: AttributionCookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: h.secureCookie, SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) HandlePartnerStates(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.RespondAppError(w, httputil.ErrUnauthorized)
		return
	}
	states, err := h.store.PartnerStates(r.Context(), user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "partner: partner states", "user_id", user.ID, "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, states)
}

func (h *Handler) HandleGetKOLApplication(w http.ResponseWriter, r *http.Request) {
	h.getApplication(w, r, RoleKOL)
}

func (h *Handler) HandleGetScoutApplication(w http.ResponseWriter, r *http.Request) {
	h.getApplication(w, r, RoleScout)
}

func (h *Handler) HandleSubmitKOLApplication(w http.ResponseWriter, r *http.Request) {
	h.submitApplication(w, r, RoleKOL)
}

func (h *Handler) HandleSubmitScoutApplication(w http.ResponseWriter, r *http.Request) {
	h.submitApplication(w, r, RoleScout)
}

func (h *Handler) getApplication(w http.ResponseWriter, r *http.Request, role PartnerRole) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.RespondAppError(w, httputil.ErrUnauthorized)
		return
	}
	states, err := h.store.PartnerStates(r.Context(), user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "partner: get application", "user_id", user.ID, "role", role, "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	state := states.KOL
	if role == RoleScout {
		state = states.Scout
	}
	httputil.RespondJSON(w, http.StatusOK, state)
}

type applicationRequest struct {
	Note string `json:"note"`
}

func (r applicationRequest) Validate() error {
	if strings.TrimSpace(r.Note) == "" {
		return errors.New("note is required")
	}
	return nil
}

func (h *Handler) submitApplication(w http.ResponseWriter, r *http.Request, role PartnerRole) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.RespondAppError(w, httputil.ErrUnauthorized)
		return
	}
	req, ok := httputil.DecodeStrict[applicationRequest](w, r)
	if !ok {
		return
	}
	item, err := h.store.SubmitApplication(r.Context(), role, user.ID, req.Note)
	if err != nil {
		switch {
		case errors.Is(err, ErrAlreadyKOL):
			httputil.RespondError(w, http.StatusConflict, "already_kol", "该账号已是 KOL")
		case errors.Is(err, ErrAlreadyScout):
			httputil.RespondError(w, http.StatusConflict, "already_scout", "该账号已是 Scout")
		case errors.Is(err, ErrRoleConflict):
			httputil.RespondError(w, http.StatusConflict, "role_conflict", "KOL 和 Scout 是不同角色，不能互相转换")
		case errors.Is(err, ErrApplicationPending):
			httputil.RespondError(w, http.StatusConflict, "application_pending", "申请正在处理中")
		default:
			slog.ErrorContext(r.Context(), "partner: submit application", "user_id", user.ID, "role", role, "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "提交申请失败")
		}
		return
	}
	httputil.RespondJSON(w, http.StatusCreated, map[string]any{"application": item})
}

// HandleLink is the only public referral entry point. It validates the link
// identifier before writing an HttpOnly attribution cookie.
func (h *Handler) HandleLink(w http.ResponseWriter, r *http.Request) {
	if _, err := h.store.PartnerByCode(r.Context(), r.PathValue("code")); err == nil {
		http.SetCookie(w, &http.Cookie{
			Name:     AttributionCookieName,
			Value:    strings.ToUpper(strings.TrimSpace(r.PathValue("code"))),
			Path:     "/",
			MaxAge:   int(h.attributionTTL.Seconds()),
			Expires:  time.Now().UTC().Add(h.attributionTTL),
			HttpOnly: true,
			Secure:   h.secureCookie,
			SameSite: http.SameSiteLaxMode,
		})
	}
	http.Redirect(w, r, referralRedirectPath(r), http.StatusFound)
}

// safeReferralTarget only allows same-origin marketing paths and the App home.
// It deliberately rejects private App routes so referral links cannot become a
// way to share account, profile, chat, or Admin URLs.
func safeReferralTarget(rawTarget string) (string, bool) {
	rawTarget = strings.TrimSpace(rawTarget)
	if rawTarget == "" || !strings.HasPrefix(rawTarget, "/") || strings.HasPrefix(rawTarget, "//") {
		return "", false
	}

	parsed, err := url.Parse(rawTarget)
	if err != nil || parsed.Host != "" || parsed.Scheme != "" {
		return "", false
	}

	cleanPath := path.Clean("/" + strings.TrimPrefix(parsed.Path, "/"))
	if cleanPath == "/app" {
		parsed.Path = cleanPath
		parsed.Fragment = ""
		return parsed.String(), true
	}

	allowedPaths := map[string]bool{"/": true, "/app": true}
	for _, locale := range []string{"zh-Hans", "zh-Hant", "en"} {
		allowedPaths["/"+locale] = true
		allowedPaths["/"+locale+"/"] = true
		for _, slug := range []string{
			"natal", "liunian", "hepan", "divination", "fengshui", "huangli", "naming", "skills",
			"about", "contact", "privacy", "terms", "disclaimer",
		} {
			allowedPaths["/"+locale+"/"+slug] = true
			allowedPaths["/"+locale+"/"+slug+"/"] = true
		}
	}
	if !allowedPaths[cleanPath] {
		return "", false
	}

	parsed.Path = cleanPath
	parsed.Fragment = ""
	return parsed.String(), true
}

func referralRedirectPath(r *http.Request) string {
	if target, ok := safeReferralTarget(r.URL.Query().Get("to")); ok {
		return target
	}
	return landingPath(r)
}

// landingPath preserves campaign query parameters and selects a supported
// locale from an explicit query value, then Accept-Language.
func landingPath(r *http.Request) string {
	locale := "/zh-Hans/"
	lang := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang")))
	if lang == "" {
		lang = strings.TrimSpace(strings.Split(r.Header.Get("Accept-Language"), ",")[0])
		if before, _, found := strings.Cut(lang, ";"); found {
			lang = strings.TrimSpace(before)
		}
	}
	switch lang {
	case "en":
		locale = "/en/"
	case "zh-hant", "zh-tw", "zh-hk", "zh":
		locale = "/zh-Hant/"
	}
	query := r.URL.Query()
	query.Del("to")
	target := &url.URL{Path: locale, RawQuery: query.Encode()}
	return target.String()
}
