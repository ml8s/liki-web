package httputil

import "net/http"

// BeginSSE switches a response to text/event-stream and returns the flusher.
// It returns false when the writer does not support flushing, before any
// header is committed, so the caller can still send a normal error.
func BeginSSE(w http.ResponseWriter) (http.Flusher, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, false
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	return flusher, true
}
