package http

import (
	"net/http"

	"github.com/ml8s/liki-web/internal/metrics"
)

// handlePageView records an aggregate, cookieless page-view counter. Paths are
// intentionally not labels: a public beacon must not create unbounded metric
// cardinality.
func handlePageView() http.HandlerFunc {
	type pageViewRequest struct {
		Path string `json:"path"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := decodeJSON[pageViewRequest](w, r); !ok {
			return
		}
		metrics.Default.Inc("analytics_pageview_total", nil)
		w.WriteHeader(http.StatusNoContent)
	}
}
