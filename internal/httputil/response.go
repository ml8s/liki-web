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

// RespondRawJSON writes a JSON body without the {data} envelope. Use it for
// endpoints whose success shape is a bare resource (arrays, provider acks).
func RespondRawJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Warn("respondRawJSON encode", "err", err)
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

// Semantic shortcuts keep one canonical message per status across packages.

func Unauthorized(w http.ResponseWriter) {
	RespondError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "请先登录")
}

func Forbidden(w http.ResponseWriter) {
	RespondError(w, http.StatusForbidden, ErrCodeForbidden, "无权访问")
}

func NotFound(w http.ResponseWriter) {
	RespondError(w, http.StatusNotFound, ErrCodeNotFound, "资源不存在")
}

func InvalidRequest(w http.ResponseWriter) {
	RespondError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "请求无效")
}

func InternalError(w http.ResponseWriter) {
	RespondError(w, http.StatusInternalServerError, ErrCodeInternal, "服务器内部错误")
}

// Decode parses a lenient JSON body (unknown fields allowed, no validation)
// and writes the canonical error envelope on failure. It is the shared
// replacement for per-package decode helpers.
func Decode[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
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
	return req, true
}

// DecodeStrict behaves like DecodeAndValidate but additionally rejects
// unknown fields. Use it for first-party request DTOs; never for
// third-party payloads such as AG-UI input.
func DecodeStrict[T validator](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondAppError(w, &AppError{Code: "too_large", Message: "Request body too large", HTTPStatus: http.StatusRequestEntityTooLarge})
			return req, false
		}
		RespondAppError(w, &AppError{Code: ErrCodeInvalidRequest, Message: "请求无效", HTTPStatus: http.StatusBadRequest})
		return req, false
	}
	if err := req.Validate(); err != nil {
		slog.Warn("validation failed", "err", err)
		RespondAppError(w, &AppError{Code: "validation_error", Message: "Invalid request parameters", HTTPStatus: http.StatusUnprocessableEntity})
		return req, false
	}
	return req, true
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
