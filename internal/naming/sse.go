package naming

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// writeSSE serializes an event and writes it as an SSE data frame.
func writeSSE(w http.ResponseWriter, flusher http.Flusher, event any) error {
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
