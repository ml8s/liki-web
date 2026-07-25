
// RegisterRoutes mounts naming product routes on the given mux.
package naming

import (
	"net/http"

	"liki-web/internal/agent"
	"liki-web/internal/checkout"
	"liki-web/internal/payment"
	"liki-web/internal/product"
)

func RegisterRoutes(mux *http.ServeMux, store *Store, svc *payment.Service, a *agent.Agent, onPaid product.OnPaid, rateLimitFn func(http.HandlerFunc) http.HandlerFunc) {
	cfg := checkout.Config{
		Product:      product.ProductNaming,
		ProductFn:    namingProductFromOrder,
		ReturnPath:   "/api/naming/payments/return/",
		RedirectPath: "/naming/chat",
		LogPrefix:    "naming:",
	}

	mux.HandleFunc("POST /api/naming/auth/login", HandleLogin(store))
	mux.HandleFunc("POST /api/naming/orders", HandleCreateOrder(store))
	mux.HandleFunc("POST /api/naming/orders/select", HandleOrderSelect(store))
	mux.HandleFunc("GET /api/naming/orders/{id}/status", checkout.MakeOrderStatus(store))
	mux.HandleFunc("POST /api/naming/orders/{id}/retry", HandleRetryOrder(store, svc))
	mux.HandleFunc("GET /api/naming/orders/{id}/messages", HandleChatHistory(store))
	mux.HandleFunc("DELETE /api/naming/orders/{id}/messages", HandleDeleteChatHistory(store))
	mux.HandleFunc("POST /api/naming/agent/chat", rateLimitFn(HandleChat(a, store)))
	mux.HandleFunc("POST /api/naming/checkout", rateLimitFn(checkout.MakeCheckout(store, svc, cfg)))
	mux.HandleFunc("POST /api/naming/webhook", payment.HandleWebhook(store, svc, "naming", onPaid, nil, nil))
	mux.HandleFunc("GET /api/naming/payments/return/{id}", checkout.MakePaymentReturn(store, cfg))
	mux.HandleFunc("POST /api/naming/verification-codes/redeem", HandleRedeemCode(store))
}

func namingProductFromOrder(o *product.CheckoutOrder) product.Product {
	if o.Product == "mingshu" {
		return product.ProductMingShu
	}
	return product.ProductNaming
}
