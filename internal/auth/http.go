package auth

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RateLimiterFunc func(http.HandlerFunc) http.HandlerFunc

type Handler struct {
	store         *Store
	service       *Service
	config        Config
	postLoginHook PostLoginHook
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

type sendCodeRequest struct {
	Email string `json:"email"`
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
	if !h.assertSameOrigin(w, r) {
		return
	}
	req, ok := decode[sendCodeRequest](w, r)
	if !ok {
		return
	}
	issued, err := h.service.RequestEmailCode(r.Context(), req.Email, clientIP(r), userAgent(r))
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
	responseOK(w, response)
}

func (h *Handler) HandleVerifyEmailCode(w http.ResponseWriter, r *http.Request) {
	if !h.assertSameOrigin(w, r) {
		return
	}
	req, ok := decode[verifyRequest](w, r)
	if !ok {
		return
	}
	result, err := h.service.VerifyEmailCode(r.Context(), req.Email, req.Code, clientIP(r), userAgent(r))
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
	responseOK(w, map[string]any{
		"user":       publicUser(result.User),
		"session_id": result.Session.ID,
		"expires_at": result.Session.ExpiresAt,
	})
}

func (h *Handler) RequireUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(h.config.CookieName)
		if err != nil || cookie.Value == "" {
			unauthorized(w)
			return
		}
		user, session, err := h.store.GetSessionUser(r.Context(), cookie.Value)
		if err != nil {
			h.clearSessionCookie(w)
			unauthorized(w)
			return
		}
		now := time.Now().UTC()
		_ = h.store.TouchSession(r.Context(), session.ID, now)
		ctx := WithAuth(r.Context(), user, session, cookie.Value)
		next(w, r.WithContext(ctx))
	}
}

func (h *Handler) HandleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	responseOK(w, publicUser(user))
}

func (h *Handler) HandleAuthIdentities(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	identities, err := h.store.ListIdentities(r.Context(), user.ID)
	if err != nil {
		internal(w)
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
	responseOK(w, out)
}

func (h *Handler) HandleSessions(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	currentToken, hasToken := SessionTokenFromContext(r.Context())
	currentHash := ""
	if hasToken {
		currentHash = h.store.sessionTokenHash(currentToken)
	}
	sessions, err := h.store.ListSessions(r.Context(), user.ID)
	if err != nil {
		internal(w)
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
	responseOK(w, out)
}

func (h *Handler) HandleRevokeSession(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	sessionID := r.PathValue("id")
	if sessionID == "" {
		invalidRequest(w)
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
			notFound(w)
			return
		}
		internal(w)
		return
	}
	responseOK(w, map[string]string{"status": "revoked"})
}

func (h *Handler) HandleRevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
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
		internal(w)
		return
	}
	responseOK(w, map[string]any{"status": "revoked", "count": count})
}

func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	token, ok := SessionTokenFromContext(r.Context())
	if !ok {
		h.clearSessionCookie(w)
		responseOK(w, map[string]string{"status": "logged_out"})
		return
	}
	user, session, err := h.store.GetSessionUser(r.Context(), token)
	if err == nil {
		_ = h.store.RevokeSession(r.Context(), user.ID, session.ID)
	}
	h.clearSessionCookie(w)
	responseOK(w, map[string]string{"status": "logged_out"})
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidEmail):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]string{"code": "invalid_email", "message": "邮箱格式不正确"}})
	case errors.Is(err, ErrCodeCooldown):
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": map[string]string{"code": "code_cooldown", "message": "验证码请求过于频繁"}})
	case errors.Is(err, ErrCodeInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"code": "invalid_code", "message": "验证码不正确"}})
	case errors.Is(err, ErrCodeExpired):
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "code_expired", "message": "验证码已过期"}})
	case errors.Is(err, ErrCodeLocked):
		writeJSON(w, http.StatusLocked, map[string]any{"error": map[string]string{"code": "code_locked", "message": "验证码已锁定"}})
	case errors.Is(err, ErrUserDisabled):
		writeJSON(w, http.StatusForbidden, map[string]any{"error": map[string]string{"code": "user_disabled", "message": "账户不可用"}})
	case errors.Is(err, ErrUserNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"error": map[string]string{"code": "user_not_found", "message": "用户不存在"}})
	case errors.Is(err, ErrMailerUnavailable):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": map[string]string{"code": "mailer_unavailable", "message": "邮件服务暂不可用"}})
	default:
		internal(w)
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

func responseOK(w http.ResponseWriter, data any) {
	type envelope struct {
		Data any `json:"data"`
	}
	writeJSON(w, http.StatusOK, envelope{Data: data})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	// Keep local to avoid coupling auth to shared response helper behavior.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = jsonEncode(w, body)
}

func clientIP(r *http.Request) string {
	if value := r.Header.Get("CF-Connecting-IP"); value != "" {
		return normalizeIP(value)
	}
	if value := r.Header.Get("X-Forwarded-For"); value != "" {
		parts := strings.Split(value, ",")
		return normalizeIP(strings.TrimSpace(parts[0]))
	}
	if value := r.Header.Get("X-Real-IP"); value != "" {
		return normalizeIP(value)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func normalizeIP(value string) string {
	value = strings.TrimSpace(value)
	if host, _, err := net.SplitHostPort(value); err == nil {
		return host
	}
	if parsed := net.ParseIP(value); parsed != nil {
		return parsed.String()
	}
	return ""
}

func userAgent(r *http.Request) string { return r.UserAgent() }

func (h *Handler) assertSameOrigin(w http.ResponseWriter, r *http.Request) bool {
	expected := expectedHost(r)
	if expected == "" {
		return true
	}
	for _, header := range []string{"Origin", "Referer"} {
		value := r.Header.Get(header)
		if value == "" {
			continue
		}
		parsed, err := url.Parse(value)
		if err != nil || parsed.Host == "" || parsed.Host != expected {
			forbidden(w)
			return false
		}
	}
	return true
}

func expectedHost(r *http.Request) string {
	if value := r.Header.Get("X-Forwarded-Host"); value != "" {
		return strings.TrimSpace(strings.Split(value, ",")[0])
	}
	return r.Host
}

func unauthorized(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "unauthorized", "message": "请先登录"}})
}

func forbidden(w http.ResponseWriter) {
	writeJSON(w, http.StatusForbidden, map[string]any{"error": map[string]string{"code": "forbidden", "message": "无权访问"}})
}

func notFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]any{"error": map[string]string{"code": "not_found", "message": "资源不存在"}})
}

func invalidRequest(w http.ResponseWriter) {
	writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"code": "invalid_request", "message": "请求无效"}})
}

func internal(w http.ResponseWriter) {
	writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"code": "internal_error", "message": "服务器内部错误"}})
}

func decode[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	if err := jsonDecode(r.Body, &req); err != nil {
		invalidRequest(w)
		return req, false
	}
	if err := validatorFn(req); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]string{"code": "validation_error", "message": "Invalid request parameters"}})
		return req, false
	}
	return req, true
}
