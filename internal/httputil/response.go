package httputil

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type validator interface {
	Validate() error
}

func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]any{"data": data}); err != nil {
		slog.Warn("respondJSON encode", "err", err)
	}
}

func RespondAppError(w http.ResponseWriter, appErr *AppError) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(appErr.HTTPStatus)
	if err := json.NewEncoder(w).Encode(map[string]any{"error": appErr}); err != nil {
		slog.Warn("respondAppError encode", "err", err)
	}
}

func RespondError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": msg}}); err != nil {
		slog.Warn("respondError encode", "err", err)
	}
}

func DecodeAndValidate[T validator](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondAppError(w, &AppError{Code: "too_large", Message: "Request body too large", HTTPStatus: http.StatusRequestEntityTooLarge})
			return req, false
		}
		RespondAppError(w, &AppError{Code: ErrCodeInvalidRequest, Message: "Invalid JSON body", HTTPStatus: http.StatusBadRequest})
		return req, false
	}
	if err := req.Validate(); err != nil {
		slog.Warn("validation failed", "err", err)
		RespondAppError(w, &AppError{Code: "validation_error", Message: "Invalid request parameters", HTTPStatus: http.StatusUnprocessableEntity})
		return req, false
	}
	return req, true
}
