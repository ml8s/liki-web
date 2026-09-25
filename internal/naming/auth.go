package naming

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/ml8s/liki-web/internal/httputil"
)

func HandleLogin(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := httputil.DecodeAndValidate[loginRequest](w, r)
		if !ok {
			return
		}
		orders, err := store.FindActiveOrdersByEmail(r.Context(), req.Email)
		if err != nil {
			slog.Error("auth: find orders by email", "err", err)
			httputil.RespondAppError(w, &httputil.AppError{
				Code: httputil.ErrCodeInternal, Message: "查询失败，请稍后重试",
				HTTPStatus: http.StatusInternalServerError,
			})
			return
		}
		if len(orders) == 0 {
			httputil.RespondAppError(w, &httputil.AppError{
				Code: httputil.ErrCodeNotFound, Message: "未找到有效订单，请先购买",
				HTTPStatus: http.StatusUnauthorized,
			})
			return
		}
		if len(orders) == 1 {
			o := orders[0]
			if httputil.SetJWTCookieOrRespondErr(w, o.Email, o.OrderID) != nil {
				return
			}
			httputil.RespondJSON(w, http.StatusOK, LoginResponse{OrderID: o.OrderID})
			return
		}
		summaries := make([]LoginOrderSummary, len(orders))
		for i, o := range orders {
			datePart := o.CreatedAt
			if len(datePart) >= 10 {
				datePart = datePart[5:10] // "MM-DD"
				datePart = strings.Replace(datePart, "-", "月", 1) + "日"
			}
			summaries[i] = LoginOrderSummary{
				OrderID:   o.OrderID,
				Summary:   "起名 · " + datePart + "购买",
				ExpiresAt: o.ChatExpiresAt,
			}
		}
		httputil.RespondJSON(w, http.StatusOK, LoginResponse{Orders: summaries})
	}
}

func HandleOrderSelect(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := httputil.DecodeAndValidate[selectOrderRequest](w, r)
		if !ok {
			return
		}
		o, err := store.GetOrder(r.Context(), req.OrderID)
		if err != nil {
			httputil.RespondAppError(w, httputil.ErrOrderNotFound)
			return
		}
		if o.Email != req.Email {
			httputil.RespondAppError(w, &httputil.AppError{
				Code: httputil.ErrCodeForbidden, Message: "无权访问此订单",
				HTTPStatus: http.StatusForbidden,
			})
			return
		}
		if httputil.SetJWTCookieOrRespondErr(w, o.Email, o.OrderID) != nil {
			return
		}
		httputil.RespondJSON(w, http.StatusOK, LoginResponse{OrderID: o.OrderID})
	}
}
