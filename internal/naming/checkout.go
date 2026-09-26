package naming

import (
	"net/http"

	"github.com/ml8s/liki-web/internal/httputil"
)

// HandleCreateOrder retains the historical route but no longer creates orders:
// custom naming is retired and is now a Liki Chat scenario.
func HandleCreateOrder(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httputil.RespondError(w, http.StatusGone, "product_retired", "起名产品已下线，请使用 Liki Chat")
	}
}

// HandleRedeemCode no longer redeems retired-product activation codes.
func HandleRedeemCode(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httputil.RespondError(w, http.StatusGone, "product_retired", "起名产品已下线，请使用 Liki Chat")
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
