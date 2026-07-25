package naming

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"liki-web/internal/agent"
	"liki-web/internal/httputil"
	"liki-web/internal/llm"
)

func HandleChat(chat *agent.Agent, store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, orderID, ok := httputil.JWTAuth(r)
		if !ok {
			httputil.RespondAppError(w, httputil.ErrUnauthorized)
			return
		}
		o, err := store.GetOrder(r.Context(), orderID)
		if err != nil {
			httputil.RespondAppError(w, httputil.ErrOrderNotFound)
			return
		}
		if o.Status != "paid" {
			httputil.RespondAppError(w, httputil.ErrOrderNotPaid)
			return
		}
		if expiresAt, err := time.Parse(time.DateTime, o.ChatExpiresAt); err == nil && time.Now().After(expiresAt) {
			httputil.RespondAppError(w, &httputil.AppError{
				Code: "expired", Message: "聊天已过期",
				HTTPStatus: http.StatusForbidden,
			})
			return
		}
		req, ok := httputil.DecodeAndValidate[chatRequest](w, r)
		if !ok {
			return
		}
		locale := "zh-Hans"
		if req.Lang == "en" {
			locale = "en"
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			httputil.RespondAppError(w, httputil.ErrInternal)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		history, err := store.LoadChatHistory(r.Context(), orderID)
		if err != nil {
			slog.Error("naming chat: load history", "err", err)
			writeSSE(w, flusher, agent.ChatEvent{Type: "error", Content: "服务暂时不可用，请稍后重试"}) //nolint:errcheck
			return
		}
		var messages []llm.Message
		for _, m := range history {
			messages = append(messages, llm.Message{
				Role:    llm.Role(m.Role),
				Content: m.Content,
			})
		}
		messages = append(messages, llm.Message{Role: llm.RoleUser, Content: req.Message})

		store.CreateChatMessage(r.Context(), orderID, string(llm.RoleUser), req.Message) //nolint:errcheck

		chatCtx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
		defer cancel()

		result, err := chat.NamingChat(chatCtx, locale, messages, func(ev agent.ChatEvent) {
			if err := writeSSE(w, flusher, ev); err != nil {
				slog.Warn("sse write", "err", err)
			}
		})
		if err != nil {
			slog.Error("naming chat", "err", err)
			writeSSE(w, flusher, agent.ChatEvent{Type: "error", Content: "服务暂时不可用，请稍后重试"}) //nolint:errcheck
			return
		}
		// messages passed to NamingChat: history + current user (no system).
		// NamingChat prepends 1 system message, so prefix to skip = 1 + len(messages).
		inputLen := 1 + len(messages)
		for _, msg := range result[inputLen:] {
			if msg.Role == llm.RoleAssistant && msg.Content != "" {
				store.CreateChatMessage(r.Context(), orderID, string(msg.Role), msg.Content) //nolint:errcheck
			}
		}
	}
}

func HandleChatHistory(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, orderID, ok := httputil.JWTAuth(r)
		if !ok {
			httputil.RespondAppError(w, httputil.ErrUnauthorized)
			return
		}
		if r.PathValue("id") != orderID {
			httputil.RespondAppError(w, httputil.ErrForbidden)
			return
		}
		o, err := store.GetOrder(r.Context(), orderID)
		if err != nil {
			httputil.RespondAppError(w, httputil.ErrOrderNotFound)
			return
		}
		if o.Status != "paid" {
			httputil.RespondAppError(w, httputil.ErrOrderNotPaid)
			return
		}
		msgs, err := store.LoadChatHistory(r.Context(), orderID)
		if err != nil {
			slog.Error("chatHistory", "err", err)
			httputil.RespondAppError(w, httputil.ErrInternal)
			return
		}
		if msgs == nil {
			msgs = []ChatMessage{}
		}
		httputil.RespondJSON(w, http.StatusOK, msgs)
	}
}

func HandleDeleteChatHistory(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, orderID, ok := httputil.JWTAuth(r)
		if !ok {
			httputil.RespondAppError(w, httputil.ErrUnauthorized)
			return
		}
		if r.PathValue("id") != orderID {
			httputil.RespondAppError(w, httputil.ErrForbidden)
			return
		}
		o, err := store.GetOrder(r.Context(), orderID)
		if err != nil {
			httputil.RespondAppError(w, httputil.ErrOrderNotFound)
			return
		}
		if o.Status != "paid" {
			httputil.RespondAppError(w, httputil.ErrOrderNotPaid)
			return
		}
		if err := store.DeleteChatHistory(r.Context(), orderID); err != nil {
			slog.Error("deleteChatHistory", "err", err)
			httputil.RespondAppError(w, httputil.ErrInternal)
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
