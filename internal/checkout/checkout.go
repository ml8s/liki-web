package checkout

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"liki-web/internal/httputil"
	"liki-web/internal/payment"
	"liki-web/internal/product"
)

// Config holds the per-product configuration for checkout handlers.
type Config struct {
	Product      product.Product
	ProductFn    func(*product.CheckoutOrder) product.Product
	ReturnPath   string
	RedirectPath string
	LogPrefix    string
}

func (c Config) resolveProduct(o *product.CheckoutOrder) product.Product {
	if c.ProductFn != nil {
		return c.ProductFn(o)
	}
	return c.Product
}

type checkoutBody struct {
	OrderID  string `json:"order_id"`
	Email    string `json:"email"`
	Provider string `json:"provider"`
}

// MakeCheckout creates a payment checkout handler.
func MakeCheckout(store product.OrderStore, svc *payment.Service, cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req checkoutBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				httputil.RespondError(w, http.StatusRequestEntityTooLarge, "too_large", "Request body too large")
				return
			}
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}
		if req.OrderID == "" || req.Email == "" {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "missing order_id or email")
			return
		}

		order, err := store.GetCheckoutOrder(r.Context(), req.OrderID)
		if err != nil {
			if appErr, ok := err.(*httputil.AppError); ok {
				httputil.RespondAppError(w, appErr)
				return
			}
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "查询订单失败")
			return
		}
		if order.Status != "pending" {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "order already processed")
			return
		}

		if err := store.UpdateEmail(r.Context(), req.OrderID, req.Email); err != nil {
			slog.Error(cfg.LogPrefix+" update email", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "更新邮箱失败")
			return
		}

		provider := req.Provider
		if provider == "" {
			provider = "dodo"
		}
		if err := store.UpdateProvider(r.Context(), req.OrderID, provider); err != nil {
			slog.Error(cfg.LogPrefix+" update provider", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "更新支付方式失败")
			return
		}

		p, err := svc.Provider(provider)
		if err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		returnURL := svc.ReturnURL() + cfg.ReturnPath + req.OrderID
		result, err := p.CreateCheckout(r.Context(), cfg.resolveProduct(order), order.Amount, req.OrderID, req.Email, returnURL)
		if err != nil {
			slog.Error(cfg.LogPrefix+" create checkout", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "checkout_error", "创建订单失败，请稍后重试")
			return
		}

		httputil.RespondJSON(w, http.StatusOK, result)
	}
}

// MakePaymentReturn processes payment gateway callback redirect.
func MakePaymentReturn(store product.OrderStore, cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID := r.PathValue("id")
		status := r.URL.Query().Get("status")
		token := r.URL.Query().Get("t")

		if orderID == "" || status != "succeeded" || !httputil.VerifyReturnToken(string(cfg.Product), orderID, token) {
			slog.Warn(cfg.LogPrefix+" invalid return callback", "orderID", orderID)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		newPayment, email, err := store.MarkPaidViaCallback(r.Context(), orderID)
		if err != nil {
			slog.Error(cfg.LogPrefix+" mark paid", "err", err)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		_ = newPayment

		if email != "" {
			if err := httputil.SetJWTCookie(w, email, orderID); err != nil {
				slog.Error(cfg.LogPrefix+" auth: sign jwt", "err", err)
			}
		}

		locale := httputil.LocaleFromAccept(r)
		http.Redirect(w, r, "/"+locale+cfg.RedirectPath+"?order_id="+orderID, http.StatusFound)
	}
}

// MakeOrderStatus returns the current status of an order.
func MakeOrderStatus(store product.OrderStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID := r.PathValue("id")
		if orderID == "" {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "missing order id")
			return
		}

		order, err := store.GetCheckoutOrder(r.Context(), orderID)
		if err != nil {
			if appErr, ok := err.(*httputil.AppError); ok {
				httputil.RespondAppError(w, appErr)
				return
			}
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "查询订单失败")
			return
		}

		httputil.RespondJSON(w, http.StatusOK, map[string]string{"status": order.Status})
	}
}
