package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ── CORS ──

func TestCORSMiddleware_AllowsProdOrigin(t *testing.T) {
	t.Parallel()
	w, rec := corsRequest("https://liki.hk", false)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if rec.AllowOrigin != "https://liki.hk" {
		t.Errorf("Allow-Origin = %q, want https://liki.hk", rec.AllowOrigin)
	}
}

func TestCORSMiddleware_DeniesUnknownOrigin(t *testing.T) {
	t.Parallel()
	w, rec := corsRequest("https://evil.com", false)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if rec.AllowOrigin != "" {
		t.Errorf("Allow-Origin = %q, want empty", rec.AllowOrigin)
	}
}

func TestCORSMiddleware_DevModeAllowsLocalhost(t *testing.T) {
	t.Parallel()
	w, rec := corsRequest("http://localhost:8080", true)
	if rec.AllowOrigin != "http://localhost:8080" {
		t.Errorf("Allow-Origin = %q, want http://localhost:8080", rec.AllowOrigin)
	}
	_ = w
}

func TestCORSMiddleware_PreflightReturns204(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodOptions, "/api/health", nil)
	r.Header.Set("Origin", "https://liki.hk")
	r.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()

	handler := CORSMiddleware(false, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for preflight")
	}))
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}

func TestCORSMiddleware_SetsCORSHeaders(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	r.Header.Set("Origin", "https://liki.hk")
	w := httptest.NewRecorder()

	handler := CORSMiddleware(false, okHandler())
	handler.ServeHTTP(w, r)

	if w.Header().Get("Access-Control-Allow-Methods") != "GET, POST, OPTIONS" {
		t.Errorf("Allow-Methods = %q", w.Header().Get("Access-Control-Allow-Methods"))
	}
	if w.Header().Get("Access-Control-Max-Age") != "86400" {
		t.Errorf("Max-Age = %q", w.Header().Get("Access-Control-Max-Age"))
	}
	if w.Header().Get("Vary") != "Origin" {
		t.Errorf("Vary = %q", w.Header().Get("Vary"))
	}
}

// ── SecurityHeaders ──

func TestSecurityHeaders_SetsNosniffAndDeny(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	handler := SecurityHeaders(okHandler())
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", w.Header().Get("X-Content-Type-Options"))
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("X-Frame-Options = %q", w.Header().Get("X-Frame-Options"))
	}
}

// ── Recover ──

func TestRecover_CatchesPanic(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	handler := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
	body := strings.TrimSpace(w.Body.String())
	if !strings.Contains(body, "internal server error") {
		t.Errorf("body = %q, want 'internal server error'", body)
	}
}

func TestRecover_PassesThroughNormalRequests(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	handler := Recover(okHandler())
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// ── BodyLimit ──

func TestBodyLimit_AcceptsSmallBody(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{"key":"value"}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	var gotBody string
	handler := BodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusInternalServerError)
			return
		}
		gotBody = string(b)
	}))
	handler.ServeHTTP(w, r)

	if gotBody != `{"key":"value"}` {
		t.Errorf("body = %q", gotBody)
	}
}

func TestBodyLimit_RejectsOversizedBody(t *testing.T) {
	t.Parallel()
	// 2 MB payload — over the 1 MB limit
	big := make([]byte, 2<<20)
	for i := range big {
		big[i] = 'x'
	}
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(big)))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := BodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Must attempt to read to trigger MaxBytesReader error
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", w.Code)
	}
}

// ── RateLimiter ──

func TestRateLimiter_AllowsRequestWithinLimit(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter()
	defer rl.Stop()

	handler := rl.Wrap(100, 10, okHandler())

	for i := range 5 {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "10.0.0.1:12345"
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want 200", i, w.Code)
		}
	}
}

func TestRateLimiter_RateLimitsAfterExceedingBurst(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter()
	defer rl.Stop()

	// Burst of 1, rate of 1 per hour (effectively 1 request only)
	handler := rl.Wrap(1.0/3600, 1, okHandler())

	// First request should pass
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodGet, "/", nil)
	r1.RemoteAddr = "10.0.0.2:12345"
	handler.ServeHTTP(w1, r1)

	// Second request should be rate limited
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.RemoteAddr = "10.0.0.2:12345"
	handler.ServeHTTP(w2, r2)

	if w1.Code != http.StatusOK {
		t.Errorf("first request: status = %d, want 200", w1.Code)
	}
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second request: status = %d, want 429", w2.Code)
	}
}

func TestRateLimiter_RetryAfterHeader(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter()
	defer rl.Stop()

	handler := rl.Wrap(1.0/3600, 1, okHandler())

	// Exhaust the burst
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodGet, "/", nil)
	r1.RemoteAddr = "10.0.0.3:12345"
	handler.ServeHTTP(w1, r1)

	// Rate limited
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.RemoteAddr = "10.0.0.3:12345"
	handler.ServeHTTP(w2, r2)

	if w2.Header().Get("Retry-After") != "60" {
		t.Errorf("Retry-After = %q, want 60", w2.Header().Get("Retry-After"))
	}
}

func TestRateLimiter_DifferentIPsAreIndependent(t *testing.T) {

	rl := NewRateLimiter()
	defer rl.Stop()

	handler := rl.Wrap(1.0/3600, 1, okHandler())

	// Exhaust for IP A
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.4:12345"
	handler.ServeHTTP(w, r)
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	// IP A should be limited now

	// IP B should still pass
	w3 := httptest.NewRecorder()
	r3 := httptest.NewRequest(http.MethodGet, "/", nil)
	r3.RemoteAddr = "10.0.0.5:12345"
	handler.ServeHTTP(w3, r3)
	if w3.Code != http.StatusOK {
		t.Errorf("different IP: status = %d, want 200", w3.Code)
	}
}

func TestRateLimiter_StopIsIdempotent(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter()
	rl.Stop()
	rl.Stop() // should not panic
}

func TestRateLimiter_CleanupRemovesStaleEntries(t *testing.T) {

	rl := NewRateLimiter()
	defer rl.Stop()

	// Create an entry
	rl.getLimiter("10.0.0.6|1|1", 1, 1)

	// Manually age it
	rl.mu.Lock()
	for _, e := range rl.entries {
		e.lastUsed = time.Now().Add(-20 * time.Minute)
	}
	rl.mu.Unlock()

	// Run cleanup
	rl.cleanupT.Reset(time.Nanosecond)
	time.Sleep(10 * time.Millisecond)

	rl.mu.Lock()
	count := len(rl.entries)
	rl.mu.Unlock()
	if count != 0 {
		t.Errorf("stale entries = %d, want 0", count)
	}
}

// ── helpers ──

type corsResult struct {
	AllowOrigin string
}

func corsRequest(origin string, devMode bool) (*httptest.ResponseRecorder, *corsResult) {
	r := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	r.Header.Set("Origin", origin)
	w := httptest.NewRecorder()

	handler := CORSMiddleware(devMode, okHandler())
	handler.ServeHTTP(w, r)

	return w, &corsResult{
		AllowOrigin: w.Header().Get("Access-Control-Allow-Origin"),
	}
}

func okHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}
