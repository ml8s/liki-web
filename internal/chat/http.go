package chat

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/birthprofile"
	"github.com/ml8s/liki-web/internal/httputil"
)

type RequireUserFunc func(http.HandlerFunc) http.HandlerFunc
type LimitFunc func(http.HandlerFunc) http.HandlerFunc

// Handler exposes chat HTTP routes.
type Handler struct {
	service     *Service
	requireUser RequireUserFunc
	limit       LimitFunc
}

func NewHandler(service *Service, requireUser RequireUserFunc, limit LimitFunc) *Handler {
	if requireUser == nil {
		requireUser = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	return &Handler{service: service, requireUser: requireUser, limit: limit}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/chat/sessions", h.requireUser(h.HandleSessions))
	mux.HandleFunc("POST /api/chat/sessions", h.requireUser(h.limit(h.HandleCreateSession)))
	mux.HandleFunc("GET /api/chat/sessions/{id}", h.requireUser(h.HandleSession))
	mux.HandleFunc("GET /api/chat/sessions/{id}/messages", h.requireUser(h.HandleMessages))
	mux.HandleFunc("POST /api/chat/sessions/{id}/ag-ui", h.requireUser(h.limit(h.HandleAGUIMessage)))
}

type createSessionRequest struct {
	Skill     string `json:"skill"`
	Title     string `json:"title"`
	ProfileID string `json:"profile_id"`
}

func (r createSessionRequest) Validate() error {
	if strings.TrimSpace(r.Skill) == "" {
		return errors.New("skill is required")
	}
	return nil
}

func (h *Handler) HandleSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.service.store.ListSessions(r.Context(), userID(r))
	if err != nil {
		httputil.InternalError(w)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, sessions)
}

func (h *Handler) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.Decode[createSessionRequest](w, r)
	if !ok {
		return
	}
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	session, err := h.service.CreateSession(r.Context(), user.ID, req.Skill, req.Title, req.ProfileID)
	if err != nil {
		if errors.Is(err, ErrSkillForbidden) {
			httputil.RespondError(w, http.StatusForbidden, "skill_forbidden", "当前权益不可使用该 skill")
			return
		}
		if errors.Is(err, birthprofile.ErrNotFound) {
			httputil.NotFound(w)
			return
		}
		slog.Error("chat: create session", "user_id", user.ID, "err", err)
		httputil.InternalError(w)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, session)
}

func (h *Handler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	sessionID := r.PathValue("id")
	if _, err := h.service.Session(r.Context(), user.ID, sessionID); err != nil {
		writeSessionError(w, err)
		return
	}
	messages, err := h.service.store.ListMessages(r.Context(), user.ID, sessionID)
	if err != nil {
		slog.Error("chat: list messages", "session_id", sessionID, "err", err)
		httputil.InternalError(w)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, messages)
}

func (h *Handler) HandleSession(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	session, err := h.service.Session(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		writeSessionError(w, err)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, session)
}

func userID(r *http.Request) string {
	user, _ := auth.UserFromContext(r.Context())
	return user.ID
}

func writeSessionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSkillForbidden):
		httputil.Forbidden(w)
	case errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrRunnerUnavailable):
		httputil.NotFound(w)
	default:
		slog.Error("chat: access session", "err", err)
		httputil.InternalError(w)
	}
}

func localeOrDefault(locale string) string {
	switch locale {
	case "en":
		return "en"
	case "zh-Hans":
		return "zh-Hans"
	default:
		return "zh-Hant"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
