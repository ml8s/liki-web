package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleHealth(t *testing.T) {
	t.Parallel()
	h := handleHealth()
	r := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"ok"`) {
		t.Errorf("body should contain ok: %s", w.Body.String())
	}
}

func TestHandleVersion(t *testing.T) {
	t.Parallel()
	h := handleVersion("2026-06-14T10:00:00Z", "v0.1.0")
	r := httptest.NewRequest("GET", "/version", nil)
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "2026-06-14") {
		t.Errorf("body should contain build time: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "v0.1.0") {
		t.Errorf("body should contain version: %s", w.Body.String())
	}
}

func TestClientIP_XForwardedFor(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	// The rightmost entry is appended by the single trusted edge proxy; the
	// leftmost entry is attacker-controlled and must not be selected.
	if got := clientIP(r); got != "5.6.7.8" {
		t.Errorf("clientIP = %q, want 5.6.7.8", got)
	}
}

func TestClientIP_RemoteAddr(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.1:12345"
	if got := clientIP(r); got != "192.168.1.1" {
		t.Errorf("clientIP = %q, want 192.168.1.1", got)
	}
}

func TestClientIP_RemoteAddrNoPort(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1"
	if got := clientIP(r); got != "10.0.0.1" {
		t.Errorf("clientIP = %q, want 10.0.0.1", got)
	}
}

func TestEd3_Health_AllMethods(t *testing.T) {
	t.Parallel()
	h := handleHealth()
	for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH"} {
		t.Run(method, func(t *testing.T) {
			r := httptest.NewRequest(method, "/api/health", nil)
			w := httptest.NewRecorder()
			h(w, r)
			if w.Code != http.StatusOK {
				t.Errorf("%s health: status=%d, want 200", method, w.Code)
			}
			if !strings.Contains(w.Body.String(), `"ok"`) {
				t.Errorf("%s health: body should contain ok: %s", method, w.Body.String())
			}
		})
	}
}

func TestEd3_Health_IgnoresQueryParams(t *testing.T) {
	t.Parallel()
	h := handleHealth()
	r := httptest.NewRequest("GET", "/api/health?foo=bar&baz=qux", nil)
	w := httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status=%d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"ok"`) {
		t.Errorf("body should contain ok: %s", w.Body.String())
	}
}
