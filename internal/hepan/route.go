
// RegisterRoutes mounts hepan product routes on the given mux.
package hepan

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
		Product:      ProductHepan,
		ReturnPath:   "/api/hepan/payments/return/",
		RedirectPath: "/hepan/report",
		LogPrefix:    "hepan:",
	}

	mux.HandleFunc("POST /api/hepan/orders", HandleCreateOrder(store))
	mux.HandleFunc("GET /api/hepan/orders/{id}/status", checkout.MakeOrderStatus(store))
	mux.HandleFunc("POST /api/hepan/checkout", checkout.MakeCheckout(store, svc, cfg))
	mux.HandleFunc("GET /api/hepan/payments/return/{id}", checkout.MakePaymentReturn(store, cfg))
	mux.HandleFunc("POST /api/hepan/webhook", webhookHandler(store, svc, "hepan", pipeline, onPaid, onReady))
	mux.HandleFunc("GET /api/hepan/orders/{id}/report", HandleGetReport(store))
}
