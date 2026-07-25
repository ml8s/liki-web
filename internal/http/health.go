package http

import (
	"net/http"
)

func handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
