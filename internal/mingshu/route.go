// RegisterRoutes mounts retired mingshu compatibility routes on the given mux.
package mingshu

import (
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux, store *Store, rateLimitFn func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("POST /api/mingshu/orders", HandleCreateOrder(store))
	mux.HandleFunc("GET /api/mingshu/orders/{id}/report", HandleGetReport(store))
}
