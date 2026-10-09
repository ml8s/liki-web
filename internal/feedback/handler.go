package feedback

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/ml8s/liki-web/internal/httputil"
)

// CreateRequest 是 POST /api/feedback 的最小契约：任何合法 JSON 对象都接受，
// 字段由 payload 内的 schema_version 自描述，后端不解字段——schema 演进不触发后端改动。
type CreateRequest struct {
	SchemaVersion string `json:"schema_version"`
}

type UpdateRequest struct {
	Status   string `json:"status"`
	IssueURL string `json:"issue_url,omitempty"`
}

func (r UpdateRequest) Validate() error {
	switch r.Status {
	case "pending", "processing", "done", "rejected":
	default:
		return errors.New("status must be pending, processing, done, or rejected")
	}
	if r.IssueURL != "" && !strings.HasPrefix(r.IssueURL, "https://") {
		return errors.New("issue_url must be an https URL")
	}
	return nil
}

func HandleCreate(store *Store, rateLimit func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "invalid body")
			return
		}
		// 强制 UTF-8（RFC 8259 §8.1）：非 UTF-8 body 一律 400，防止 GBK 乱码入库。
		if !utf8.Valid(body) {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "body must be UTF-8")
			return
		}
		// 极简校验：必须是合法 JSON 对象（map），且非空。不解析业务字段——
		// schema 演进不触发后端改动（含 legacy category/message 与 feedback-v1 两种格式）。
		var probe map[string]any
		if err := json.Unmarshal(body, &probe); err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "body must be a JSON object")
			return
		}
		if len(probe) == 0 {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "body must not be empty")
			return
		}

		hash := sha256Hex(body)
		if id, status, ok := store.ExistingByHash(hash); ok {
			// 字段无关去重：相同内容直接返回既有 id。
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(map[string]string{"id": id, "status": status, "duplicate": "true"})
			return
		}

		fb := &Feedback{Payload: string(body), Hash: hash}
		if err := store.Create(fb); err != nil {
			httputil.InternalError(w)
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
			httputil.Forbidden(w)
			return
		}
		status := r.URL.Query().Get("status")
		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := parseInt(l); err == nil && n > 0 {
				limit = n
			}
		}
		offset := 0
		if o := r.URL.Query().Get("offset"); o != "" {
			if n, err := parseInt(o); err == nil && n > 0 {
				offset = n
			}
		}
		list, err := store.List(status, limit, offset)
		if err != nil {
			httputil.InternalError(w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	}
}

func HandleUpdate(store *Store, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authAPIKey(r, apiKey) {
			httputil.Forbidden(w)
			return
		}
		id := r.PathValue("id")
		if id == "" {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "id required")
			return
		}
		req, ok := httputil.DecodeStrict[UpdateRequest](w, r)
		if !ok {
			return
		}
		if err := store.Update(id, req.Status, req.IssueURL); err != nil {
			httputil.NotFound(w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func authAPIKey(r *http.Request, key string) bool {
	if key == "" {
		return false
	}
	val := r.Header.Get("Authorization")
	if strings.HasPrefix(val, "Bearer ") {
		return subtle.ConstantTimeCompare([]byte(val[7:]), []byte(key)) == 1
	}
	return subtle.ConstantTimeCompare([]byte(val), []byte(key)) == 1
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
