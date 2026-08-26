package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"liki-web/internal/httputil"
)

type envelope struct {
	Data  any                `json:"data,omitempty"`
	Error *httputil.AppError `json:"error,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(envelope{Data: data}); err != nil {
		slog.Warn("respondJSON encode", "err", err)
	}
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	httputil.RespondAppError(w, &httputil.AppError{Code: code, Message: message, HTTPStatus: status})
}
