package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileServerServesAppSPAAndAssets(t *testing.T) {
	root := t.TempDir()
	appDir := filepath.Join(root, "dist", "app")
	assetDir := filepath.Join(appDir, "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	index := "<html>SPA</html>"
	if err := os.WriteFile(filepath.Join(appDir, "index.html"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "app.js"), []byte("export {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	handler := fileServer(root, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next invoked for %s", r.URL.Path)
	}))

	tests := []struct {
		path       string
		status     int
		content    string
		contentype string
	}{
		{path: "/app", status: http.StatusOK, content: index, contentype: "text/html"},
		{path: "/app/login", status: http.StatusOK, content: index, contentype: "text/html"},
		{path: "/app/assets/app.js", status: http.StatusOK, content: "export {}", contentype: "text/javascript"},
		{path: "/app/assets/missing.js", status: http.StatusNotFound, content: "404 page not found", contentype: "text/plain"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.status {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.content) {
				t.Fatalf("body=%q want %q", rec.Body.String(), tt.content)
			}
			if !strings.Contains(rec.Header().Get("Content-Type"), tt.contentype) {
				t.Fatalf("content-type=%q", rec.Header().Get("Content-Type"))
			}
			if tt.path == "/app/assets/app.js" {
				cache := rec.Header().Get("Cache-Control")
				if cache != "public, max-age=31536000, immutable" {
					t.Fatalf("asset cache-control = %q, want immutable one-year cache", cache)
				}
			}
		})
	}
}
