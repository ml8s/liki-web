package chat

import (
	"encoding/json"
	"net/http"
)

type validator interface {
	Validate() error
}

func responseOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func decode[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"code": "invalid_request", "message": "Invalid JSON body"}})
		return req, false
	}
	if validator, ok := any(req).(validator); ok {
		if err := validator.Validate(); err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]string{"code": "validation_error", "message": "Invalid request parameters"}})
			return req, false
		}
	}
	return req, true
}
