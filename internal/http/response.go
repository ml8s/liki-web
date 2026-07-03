package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// envelope is the unified response wrapper.
type envelope struct {
	Data  any       `json:"data,omitempty"`
	Error *AppError `json:"error,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(envelope{Data: data}); err != nil {
		slog.Warn("respondJSON encode", "err", err)
	}
}

// respondError writes a JSON error response with the given HTTP status, code, and message.
func respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(envelope{Error: &AppError{Code: code, Message: message}}); err != nil {
		slog.Warn("respondError encode", "err", err)
	}
}

// respondAppError writes a JSON error response using the HTTP status, code, and
// message from the given AppError.
func respondAppError(w http.ResponseWriter, appErr *AppError) {
	respondError(w, appErr.HTTPStatus, appErr.Code, appErr.Message)
}

func respondStatus(w http.ResponseWriter, status int, s string) {
	respondJSON(w, status, map[string]string{"status": s})
}
