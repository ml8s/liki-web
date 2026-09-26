// RegisterRoutes mounts retired hepan compatibility routes on the given mux.
package hepan

import (
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux, store *Store, rateLimitFn func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("POST /api/hepan/orders", HandleCreateOrder(store))
	mux.HandleFunc("GET /api/hepan/orders/{id}/report", HandleGetReport(store))
}
