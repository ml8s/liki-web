package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type ReadinessChecker func(context.Context) error

func handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func handleReady(check ReadinessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if check == nil {
			respondJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "unready", "error": "readiness_not_configured",
			})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := check(ctx); err != nil {
			slog.Error("readiness check failed", "err", err)
			respondJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "unready", "error": "readiness_check_failed",
			})
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

// handleVersion returns the build version and build time of the running binary.
func handleVersion(buildTime, version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]string{
			"build_time": buildTime,
			"version":    version,
		})
	}
}
