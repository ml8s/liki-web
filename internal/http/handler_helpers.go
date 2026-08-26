package http

import (
	"encoding/json"
	"errors"
	"net/http"
)

// decodeJSON decodes the JSON request body into T.
// On failure it writes a standard error response and returns false.
func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			respondError(w, http.StatusRequestEntityTooLarge, ErrCodeTooLarge, "Request body too large")
			return req, false
		}
		respondError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return req, false
	}
	return req, true
}
