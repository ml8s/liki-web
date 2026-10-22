package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/ml8s/liki-web/internal/httputil"

	"golang.org/x/time/rate"
)

// ── Middleware ───────────────────────────────────────────────

var prodOrigins = map[string]bool{
	"https://liki.hk": true,
}

var devOrigins = map[string]bool{
	"http://localhost:8080": true,
	"http://localhost:8081": true,
}

func CORSMiddleware(devMode bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := prodOrigins[origin]
		if !allowed && devMode {
			allowed = devOrigins[origin]
		}
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}
		if r.Method == http.MethodOptions && origin != "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders sets app-layer fallback headers. CSP, HSTS, and the
// privacy headers are owned by the edge Caddy (liki-deploy) — do not
// duplicate them here, a second source will drift.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rv := recover(); rv != nil {
				slog.ErrorContext(r.Context(), "panic recovered", "panic", rv, "stack", string(debug.Stack()))
				httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func BodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		next.ServeHTTP(w, r)
	})
}

// ── Rate Limiter ─────────────────────────────────────────────

type ipLimiter struct {
	limiter  *rate.Limiter
	lastUsed time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	entries  map[string]*ipLimiter
	cleanupT *time.Ticker
	done     chan struct{}
	stopOnce sync.Once
}

func clientIP(r *http.Request) string { return httputil.ClientIP(r) }

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		entries:  make(map[string]*ipLimiter),
		cleanupT: time.NewTicker(10 * time.Minute),
		done:     make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	for {
		select {
		case <-rl.cleanupT.C:
			rl.mu.Lock()
			for ip, e := range rl.entries {
				if time.Since(e.lastUsed) > 10*time.Minute {
					delete(rl.entries, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.done:
			return
		}
	}
}

func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() {
		close(rl.done)
	})
	rl.cleanupT.Stop()
}

func (rl *RateLimiter) Wrap(r rate.Limit, burst int, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ip := clientIP(req)
		limiter := rl.getLimiter(ip, r, burst)
		if !limiter.Allow() {
			w.Header().Set("Retry-After", "60")
			httputil.RespondError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
			return
		}
		next(w, req)
	}
}

func (rl *RateLimiter) getLimiter(ip string, r rate.Limit, burst int) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	key := fmt.Sprintf("%s|%.0f|%d", ip, float64(r), burst)
	e, ok := rl.entries[key]
	if !ok {
		limiter := rate.NewLimiter(r, burst)
		rl.entries[key] = &ipLimiter{limiter: limiter, lastUsed: time.Now()}
		return limiter
	}
	e.lastUsed = time.Now()
	return e.limiter
}
