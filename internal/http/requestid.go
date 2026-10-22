package http

import (
	"net/http"
	"strings"

	"github.com/ml8s/liki-web/internal/httputil"
)

// RequestID 为每个请求生成/透传 X-Request-Id（响应头 + context），
// 配合 httputil.RequestIDHandler 让同请求日志可关联。
// 上游传来的 ID 仅在合法字符集与长度内透传，防止头注入与日志污染。
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-Id"))
		if !validRequestID(id) {
			id = httputil.NewRequestID()
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(httputil.WithRequestID(r.Context(), id)))
	})
}

func validRequestID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, ch := range id {
		switch {
		case ch >= '0' && ch <= '9', ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch == '-', ch == '_':
		default:
			return false
		}
	}
	return true
}
