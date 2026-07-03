package http

import "net/http"

// AppError represents a structured API error with a machine-readable code
// and a human-readable message. HTTPStatus is used internally by the handler
// and is not serialized in the JSON response body.
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *AppError) Error() string { return e.Message }

// Predefined error codes.
const (
	ErrCodeInvalidRequest  = "invalid_request"
	ErrCodeValidationError = "validation_error"
	ErrCodeUnauthorized    = "unauthorized"
	ErrCodeForbidden       = "forbidden"
	ErrCodeNotFound        = "not_found"
	ErrCodeExpired         = "expired"
	ErrCodeTooLarge        = "too_large"
	ErrCodeInternal        = "internal_error"
	ErrCodeCheckoutError   = "checkout_error"
	ErrCodeInvalidPath     = "invalid_path"
	ErrCodeReadError       = "read_error"
	ErrCodeBadWebhook      = "bad_webhook"
	ErrCodeWebhookError    = "webhook_error"
)

// Common application errors. Handlers may define additional ones inline.
var (
	ErrInvalidJSON     = &AppError{Code: ErrCodeInvalidRequest, Message: "Invalid JSON", HTTPStatus: http.StatusBadRequest}
	ErrOrderNotFound   = &AppError{Code: ErrCodeNotFound, Message: "订单不存在", HTTPStatus: http.StatusNotFound}
	ErrOrderNotPaid    = &AppError{Code: ErrCodeForbidden, Message: "订单未支付", HTTPStatus: http.StatusForbidden}
	ErrUnauthorized    = &AppError{Code: ErrCodeUnauthorized, Message: "请先登录或购买", HTTPStatus: http.StatusUnauthorized}
	ErrInternal        = &AppError{Code: ErrCodeInternal, Message: "服务器内部错误", HTTPStatus: http.StatusInternalServerError}
	ErrMissingOrderID  = &AppError{Code: ErrCodeInvalidPath, Message: "missing order id", HTTPStatus: http.StatusBadRequest}
)
