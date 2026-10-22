package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ml8s/liki-web/internal/httputil"
)

type RateLimiterFunc func(http.HandlerFunc) http.HandlerFunc

type Handler struct {
	store         *Store
	service       *Service
	config        Config
	postLoginHook PostLoginHook
	partnerRoles  PartnerRolesProvider
}

func NewHandler(store *Store, service *Service, config Config) (*Handler, error) {
	if store == nil || service == nil {
		return nil, errors.New("auth: store and service are required")
	}
	if err := config.normalize(); err != nil {
		return nil, err
	}
	return &Handler{store: store, service: service, config: config}, nil
}

type PostLoginHook func(w http.ResponseWriter, r *http.Request, user User, isNewUser bool)

func (h *Handler) SetPostLoginHook(hook PostLoginHook) {
	h.postLoginHook = hook
}

// PartnerRolesProvider 返回账号的合作角色列表（身份摘要）。由 partner store
// 在启动时注入，auth 模块不依赖 partner 模块。
type PartnerRolesProvider func(ctx context.Context, userID string) ([]string, error)

func (h *Handler) SetPartnerRolesProvider(provider PartnerRolesProvider) {
	h.partnerRoles = provider
}

type sendCodeRequest struct {
	Email  string `json:"email"`
	Locale string `json:"locale,omitempty"`
}

func (r sendCodeRequest) Validate() error {
	_, err := NormalizeEmail(r.Email)
	return err
}

type verifyRequest struct {
	Email  string `json:"email"`
	Code   string `json:"code"`
	Locale string `json:"locale,omitempty"`
}

func (r verifyRequest) Validate() error {
	if _, err := NormalizeEmail(r.Email); err != nil {
		return err
	}
	if strings.TrimSpace(r.Code) == "" {
		return errors.New("code is required")
	}
	return nil
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, sendCodeLimiter, verifyLimiter RateLimiterFunc) {
	if sendCodeLimiter == nil {
		sendCodeLimiter = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if verifyLimiter == nil {
		verifyLimiter = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}

	mux.HandleFunc("POST /api/auth/email/code", sendCodeLimiter(h.HandleSendEmailCode))
	mux.HandleFunc("POST /api/auth/email/verify", verifyLimiter(h.HandleVerifyEmailCode))
	mux.HandleFunc("POST /api/auth/logout", h.RequireUser(h.HandleLogout))
	mux.HandleFunc("POST /api/auth/sessions/revoke-all", h.RequireUser(h.HandleRevokeAllSessions))
	mux.HandleFunc("GET /api/me", h.RequireUser(h.HandleMe))
	mux.HandleFunc("GET /api/me/auth/identities", h.RequireUser(h.HandleAuthIdentities))
	mux.HandleFunc("GET /api/me/sessions", h.RequireUser(h.HandleSessions))
	mux.HandleFunc("DELETE /api/me/sessions/{id}", h.RequireUser(h.HandleRevokeSession))
}

func (h *Handler) HandleSendEmailCode(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeStrict[sendCodeRequest](w, r)
	if !ok {
		return
	}
	issued, err := h.service.RequestEmailCode(r.Context(), req.Email, clientIP(r), userAgent(r), req.Locale)
	if err != nil {
		h.writeError(w, err)
		return
	}
	response := map[string]any{
		"email":      issued.Email,
		"expires_at": issued.ExpiresAt,
	}
	if issued.DevCode != "" {
		response["dev_code"] = issued.DevCode
	}
	httputil.RespondJSON(w, http.StatusOK, response)
}

func (h *Handler) HandleVerifyEmailCode(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeStrict[verifyRequest](w, r)
	if !ok {
		return
	}
	result, err := h.service.VerifyEmailCode(r.Context(), req.Email, req.Code, clientIP(r), userAgent(r), req.Locale)
	if err != nil {
		h.writeError(w, err)
		return
	}
	maxAge := h.service.SessionCookieMaxAge()
	http.SetCookie(w, &http.Cookie{
		Name:     h.config.CookieName,
		Value:    result.SessionToken,
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  result.Session.ExpiresAt,
		HttpOnly: true,
		Secure:   h.config.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	})
	if h.postLoginHook != nil {
		h.postLoginHook(w, r, result.User, result.IsNewUser)
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{
		"user":       h.publicUserFor(r.Context(), result.User),
		"session_id": result.Session.ID,
		"expires_at": result.Session.ExpiresAt,
	})
}

func (h *Handler) RequireUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(h.config.CookieName)
		if err != nil || cookie.Value == "" {
			httputil.Unauthorized(w)
			return
		}
		user, session, err := h.store.GetSessionUser(r.Context(), cookie.Value)
		if err != nil {
			h.clearSessionCookie(w)
			httputil.Unauthorized(w)
			return
		}
		now := time.Now().UTC()
		_ = h.store.TouchSession(r.Context(), session.ID, now)
		ctx := WithAuth(r.Context(), user, session, cookie.Value)
		next(w, r.WithContext(ctx))
	}
}

// IdentityFromRequest resolves the current unified app session without
// mounting authenticated routes. It returns only active sessions.
func (h *Handler) IdentityFromRequest(r *http.Request) (User, bool) {
	cookie, err := r.Cookie(h.config.CookieName)
	if err != nil || cookie.Value == "" {
		return User{}, false
	}
	user, _, err := h.store.GetSessionUser(r.Context(), cookie.Value)
	if err != nil {
		return User{}, false
	}
	return user, true
}

func (h *Handler) HandleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, h.publicUserFor(r.Context(), user))
}

func (h *Handler) HandleAuthIdentities(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	identities, err := h.store.ListIdentities(r.Context(), user.ID)
	if err != nil {
		httputil.InternalError(w)
		return
	}
	_, hasToken := SessionTokenFromContext(r.Context())
	out := make([]PublicAuthIdentity, 0, len(identities))
	for _, identity := range identities {
		out = append(out, PublicAuthIdentity{
			ID: identity.ID, Provider: string(identity.Provider), Email: identity.Email,
			CreatedAt: identity.CreatedAt, LastUsedAt: identity.LastUsedAt,
			IsCurrent: hasToken && identity.Provider == ProviderEmail,
		})
	}
	httputil.RespondJSON(w, http.StatusOK, out)
}

func (h *Handler) HandleSessions(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	currentToken, hasToken := SessionTokenFromContext(r.Context())
	currentHash := ""
	if hasToken {
		currentHash = h.store.sessionTokenHash(currentToken)
	}
	sessions, err := h.store.ListSessions(r.Context(), user.ID)
	if err != nil {
		httputil.InternalError(w)
		return
	}
	out := make([]PublicSession, 0, len(sessions))
	for _, session := range sessions {
		out = append(out, PublicSession{
			ID: session.ID, CreatedAt: session.CreatedAt, ExpiresAt: session.ExpiresAt,
			LastSeenAt: session.LastSeenAt, IPHash: session.IPHash, UserAgent: session.UserAgent,
			IsCurrent: hasToken && session.TokenHash == currentHash,
		})
	}
	httputil.RespondJSON(w, http.StatusOK, out)
}

func (h *Handler) HandleRevokeSession(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	sessionID := r.PathValue("id")
	if sessionID == "" {
		httputil.InvalidRequest(w)
		return
	}
	currentToken, hasToken := SessionTokenFromContext(r.Context())
	currentSessionID := ""
	if hasToken {
		_, session, err := h.store.GetSessionUser(r.Context(), currentToken)
		if err == nil {
			currentSessionID = session.ID
		}
	}
	if sessionID == currentSessionID {
		h.HandleLogout(w, r)
		return
	}
	if err := h.store.RevokeSession(r.Context(), user.ID, sessionID); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			httputil.NotFound(w)
			return
		}
		httputil.InternalError(w)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *Handler) HandleRevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	currentToken, hasToken := SessionTokenFromContext(r.Context())
	currentSessionID := ""
	if hasToken {
		if _, session, err := h.store.GetSessionUser(r.Context(), currentToken); err == nil {
			currentSessionID = session.ID
		}
	}
	count, err := h.store.RevokeAllSessions(r.Context(), user.ID, currentSessionID)
	if err != nil {
		httputil.InternalError(w)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"status": "revoked", "count": count})
}

func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	token, ok := SessionTokenFromContext(r.Context())
	if !ok {
		h.clearSessionCookie(w)
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
		return
	}
	user, session, err := h.store.GetSessionUser(r.Context(), token)
	if err == nil {
		_ = h.store.RevokeSession(r.Context(), user.ID, session.ID)
	}
	h.clearSessionCookie(w)
	httputil.RespondJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidEmail):
		httputil.RespondError(w, http.StatusUnprocessableEntity, "invalid_email", "邮箱格式不正确")
	case errors.Is(err, ErrCodeCooldown):
		httputil.RespondError(w, http.StatusTooManyRequests, "code_cooldown", "验证码请求过于频繁")
	case errors.Is(err, ErrCodeInvalid):
		httputil.RespondError(w, http.StatusBadRequest, "invalid_code", "验证码不正确")
	case errors.Is(err, ErrCodeExpired):
		httputil.RespondError(w, http.StatusUnauthorized, "code_expired", "验证码已过期")
	case errors.Is(err, ErrCodeLocked):
		httputil.RespondError(w, http.StatusLocked, "code_locked", "验证码已锁定")
	case errors.Is(err, ErrUserDisabled):
		httputil.RespondError(w, http.StatusForbidden, "user_disabled", "账户不可用")
	case errors.Is(err, ErrUserNotFound):
		httputil.RespondError(w, http.StatusNotFound, "user_not_found", "用户不存在")
	case errors.Is(err, ErrMailerUnavailable):
		httputil.RespondError(w, http.StatusServiceUnavailable, "mailer_unavailable", "邮件服务暂不可用")
	default:
		httputil.InternalError(w)
	}
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: h.config.CookieName, Value: "", Path: "/", MaxAge: -1,
		Expires: time.Unix(1, 0), HttpOnly: true, Secure: h.config.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func publicUser(user User) PublicUser {
	return PublicUser{ID: user.ID, Email: user.Email, Status: user.Status, Locale: user.Locale, LastLoginAt: user.LastLoginAt}
}

// publicUserFor 在身份响应上附加合作角色摘要。查询失败时降级为空角色
// （入口不显示）并记日志——身份接口本身不因附属查询失败而不可用。
func (h *Handler) publicUserFor(ctx context.Context, user User) PublicUser {
	out := publicUser(user)
	out.PartnerRoles = []string{}
	if h.partnerRoles == nil {
		return out
	}
	roles, err := h.partnerRoles(ctx, user.ID)
	if err != nil {
		slog.WarnContext(ctx, "auth: partner roles lookup", "user_id", user.ID, "err", err)
		return out
	}
	if roles != nil {
		out.PartnerRoles = roles
	}
	return out
}

func clientIP(r *http.Request) string {
	return httputil.ClientIP(r)
}

func userAgent(r *http.Request) string { return r.UserAgent() }
