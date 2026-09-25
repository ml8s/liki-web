// RegisterRoutes mounts naming product routes on the given mux.
package naming

import (
	"net/http"

	"github.com/ml8s/liki-web/internal/agent"
)

func RegisterRoutes(mux *http.ServeMux, store *Store, a *agent.Agent, rateLimitFn func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("POST /api/naming/auth/login", HandleLogin(store))
	mux.HandleFunc("POST /api/naming/orders", HandleCreateOrder(store))
	mux.HandleFunc("POST /api/naming/orders/select", HandleOrderSelect(store))
	mux.HandleFunc("GET /api/naming/orders/{id}/status", HandleOrderStatus(store))
	mux.HandleFunc("GET /api/naming/orders/{id}/messages", HandleChatHistory(store))
	mux.HandleFunc("DELETE /api/naming/orders/{id}/messages", HandleDeleteChatHistory(store))
	mux.HandleFunc("POST /api/naming/agent/chat", rateLimitFn(HandleChat(a, store)))
	mux.HandleFunc("POST /api/naming/verification-codes/redeem", HandleRedeemCode(store))
}
