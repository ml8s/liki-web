package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/metrics"
)

// handlePageView records an aggregate, cookieless page-view counter. Paths are
// intentionally not labels: a public beacon must not create unbounded metric
// cardinality.
type pageViewRequest struct {
	Path string `json:"path"`
}

func (r pageViewRequest) Validate() error {
	if r.Path == "" || !strings.HasPrefix(r.Path, "/") {
		return errors.New("path must be an absolute path")
	}
	return nil
}

func handlePageView() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := httputil.DecodeStrict[pageViewRequest](w, r); !ok {
			return
		}
		metrics.Default.Inc(metrics.MetricAnalyticsPageview, nil)
		w.WriteHeader(http.StatusNoContent)
	}
}
