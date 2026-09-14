package chat

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"liki-web/internal/agent"
	"liki-web/internal/auth"
	"liki-web/internal/profile"
	"liki-web/internal/sse"
)

type RequireUserFunc func(http.HandlerFunc) http.HandlerFunc

type LimitFunc func(http.HandlerFunc) http.HandlerFunc

type Handler struct {
	service     *Service
	requireUser RequireUserFunc
	limit       LimitFunc
}

func NewHandler(service *Service, requireUser RequireUserFunc, limit LimitFunc) *Handler {
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	return &Handler{service: service, requireUser: requireUser, limit: limit}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, limit LimitFunc) {
	if limit == nil {
		limit = h.limit
	}
	mux.HandleFunc("GET /api/chat/sessions", h.requireUser(h.HandleSessions))
	mux.HandleFunc("POST /api/chat/sessions", h.requireUser(limit(h.HandleCreateSession)))
	mux.HandleFunc("GET /api/chat/sessions/{id}", h.requireUser(h.HandleSession))
	mux.HandleFunc("GET /api/chat/sessions/{id}/messages", h.requireUser(h.HandleMessages))
	mux.HandleFunc("POST /api/chat/sessions/{id}/messages", h.requireUser(limit(h.HandleSendMessage)))
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

type sendMessageRequest struct {
	Content string `json:"content"`
}

func (r sendMessageRequest) Validate() error {
	if strings.TrimSpace(r.Content) == "" {
		return errors.New("content is required")
	}
	return nil
}

func (h *Handler) HandleSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.service.store.ListSessions(r.Context(), userID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	responseOK(w, sessions)
}

func (h *Handler) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[createSessionRequest](w, r)
	if !ok {
		return
	}
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	session, err := h.service.CreateSession(r.Context(), user.ID, req.Skill, req.Title, req.ProfileID)
	if err != nil {
		if errors.Is(err, ErrSkillForbidden) {
			writeError(w, http.StatusForbidden, "skill_forbidden", "当前权益不可使用该 skill")
			return
		}
		if errors.Is(err, profile.ErrNotFound) {
			notFound(w)
			return
		}
		slog.Error("chat: create session", "user_id", user.ID, "err", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	responseOK(w, session)
}

func (h *Handler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
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
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	responseOK(w, messages)
}

func (h *Handler) HandleSession(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	session, err := h.service.Session(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		writeSessionError(w, err)
		return
	}
	responseOK(w, session)
}

func (h *Handler) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[sendMessageRequest](w, r)
	if !ok {
		return
	}
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	sessionID := r.PathValue("id")

	// Validate the HTTP boundary before switching to an event response. Once
	// SSE has begun, transport and domain failures use the same error event.
	if _, err := h.service.Session(r.Context(), user.ID, sessionID); err != nil {
		writeSessionError(w, err)
		return
	}

	flusher, ok := sse.Begin(w)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器不支持流式响应")
		return
	}

	// A failed SSE write means the client disconnected. Cancel the derived
	// context so the Agent stops instead of continuing a generation nobody sees.
	streamCtx, cancelStream := context.WithCancel(r.Context())
	defer cancelStream()
	var writeErr error
	userMessage, assistantMessage, session, err := h.service.SendMessage(
		streamCtx, user.ID, sessionID, req.Content,
		func(ev agent.ChatEvent) {
			if writeErr != nil {
				return
			}
			if err := sse.Write(w, flusher, ev); err != nil {
				writeErr = err
				cancelStream()
				slog.Warn("chat: write sse event", "err", err)
			}
		},
	)
	if err != nil && writeErr == nil && !errors.Is(err, context.Canceled) {
		slog.Error("chat: send message", "session_id", sessionID, "err", err)
	}
	if err != nil {
		if writeErr == nil && !errors.Is(err, context.Canceled) {
			_ = sse.Write(w, flusher, agent.ChatEvent{Type: agent.EventError, Content: "服务暂时不可用，请稍后重试"})
		}
		return
	}
	if writeErr != nil {
		return
	}

	_ = sse.Write(w, flusher, agent.ChatEvent{
		Type: agent.EventDone,
		Data: map[string]any{
			"session":           session,
			"active_skill":      session.Skill,
			"user_message":      userMessage,
			"assistant_message": assistantMessage,
		},
	})
}

func unauthorized(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "unauthorized", "message": "请先登录"}})
}

func notFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]any{"error": map[string]string{"code": "not_found", "message": "会话不存在"}})
}

func forbidden(w http.ResponseWriter) {
	writeJSON(w, http.StatusForbidden, map[string]any{"error": map[string]string{"code": "skill_forbidden", "message": "当前权益不可使用该 skill"}})
}

func writeSessionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSkillForbidden):
		forbidden(w)
	case errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrRunnerUnavailable):
		notFound(w)
	default:
		slog.Error("chat: access session", "err", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
