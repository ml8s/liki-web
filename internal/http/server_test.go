package http

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/feedback"
	"github.com/ml8s/liki-web/internal/partner"

	_ "modernc.org/sqlite"
)

func TestRegisterRoutes_DoesNotPanic(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	rl := NewRateLimiter()
	defer rl.Stop()
	deps := ServerDeps{}
	RegisterRoutes(mux, deps, "test", "dev", rl, rl)

	r := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("health check after RegisterRoutes: status = %d, want 200", w.Code)
	}
}

func TestRegisterRoutes_Feedback(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	rl := NewRateLimiter()
	defer rl.Stop()
	fdb := openFeedbackDB(t)
	fstore, err := feedback.NewStore(fdb)
	if err != nil {
		t.Fatal(err)
	}
	deps := ServerDeps{
		FeedbackStore:  fstore,
		FeedbackAPIKey: "test-key",
	}
	RegisterRoutes(mux, deps, "test", "dev", rl, rl)

	// POST without auth — should work (rate-limited but not auth-gated)
	r := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(`{"category":"bug","message":"test"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Errorf("POST /api/feedback: status = %d, want 201: %s", w.Code, w.Body.String())
	}

	// GET without auth — 403
	r2 := httptest.NewRequest("GET", "/api/feedback", nil)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, r2)
	if w2.Code != 403 {
		t.Errorf("GET /api/feedback without auth: status = %d, want 403", w2.Code)
	}

	// GET with auth — 200
	r3 := httptest.NewRequest("GET", "/api/feedback", nil)
	r3.Header.Set("Authorization", "test-key")
	w3 := httptest.NewRecorder()
	mux.ServeHTTP(w3, r3)
	if w3.Code != 200 {
		t.Errorf("GET /api/feedback with auth: status = %d, want 200", w3.Code)
	}
}

func TestRegisterRoutes_Referral(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	rl := NewRateLimiter()
	defer rl.Stop()
	handler := partner.NewHandler(nil, func(next http.HandlerFunc) http.HandlerFunc { return next }, false)
	deps := ServerDeps{
		PartnerHandler: handler,
	}
	RegisterRoutes(mux, deps, "test", "dev", rl, rl)

	// The handler reads auth context and returns unauthorized for a test
	// request. A 404 would mean the public partner routes were never mounted.
	r := httptest.NewRequest(http.MethodGet, "/api/partner-portal/scout/application", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/partner-portal/scout/application: status = %d, want 401", w.Code)
	}
}

func openFeedbackDB(t *testing.T) *sql.DB {
	t.Helper()
	f, err := os.CreateTemp("", "feedback-srv-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	db, err := sql.Open("sqlite", f.Name()+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close(); _ = os.Remove(f.Name()) })
	return db
}

func TestRegisterRoutes_JSONRPC(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	rl := NewRateLimiter()
	defer rl.Stop()
	deps := ServerDeps{}
	RegisterRoutes(mux, deps, "test", "dev", rl, rl)

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

func TestLegacyPageRedirectTargets(t *testing.T) {
	tests := []struct {
		path string
		want string
		ok   bool
	}{
		{"/zh-Hant/mingshu", "/zh-Hant/natal/", true},
		{"/en/bazi", "/en/natal/", true},
		{"/zh-Hans/partners", "/zh-Hans/", true},
		{"/zh-Hant/natal", "", false},
	}
	for _, tc := range tests {
		got, ok := legacyPageRedirect(tc.path)
		if ok != tc.ok || got != tc.want {
			t.Errorf("legacyPageRedirect(%q) = (%q,%v), want (%q,%v)", tc.path, got, ok, tc.want, tc.ok)
		}
	}
}

func TestFileServer_ServesDirIndex(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(dir+"/zh-Hans", 0755)                                         //nolint:errcheck
	os.WriteFile(dir+"/zh-Hans/index.html", []byte("<h1>zh-Hans</h1>"), 0600) //nolint:errcheck
	os.MkdirAll(dir+"/en", 0755)                                              //nolint:errcheck
	os.WriteFile(dir+"/en/index.html", []byte("<h1>en</h1>"), 0600)           //nolint:errcheck
	os.MkdirAll(dir+"/zh-Hant", 0755)                                         //nolint:errcheck
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

func TestFileServer_ReferralLinkPassThrough(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(dir+"/zh-Hant", 0755)                                      //nolint:errcheck
	os.WriteFile(dir+"/zh-Hant/index.html", []byte("<h1>home</h1>"), 0600) //nolint:errcheck

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Stands in for the mux: GET /r/{code} redirects to the locale landing page.
		http.Redirect(w, r, "/zh-Hant/", http.StatusFound)
	})
	h := fileServer(dir, next)

	r2 := httptest.NewRequest("GET", "/r/TESTCODE", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r2)
	if w.Code != http.StatusFound {
		t.Fatalf("GET /r/TESTCODE: status=%d, want 302 (mux route must not be shadowed by static files)", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/zh-Hant/" {
		t.Fatalf("GET /r/TESTCODE: location=%q, want /zh-Hant/", loc)
	}
}

func TestFileServer_FallbackToZhHant(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(dir+"/zh-Hant", 0755)                                         //nolint:errcheck
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
