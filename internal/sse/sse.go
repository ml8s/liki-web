// Package sse contains the single server-sent event writer used by chat
// product surfaces.
package sse

import (
	"net/http"
)

// Begin sets the standard headers and returns whether the response supports
// flushing. Callers must not have written the response yet.
func Begin(w http.ResponseWriter) (http.Flusher, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, false
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	return flusher, true
}
