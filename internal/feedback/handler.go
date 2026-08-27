package feedback

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"
)

type CreateRequest struct {
	Category string `json:"category"`
	Message  string `json:"message"`
	Context  string `json:"context,omitempty"`
}

func HandleCreate(store *Store, rateLimit func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "invalid JSON", 400)
			return
		}
		// 强制 UTF-8（RFC 8259 §8.1）：encoding/json 会把非法字节静默替换为 U+FFFD 入库，
		// Windows GBK 客户端的两条反馈因此不可逆损坏（feedback 3376740b/8d453a6a）。
		// fail loud：调用方收到 400 后自查编码重发，而不是存下乱码。
		if !utf8.Valid(body) {
			http.Error(w, "body must be UTF-8", 400)
			return
		}
		var req CreateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "invalid JSON", 400)
			return
		}
		if strings.TrimSpace(req.Message) == "" {
			http.Error(w, "message required", 400)
			return
		}
		if req.Category == "" {
			req.Category = "other"
		}
		fb := &Feedback{
			Category: req.Category,
			Message:  req.Message,
			Context:  req.Context,
		}
		if err := store.Create(fb); err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": fb.ID, "status": fb.Status})
	}
	if rateLimit != nil {
		return rateLimit(handler)
	}
	return handler
}

func HandleList(store *Store, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authAPIKey(r, apiKey) {
			http.Error(w, "unauthorized", http.StatusForbidden)
			return
		}
		status := r.URL.Query().Get("status")
		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := parseInt(l); err == nil && n > 0 {
				limit = n
			}
		}
		list, err := store.List(status, limit)
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	}
}

type UpdateRequest struct {
	Status   string `json:"status"`
	IssueURL string `json:"issue_url,omitempty"`
}

func HandleUpdate(store *Store, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authAPIKey(r, apiKey) {
			http.Error(w, "unauthorized", http.StatusForbidden)
			return
		}
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "id required", 400)
			return
		}
		var req UpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", 400)
			return
		}
		if req.Status == "" {
			http.Error(w, "status required", 400)
			return
		}
		if err := store.Update(id, req.Status, req.IssueURL); err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

func authAPIKey(r *http.Request, key string) bool {
	if key == "" {
		return false
	}
	val := r.Header.Get("Authorization")
	if strings.HasPrefix(val, "Bearer ") {
		return val[7:] == key
	}
	return val == key
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("not a number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
