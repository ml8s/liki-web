package httputil

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type requestIDKey struct{}

// WithRequestID 把请求关联 ID 放入 context（由 HTTP 中间件调用）。
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestIDFrom 返回 context 中的请求关联 ID；未设置时返回空串。
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// NewRequestID 生成请求关联 ID（uuid v4）。
func NewRequestID() string {
	return uuid.NewString()
}

// RequestIDHandler 包装 slog handler：对 *Context 变体的日志调用自动附加 request_id，
// 让同一次请求的所有日志可关联（中间件为每个请求注入 ID）。
type RequestIDHandler struct {
	next slog.Handler
}

func NewRequestIDHandler(next slog.Handler) slog.Handler {
	return &RequestIDHandler{next: next}
}

func (h *RequestIDHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *RequestIDHandler) Handle(ctx context.Context, record slog.Record) error {
	if id := RequestIDFrom(ctx); id != "" {
		record.AddAttrs(slog.String("request_id", id))
	}
	return h.next.Handle(ctx, record)
}

func (h *RequestIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &RequestIDHandler{next: h.next.WithAttrs(attrs)}
}

func (h *RequestIDHandler) WithGroup(name string) slog.Handler {
	return &RequestIDHandler{next: h.next.WithGroup(name)}
}
