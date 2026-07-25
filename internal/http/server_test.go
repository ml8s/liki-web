package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"liki-web/internal/payment"
)

func TestRegisterRoutes_DoesNotPanic(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	rl := NewRateLimiter()
	defer rl.Stop()
	deps := ServerDeps{
		Payment:   &payment.Service{},
		NamingStore:    nil,
		Analytics: &Analytics{},
	}
	RegisterRoutes(mux, deps, "test", "dev", rl)

	r := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("health check after RegisterRoutes: status = %d, want 200", w.Code)
	}
}

func TestRegisterRoutes_JSONRPC(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	rl := NewRateLimiter()
	defer rl.Stop()
	deps := ServerDeps{
		Payment:   &payment.Service{},
		NamingStore:    nil,
		Analytics: &Analytics{},
	}
	RegisterRoutes(mux, deps, "test", "dev", rl)

	r := httptest.NewRequest("POST", "/jsonrpc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("jsonrpc route: status = %d, want 404", w.Code)
	}
}

func TestEd3_Version_AllMethods(t *testing.T) {
	t.Parallel()
	h := handleVersion("test-build", "v0.1.0")
	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			r := httptest.NewRequest(method, "/api/version", nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Errorf("%s version: status=%d, want 200", method, w.Code)
			}
			if !strings.Contains(w.Body.String(), "test-build") {
				t.Errorf("%s version: body missing build time", method)
			}
			if !strings.Contains(w.Body.String(), "v0.1.0") {
				t.Errorf("%s version: body missing version", method)
			}
		})
	}
}


// ── fileServer ──

func TestFileServer_ServesDirIndex(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(dir+"/zh-Hans", 0755) //nolint:errcheck
	os.WriteFile(dir+"/zh-Hans/index.html", []byte("<h1>zh-Hans</h1>"), 0600) //nolint:errcheck
	os.MkdirAll(dir+"/en", 0755) //nolint:errcheck
	os.WriteFile(dir+"/en/index.html", []byte("<h1>en</h1>"), 0600) //nolint:errcheck
	os.MkdirAll(dir+"/zh-Hant", 0755) //nolint:errcheck
	os.WriteFile(dir+"/zh-Hant/index.html", []byte("<h1>zh-Hant</h1>"), 0600) //nolint:errcheck

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	h := fileServer(dir, next)

	tests := []struct {
		path   string
		expect int
		body   string
	}{
		{"/zh-Hans/", http.StatusOK, "zh-Hans"},
		{"/en/", http.StatusOK, "en"},
		{"/zh-Hant/", http.StatusOK, "zh-Hant"},
	}
	for _, tc := range tests {
		r2 := httptest.NewRequest("GET", tc.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r2)
		if w.Code != tc.expect {
			t.Errorf("%s: status=%d, want %d", tc.path, w.Code, tc.expect)
		}
		if tc.body != "" && !strings.Contains(w.Body.String(), tc.body) {
			t.Errorf("%s: body=%q, want contains %q", tc.path, w.Body.String(), tc.body)
		}
	}
}

func TestFileServer_FallbackToZhHant(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(dir+"/zh-Hant", 0755) //nolint:errcheck
	os.WriteFile(dir+"/zh-Hant/index.html", []byte("<h1>default</h1>"), 0600) //nolint:errcheck

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	h := fileServer(dir, next)
	r := httptest.NewRequest("GET", "/somepage", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// Should fall through to zh-Hant/somepage → not found as dir → fileServer calls next handler
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
