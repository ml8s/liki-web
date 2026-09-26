package naming

import (
	"net/http"

	"github.com/ml8s/liki-web/internal/httputil"
)

// HandleLogin retains the historical route but email alone can never
// authenticate an order. Custom naming is retired and is now a Liki Chat
// scenario.
func HandleLogin(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httputil.RespondError(w, http.StatusGone, "product_retired", "起名订单登录已下线，请使用 Liki Chat")
	}
}

// HandleOrderSelect no longer exchanges an email and order ID for an order
// JWT. Paid historical chats remain reachable through their existing bounded
// order-access capability.
func HandleOrderSelect(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httputil.RespondError(w, http.StatusGone, "product_retired", "起名订单选择已下线，请使用 Liki Chat")
	}
}
