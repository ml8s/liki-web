package entitlement

import (
	"encoding/json"
	"net/http"
)

func jsonEncode(w http.ResponseWriter, body any) error {
	return json.NewEncoder(w).Encode(body)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = jsonEncode(w, body)
}
