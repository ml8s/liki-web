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
		slog.Warn("respondJSON encode", "err", err) // 无 ctx 可用：响应写失败告警，签名不为日志扩参
	}
}

// RespondRawJSON writes a JSON body without the {data} envelope. Use it for
// endpoints whose success shape is a bare resource (arrays, provider acks).
func RespondRawJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Warn("respondRawJSON encode", "err", err) // 无 ctx 可用：响应写失败告警，签名不为日志扩参
	}
}

func RespondAppError(w http.ResponseWriter, appErr *AppError) {
	respondProblem(w, appErr.HTTPStatus, appErr.Code, appErr.Message)
}

func RespondError(w http.ResponseWriter, status int, code, msg string) {
	respondProblem(w, status, code, msg)
}

// respondProblem writes an RFC 9457 Problem Details document
// (application/problem+json): title is the human-readable status summary,
// detail the human explanation for this occurrence, and the machine-readable
// code rides the `code` extension member (the runtime's problem documents do
// the same). `type` keeps carrying the code for existing clients.
func respondProblem(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	body := map[string]any{
		"type":   code,
		"title":  http.StatusText(status),
		"status": status,
		"code":   code,
		"detail": detail,
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Warn("respondProblem encode", "err", err) // 无 ctx 可用：响应写失败告警，签名不为日志扩参
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
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			RespondAppError(w, &AppError{Code: "too_large", Message: "Request body too large", HTTPStatus: http.StatusRequestEntityTooLarge})
			return req, false
		}
		RespondAppError(w, &AppError{Code: ErrCodeInvalidRequest, Message: "请求无效", HTTPStatus: http.StatusBadRequest})
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
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			RespondAppError(w, &AppError{Code: "too_large", Message: "Request body too large", HTTPStatus: http.StatusRequestEntityTooLarge})
			return req, false
		}
		RespondAppError(w, &AppError{Code: ErrCodeInvalidRequest, Message: "请求无效", HTTPStatus: http.StatusBadRequest})
		return req, false
	}
	if err := req.Validate(); err != nil {
		slog.WarnContext(r.Context(), "validation failed", "err", err)
		RespondAppError(w, &AppError{Code: "validation_error", Message: "Invalid request parameters", HTTPStatus: http.StatusUnprocessableEntity})
		return req, false
	}
	return req, true
}

func DecodeAndValidate[T validator](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			RespondAppError(w, &AppError{Code: "too_large", Message: "Request body too large", HTTPStatus: http.StatusRequestEntityTooLarge})
			return req, false
		}
		RespondAppError(w, &AppError{Code: ErrCodeInvalidRequest, Message: "请求无效", HTTPStatus: http.StatusBadRequest})
		return req, false
	}
	if err := req.Validate(); err != nil {
		slog.WarnContext(r.Context(), "validation failed", "err", err)
		RespondAppError(w, &AppError{Code: "validation_error", Message: "Invalid request parameters", HTTPStatus: http.StatusUnprocessableEntity})
		return req, false
	}
	return req, true
}
