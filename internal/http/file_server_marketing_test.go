package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestFileServerMarketingContract 锁定营销站静态契约：
//   - 不存在的语言路径必须 404（不得回落语言首页造成软 404）；
//   - 目录 URL 301 到带斜杠版本；
//   - / 按 Accept-Language 协商。
func TestFileServerMarketingContract(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "dist", "zh-Hans", "index.html"), "<html>Hans home</html>")
	writeTestFile(t, filepath.Join(root, "dist", "zh-Hant", "index.html"), "<html>Hant home</html>")
	writeTestFile(t, filepath.Join(root, "dist", "en", "index.html"), "<html>EN home</html>")
	writeTestFile(t, filepath.Join(root, "dist", "en", "terms", "index.html"), "<html>EN terms</html>")
	writeTestFile(t, filepath.Join(root, "dist", "zh-Hant", "about", "index.html"), "<html>Hant about</html>")
	writeTestFile(t, filepath.Join(root, "dist", "zh-Hans", "about", "index.html"), "<html>Hans about</html>")
	writeTestFile(t, filepath.Join(root, "dist", "zh-Hant", "hant-only", "index.html"), "<html>hant only</html>")
	writeTestFile(t, filepath.Join(root, "dist", "zh-Hans", "natal", "index.html"), "<html>Hans natal</html>")
	writeTestFile(t, filepath.Join(root, "dist", "en", "natal", "index.html"), "<html>EN natal</html>")
	writeTestFile(t, filepath.Join(root, "dist", "zh-Hant", "natal", "index.html"), "<html>Hant natal</html>")
	writeTestFile(t, filepath.Join(root, "dist", "zh-Hant", "hepan", "index.html"), "<html>Hant hepan</html>")
	writeTestFile(t, filepath.Join(root, "404.html"), "<html>not found</html>")

	handler := fileServer(root, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next invoked for %s", r.URL.Path)
	}))

	tests := []struct {
		name       string
		path       string
		accept     string
		wantStatus int
		wantLoc    string
		wantBody   string
	}{
		{name: "unknown locale path is 404", path: "/zh-Hans/does-not-exist", wantStatus: http.StatusNotFound, wantBody: "not found"},
		{name: "locale home still serves", path: "/zh-Hans/", wantStatus: http.StatusOK, wantBody: "Hans home"},
		{name: "legacy html path is 404", path: "/zh-Hant/naming.html", wantStatus: http.StatusNotFound, wantBody: "not found"},
		{name: "directory without slash normalizes", path: "/en/terms", wantStatus: http.StatusMovedPermanently, wantLoc: "/en/terms/"},
		{name: "root negotiates zh-Hans", path: "/", accept: "zh-CN", wantStatus: http.StatusFound, wantLoc: "/zh-Hans/"},
		{name: "root defaults to zh-Hant", path: "/", wantStatus: http.StatusFound, wantLoc: "/zh-Hant/"},
		{name: "admin.html redirects to app admin", path: "/admin.html", wantStatus: http.StatusMovedPermanently, wantLoc: "/app/admin"},
		{name: "unprefixed path negotiates", path: "/about/", accept: "zh-CN", wantStatus: http.StatusMovedPermanently, wantLoc: "/zh-Hans/about/"},
		{name: "unprefixed path defaults", path: "/about/", wantStatus: http.StatusMovedPermanently, wantLoc: "/zh-Hant/about/"},
		{name: "unprefixed path falls back when negotiated locale lacks page", path: "/hant-only/", accept: "en-US", wantStatus: http.StatusMovedPermanently, wantLoc: "/zh-Hant/hant-only/"},
		{name: "removed skills entry is 404", path: "/skills", wantStatus: http.StatusNotFound, wantBody: "not found"},
		{name: "removed skill artifact is 404", path: "/skills/liki/SKILL.md", wantStatus: http.StatusNotFound, wantBody: "not found"},
		{name: "hepan page serves", path: "/zh-Hant/hepan/", wantStatus: http.StatusOK, wantBody: "Hant hepan"},
		{name: "natal page serves", path: "/zh-Hans/natal/", wantStatus: http.StatusOK, wantBody: "Hans natal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.accept != "" {
				req.Header.Set("Accept-Language", tt.accept)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if tt.wantLoc != "" && rec.Header().Get("Location") != tt.wantLoc {
				t.Fatalf("location=%q want %q", rec.Header().Get("Location"), tt.wantLoc)
			}
			if tt.wantBody != "" && !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("body=%q want contains %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestFileServerCachePolicy(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "dist", "zh-Hant", "index.html"), "<html>home</html>")
	writeTestFile(t, filepath.Join(root, "dist", "assets", "app.abc123.js"), "export {}")
	writeTestFile(t, filepath.Join(root, "dist", "css", "app.css"), "body{}")
	writeTestFile(t, filepath.Join(root, "dist", "app", "index.html"), "<html>app</html>")
	writeTestFile(t, filepath.Join(root, "404.html"), "<html>not found</html>")

	handler := fileServer(root, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next invoked for %s", r.URL.Path)
	}))

	tests := []struct {
		path      string
		wantCache string
	}{
		{"/zh-Hant/", "public, max-age=0, s-maxage=300, stale-while-revalidate=600"},
		{"/assets/app.abc123.js", "public, max-age=31536000, immutable"},
		{"/css/app.css", "public, max-age=86400"},
		{"/app", "no-store"},
		{"/does-not-exist", "no-store"},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if got := rec.Header().Get("Cache-Control"); got != tc.wantCache {
				t.Fatalf("cache-control = %q, want %q", got, tc.wantCache)
			}
		})
	}
}
