package account

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"liki-web/internal/auth"
	"liki-web/internal/chat"
	"liki-web/internal/profile"
)

type RequireUserFunc func(http.HandlerFunc) http.HandlerFunc

type Handler struct {
	profiles    *profile.Store
	chat        *chat.Store
	requireUser RequireUserFunc
}

func NewHandler(profiles *profile.Store, chatStore *chat.Store, requireUser RequireUserFunc) *Handler {
	return &Handler{profiles: profiles, chat: chatStore, requireUser: requireUser}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/account/erasure", h.requireUser(h.HandleErasure))
}

// HandleErasure 合规强删（个保法删除权）：匿名化该用户全部档案（清除出生信息等
// PII 并软删），并清除会话中内嵌 PII 的 system_profile 消息。会话结构保留。
func (h *Handler) HandleErasure(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "unauthorized", "message": "请先登录"}})
		return
	}
	profilesCleared, err := h.profiles.AnonymizeUser(r.Context(), user.ID)
	if err != nil {
		slog.Error("account: erase profiles", "user_id", user.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"code": "internal_error", "message": "服务器内部错误"}})
		return
	}
	messagesCleared, err := h.chat.AnonymizeUserMessages(r.Context(), user.ID)
	if err != nil {
		slog.Error("account: erase chat PII", "user_id", user.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"code": "internal_error", "message": "服务器内部错误"}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"status": "erased", "profiles_cleared": profilesCleared, "messages_cleared": messagesCleared},
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
