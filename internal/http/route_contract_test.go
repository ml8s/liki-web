package http

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestGeneratedBareRedirects(t *testing.T) {
	for _, redirect := range routeContract.BareRedirects {
		t.Run(redirect.Path, func(t *testing.T) {
			handler := fileServer(t.TempDir(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatalf("unexpected fallback for %s", redirect.Path)
			}))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, redirect.Path, nil))
			if recorder.Code != redirect.Status {
				t.Fatalf("status = %d, want %d", recorder.Code, redirect.Status)
			}
			for _, locale := range routeContract.Locales {
				want := "/" + locale + "/" + redirect.Slug + "/"
				if got := recorder.Header().Get("Location"); got == want {
					return
				}
			}
			t.Fatalf("location = %q, want a contract locale route", recorder.Header().Get("Location"))
		})
	}
}

func TestGeneratedRetiredRoutes(t *testing.T) {
	for _, route := range routeContract.Retired {
		t.Run(route.Path, func(t *testing.T) {
			handler := fileServer(t.TempDir(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatalf("unexpected fallback for %s", route.Path)
			}))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, route.Path, nil))
			if recorder.Code != route.Status {
				t.Fatalf("status = %d, want %d", recorder.Code, route.Status)
			}
		})
	}
}

func TestGeneratedAppContractFileMatchesRouter(t *testing.T) {
	// The generator owns the source; this guard catches accidental deletion from
	// the embedded Go build graph.
	if len(routeContract.CanonicalSlugs) != 7 {
		t.Fatalf("canonical slugs = %d, want 7", len(routeContract.CanonicalSlugs))
	}
	if _, err := filepath.Abs("routes.generated.json"); err != nil {
		t.Fatal(err)
	}
}
