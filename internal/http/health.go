package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/ml8s/liki-web/internal/httputil"
)

type ReadinessChecker func(context.Context) error

func handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func handleReady(check ReadinessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if check == nil {
			httputil.RespondJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "unready", "error": "readiness_not_configured",
			})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := check(ctx); err != nil {
			slog.ErrorContext(ctx, "readiness check failed", "err", err)
			httputil.RespondJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "unready", "error": "readiness_check_failed",
			})
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

// handleVersion returns the build version and build time of the running binary.
func handleVersion(buildTime, version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httputil.RespondJSON(w, http.StatusOK, map[string]string{
			"build_time": buildTime,
			"version":    version,
		})
	}
}
