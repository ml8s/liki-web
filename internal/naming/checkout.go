package naming

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"liki-web/internal/httputil"
	"liki-web/internal/product"

	"github.com/google/uuid"
)

// HandleCreateOrder creates a naming order.
func HandleCreateOrder(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}

		locale := httputil.LocaleFromAccept(r)
		orderID := uuid.New().String()
		currency := product.DetectCurrency(r.Header.Get("CF-IPCountry"), r.Header.Get("X-Geo-Country"))
		amount, err := product.AmountMinor(product.ProductNaming, currency)
		if err != nil {
			slog.Error("naming: resolve price", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "创建订单失败")
			return
		}

		if err := store.CreateOrder(r.Context(), orderID, string(product.ProductNaming), amount, string(currency), req.Email, locale); err != nil {
			slog.Error("naming: create order", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "创建订单失败")
			return
		}

		httputil.RespondJSON(w, http.StatusOK, map[string]string{"order_id": orderID})
	}
}

// HandleRedeemCode redeems a verification code to create a paid order.
func HandleRedeemCode(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Code  string `json:"code"`
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}
		if req.Code == "" || req.Email == "" {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "code 和 email 不能为空")
			return
		}

		lang := httputil.LocaleFromAccept(r)
		orderID := uuid.New().String()
		params := RedeemCodeParams{
			Code:          req.Code,
			Email:         req.Email,
			OrderID:       orderID,
			Product:       "naming",
			Currency:      "CNY",
			Amount:        int64(990),
			ChatExpiresAt: "2099-12-31",
			Locale:        lang,
		}

		if err := store.RedeemCode(r.Context(), params); err != nil {
			slog.Warn("naming: redeem", "err", err)
			httputil.RespondError(w, http.StatusBadRequest, "redeem_failed", err.Error())
			return
		}

		if httputil.SetJWTCookieOrRespondErr(w, req.Email, orderID) != nil {
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"order_id": orderID})
	}
}

func HandleOrderStatus(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID := r.PathValue("id")
		if orderID == "" {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "missing order id")
			return
		}
		order, err := store.GetOrder(r.Context(), orderID)
		if err != nil {
			httputil.RespondAppError(w, httputil.ErrOrderNotFound)
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"status": order.Status})
	}
}
