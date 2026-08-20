package naming

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"liki-web/internal/httputil"
	"liki-web/internal/payment"
	"liki-web/internal/product"

	"github.com/google/uuid"
)

// HandleCreateOrder creates a naming order.
func HandleCreateOrder(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Product  string `json:"product"`
			Currency string `json:"currency"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}

		locale := httputil.LocaleFromAccept(r)
		orderID := uuid.New().String()
		amount, currency := namingAmount(req.Currency)

		if err := store.CreateOrder(r.Context(), orderID, req.Product, amount, currency, "", locale); err != nil {
			slog.Error("naming: create order", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "创建订单失败")
			return
		}

		httputil.RespondJSON(w, http.StatusOK, map[string]string{"order_id": orderID})
	}
}

// HandleRetryOrder creates a new checkout for an existing (paid) order.
func HandleRetryOrder(store *Store, svc *payment.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID := r.PathValue("id")
		if orderID == "" {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "missing order id")
			return
		}

		o, err := store.GetOrder(r.Context(), orderID)
		if err != nil {
			httputil.RespondAppError(w, httputil.ErrOrderNotFound)
			return
		}

		provider := o.Provider
		if provider == "" {
			provider = "dodo"
		}
		p, err := svc.Provider(provider)
		if err != nil {
			httputil.RespondAppError(w, httputil.ErrInternal)
			return
		}

		returnURL := svc.ReturnURL() + "/api/naming/payments/return/" + orderID
		result, err := p.CreateCheckout(r.Context(), namingProduct(o.Product), o.Amount, orderID, o.Email, returnURL)
		if err != nil {
			slog.Error("naming: retry checkout", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "checkout_error", "创建订单失败")
			return
		}

		httputil.RespondJSON(w, http.StatusOK, result)
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
Amount: 990,
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

func namingAmount(currency string) (int, string) {
	if currency == "USD" {
		return 990, "USD"
	}
	return 990, "CNY"
}

func namingProduct(s string) product.Product {
	switch s {
	case "lifebook":
		return product.ProductMingShu
	default:
		return product.ProductNaming
	}
}
