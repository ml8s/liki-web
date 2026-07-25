package httputil

import "net/http"

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *AppError) Error() string { return e.Message }

const (
	ErrCodeInvalidRequest  = "invalid_request"
	ErrCodeUnauthorized    = "unauthorized"
	ErrCodeForbidden       = "forbidden"
	ErrCodeNotFound        = "not_found"
	ErrCodeInternal        = "internal_error"
)

var (
	ErrOrderNotFound = &AppError{Code: ErrCodeNotFound, Message: "订单不存在", HTTPStatus: http.StatusNotFound}
	ErrOrderNotPaid  = &AppError{Code: ErrCodeForbidden, Message: "订单未支付", HTTPStatus: http.StatusForbidden}
	ErrUnauthorized  = &AppError{Code: ErrCodeUnauthorized, Message: "请先登录或购买", HTTPStatus: http.StatusUnauthorized}
	ErrForbidden     = &AppError{Code: ErrCodeForbidden, Message: "无权访问", HTTPStatus: http.StatusForbidden}
	ErrInternal      = &AppError{Code: ErrCodeInternal, Message: "服务器内部错误", HTTPStatus: http.StatusInternalServerError}
)
