package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFileServerServesAppSPA(t *testing.T) {
	root := t.TempDir()
	appDir := filepath.Join(root, "dist", "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "index.html"), []byte("<html>SPA</html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	handler := fileServer(root, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next invoked for %s", r.URL.Path)
	}))
	for _, path := range []string{"/app", "/app/", "/app/login"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if rec.Body.String() != "<html>SPA</html>" {
				t.Fatalf("body=%q", rec.Body.String())
			}
		})
	}
}
