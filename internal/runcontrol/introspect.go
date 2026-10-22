// Package runcontrol owns the direct-connect control plane: session
// introspection for the edge (Caddy forward_auth, contract C1) and, in later
// phases, the run-completed callback receiver. It owns no entitlement policy
// and no agent execution.
package runcontrol

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/chat"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/product"
)

// SessionResolver resolves a session cookie value into the authenticated user.
// Implemented by *auth.Store.
type SessionResolver interface {
	GetSessionUser(ctx context.Context, token string) (auth.User, auth.AuthSession, error)
}

// EntitlementChecker reports whether a user holds a skill. Implemented by
// *entitlement.Store.
type EntitlementChecker interface {
	HasSkill(ctx context.Context, userID string, skill product.Skill) (bool, error)
}

// IntrospectConfig wires the edge introspection endpoint.
type IntrospectConfig struct {
	CookieName   string
	Sessions     SessionResolver
	Entitlements EntitlementChecker
	Logger       *slog.Logger
}

// NewIntrospectHandler returns the RFC 7662-shaped introspection endpoint used
// by Caddy forward_auth. Caddy forwards the original request (session cookie
// intact) and the original path in X-Forwarded-Uri (/rt/<profile>/ag-ui).
//
// Status contract (contract C1): unknown profile 403; invalid or expired
// session 401; missing entitlement 401; success 200 with the RFC 7662 shape.
func NewIntrospectHandler(cfg IntrospectConfig) http.HandlerFunc {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := profileFromForwardedURI(r.Header.Get("X-Forwarded-Uri"))
		if !ok {
			httputil.RespondError(w, http.StatusForbidden, "profile_not_allowed", "unknown expert profile")
			return
		}
		cookie, err := r.Cookie(cfg.CookieName)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			httputil.Unauthorized(w)
			return
		}
		user, session, err := cfg.Sessions.GetSessionUser(r.Context(), cookie.Value)
		if err != nil {
			httputil.Unauthorized(w)
			return
		}
		allowed, err := cfg.Entitlements.HasSkill(r.Context(), user.ID, product.SkillConsultation)
		if err != nil || !allowed {
			httputil.Unauthorized(w)
			return
		}
		logger.Info("run introspect", "user_id", user.ID, "profile", profile)
		// Caddy forward_auth copy_headers 从响应头注入受信身份（契约 C4）。
		w.Header().Set("Remote-User", user.ID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"active": true,
			"sub":    user.ID,
			"exp":    session.ExpiresAt.Unix(),
			"scope":  profile,
		})
	}
}

// profileFromForwardedURI extracts <profile> from /rt/<profile>/ag-ui and
// validates it against the deployed expert set.
func profileFromForwardedURI(uri string) (string, bool) {
	path := uri
	if index := strings.IndexByte(path, '?'); index >= 0 {
		path = path[:index]
	}
	const prefix = "/rt/"
	const suffix = "/ag-ui"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}
	profile := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	if !chat.ValidExpert(profile) {
		return "", false
	}
	return profile, true
}
