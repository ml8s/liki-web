// Package sse contains the single server-sent event writer used by chat
// product surfaces.
package sse

import (
	"encoding/json"
	"fmt"
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

// Write serializes one event as an SSE data frame and flushes it.
func Write(w http.ResponseWriter, flusher http.Flusher, event any) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("sse marshal: %w", err)
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
