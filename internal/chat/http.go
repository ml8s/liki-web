package chat

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
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
	mux.HandleFunc("PATCH /api/chat/sessions/{id}", h.requireUser(h.limit(h.HandleRenameSession)))
	mux.HandleFunc("DELETE /api/chat/sessions/{id}", h.requireUser(h.limit(h.HandleDeleteSession)))
	mux.HandleFunc("GET /api/chat/sessions/{id}/messages", h.requireUser(h.HandleMessages))
	mux.HandleFunc("POST /api/chat/sessions/{id}/ag-ui", h.requireUser(h.limit(h.HandleAGUIMessage)))
}

type createSessionRequest struct {
	Skill     string               `json:"skill"`
	Intent    string               `json:"intent"`
	Title     string               `json:"title"`
	ProfileID string               `json:"profile_id"`
	Context   *ConsultationContext `json:"context,omitempty"`
}

func (r createSessionRequest) Validate() error {
	if strings.TrimSpace(r.Skill) == "" {
		return errors.New("skill is required")
	}
	if intent := strings.TrimSpace(r.Intent); intent != "" {
		if !validConsultationIntent(intent) {
			return errors.New("intent is invalid")
		}
	}
	if strings.TrimSpace(r.Intent) == "" && r.Context == nil {
		return errors.New("intent is required")
	}
	// Consultation validation is handled after strict transport decoding so the
	// endpoint can return its domain-specific invalid_consultation_context code.
	return nil
}

// resolvedIntent derives the canonical intent from the server-owned vocabulary.
func (r createSessionRequest) resolvedIntent() (string, error) {
	intent := strings.TrimSpace(r.Intent)
	if r.Context != nil {
		intent = r.Context.Intent
	}
	if !validConsultationIntent(intent) {
		return "", errors.New("intent is invalid")
	}
	return intent, nil
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
	req, ok := httputil.DecodeStrict[createSessionRequest](w, r)
	if !ok {
		return
	}
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	if err := req.Context.Validate(); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid_consultation_context", err.Error())
		return
	}
	intent, err := req.resolvedIntent()
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	var consultationContext json.RawMessage
	if req.Context != nil {
		raw, err := json.Marshal(req.Context)
		if err != nil {
			slog.ErrorContext(r.Context(), "chat: encode consultation context", "user_id", user.ID, "err", err)
			httputil.InternalError(w)
			return
		}
		consultationContext = raw
	}
	session, err := h.service.CreateSessionWithContext(
		r.Context(), user.ID, req.Skill, intent, req.Title, req.ProfileID, consultationContext,
	)
	if err != nil {
		if errors.Is(err, ErrIntentInvalid) {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "intent is invalid")
			return
		}
		if errors.Is(err, ErrSkillForbidden) {
			httputil.RespondError(w, http.StatusForbidden, "skill_forbidden", "当前权益不可使用该 skill")
			return
		}
		if errors.Is(err, birthprofile.ErrNotFound) {
			httputil.NotFound(w)
			return
		}
		slog.ErrorContext(r.Context(), "chat: create session", "user_id", user.ID, "err", err)
		httputil.InternalError(w)
		return
	}
	httputil.RespondJSON(w, http.StatusCreated, session)
}

func (h *Handler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	sessionID := r.PathValue("id")
	if _, err := h.service.Session(r.Context(), user.ID, sessionID); err != nil {
		writeSessionError(r.Context(), w, err)
		return
	}
	limit := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil {
			limit = parsed
		}
	}
	messages, nextCursor, err := h.service.store.ListMessages(
		r.Context(), user.ID, sessionID, strings.TrimSpace(r.URL.Query().Get("before")), limit,
	)
	if err != nil {
		if errors.Is(err, ErrInvalidMessageCursor) {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "invalid message cursor")
			return
		}
		slog.ErrorContext(r.Context(), "chat: list messages", "session_id", sessionID, "err", err)
		httputil.InternalError(w)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{
		"messages":    messages,
		"next_cursor": nextCursor,
	})
}

func (h *Handler) HandleSession(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	session, err := h.service.Session(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		writeSessionError(r.Context(), w, err)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, session)
}

func userID(r *http.Request) string {
	user, _ := auth.UserFromContext(r.Context())
	return user.ID
}

const maxSessionTitleRunes = 60

type renameSessionRequest struct {
	Title string `json:"title"`
}

func (r renameSessionRequest) Validate() error {
	title := strings.TrimSpace(r.Title)
	if title == "" || len([]rune(title)) > maxSessionTitleRunes {
		return errors.New("title must contain 1-60 characters")
	}
	return nil
}

func (h *Handler) HandleRenameSession(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	req, ok := httputil.DecodeStrict[renameSessionRequest](w, r)
	if !ok {
		return
	}
	title := strings.TrimSpace(req.Title)
	session, err := h.service.RenameSession(r.Context(), user.ID, r.PathValue("id"), title)
	if err != nil {
		writeSessionError(r.Context(), w, err)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, session)
}

func (h *Handler) HandleDeleteSession(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	if err := h.service.DeleteSession(r.Context(), user.ID, r.PathValue("id")); err != nil {
		writeSessionError(r.Context(), w, err)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func writeSessionError(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSkillForbidden):
		httputil.Forbidden(w)
	case errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrRunnerUnavailable):
		httputil.NotFound(w)
	default:
		slog.ErrorContext(ctx, "chat: access session", "err", err)
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
