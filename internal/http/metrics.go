package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ml8s/liki-web/internal/metrics"
)

// statusRecorder 记录最终状态码（WriteHeader 只第一次生效，Write 视为 200）。
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

// Flush is required by SSE routes such as the AG-UI BFF.
func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Metrics 记录按「路由模板 × 方法 × 状态类」的请求数与累计耗时（秒）。
// 路由模板取自 ServeMux 的 r.Pattern（低基数，绝不使用原始 path）；
// 未匹配到路由的请求归入 "unmatched"。
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		metrics.Default.Inc(metrics.MetricHTTPRequests, map[string]string{
			"route":        route,
			"method":       r.Method,
			"status_class": fmt.Sprintf("%dxx", status/100),
		})
		metrics.Default.ObserveDurationSeconds(
			metrics.MetricHTTPRequestDuration,
			map[string]string{"route": route, "method": r.Method, "status_class": fmt.Sprintf("%dxx", status/100)},
			time.Since(start).Seconds(),
		)
	})
}
