// RegisterRoutes mounts mingshu product routes on the given mux.
package mingshu

import (
	"net/http"

	"liki-web/internal/checkout"
	"liki-web/internal/payment"
	"liki-web/internal/product"
)

func webhookHandler(store *Store, svc *payment.Service, logPrefix string, gen *Generator, onPaid product.OnPaid, onReady product.OnReportReady) http.HandlerFunc {
	var p payment.WebhookPipeline
	if gen != nil {
		p = gen
	}
	return payment.HandleWebhook(store, svc, logPrefix, onPaid, p, onReady)
}

func RegisterRoutes(mux *http.ServeMux, store *Store, svc *payment.Service, pipeline *Generator, onPaid product.OnPaid, onReady product.OnReportReady) {
	cfg := checkout.Config{
		Product:      product.ProductMingShu,
		ReturnPath:   "/api/mingshu/payments/return/",
		RedirectPath: "/mingshu/report",
		LogPrefix:    "mingshu:",
	}

	mux.HandleFunc("POST /api/mingshu/orders", HandleCreateOrder(store))
	mux.HandleFunc("GET /api/mingshu/orders/{id}/status", checkout.MakeOrderStatus(store))
	mux.HandleFunc("POST /api/mingshu/checkout", checkout.MakeCheckout(store, svc, cfg))
	mux.HandleFunc("GET /api/mingshu/payments/return/{id}", checkout.MakePaymentReturn(store, cfg))
	mux.HandleFunc("POST /api/mingshu/webhook", webhookHandler(store, svc, "mingshu", pipeline, onPaid, onReady))
	mux.HandleFunc("GET /api/mingshu/orders/{id}/report", HandleGetReport(store))
}
