package http

import (
	stdhttp "net/http"
	"net/url"
	"strings"

	"github.com/ml8s/liki-web/internal/httputil"
)

// SameOrigin protects cookie-authenticated state changes. SameSite=Lax is the
// primary cookie boundary; Origin/Referer verification is the application
// boundary. Provider webhooks are explicitly bypassed.
func SameOrigin(devMode bool, next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if !unsafeMethod(r.Method) || sameOriginBypass(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if devMode && sameOriginAllowed(r, devOrigins) {
			next.ServeHTTP(w, r)
			return
		}
		if !devMode && sameOriginExpectedHost(r) {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeSameOriginError(w)
	})
}

func unsafeMethod(method string) bool {
	switch method {
	case stdhttp.MethodPost, stdhttp.MethodPut, stdhttp.MethodPatch, stdhttp.MethodDelete:
		return true
	default:
		return false
	}
}

func sameOriginBypass(path string) bool {
	// Webhooks authenticate with provider signatures, not browser cookies.
	// The aipay 402 flow is a non-browser agent redemption: the credential
	// is the Payment-Proof header, so same-origin is meaningless there.
	// Both stay rate limited.
	return path == "/api/payments/webhook" || path == "/api/aipay"
}

func sameOriginAllowed(r *stdhttp.Request, allowed map[string]bool) bool {
	if origin := headerOrigin(r, "Origin"); origin != "" {
		return allowed[origin]
	}
	referer := headerOrigin(r, "Referer")
	return referer != "" && allowed[referer]
}

func sameOriginExpectedHost(r *stdhttp.Request) bool {
	if r.Header.Get("Origin") == "" && r.Header.Get("Referer") == "" {
		return false
	}
	expected := strings.ToLower(httputil.ForwardedHost(r))
	if expected == "" {
		return false
	}
	// Behind the edge, Caddy sets X-Forwarded-Proto; production is HTTPS.
	expectedScheme := r.Header.Get("X-Forwarded-Proto")
	if expectedScheme == "" {
		expectedScheme = "https"
	}
	for _, name := range []string{"Origin", "Referer"} {
		value := r.Header.Get(name)
		if value == "" {
			continue
		}
		parsed, err := url.Parse(value)
		if err != nil || strings.ToLower(parsed.Host) != expected {
			return false
		}
		if !strings.EqualFold(parsed.Scheme, expectedScheme) {
			return false
		}
	}
	return true
}

func headerOrigin(r *stdhttp.Request, name string) string {
	value := r.Header.Get(name)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme == "http" {
		return "http://" + parsed.Host
	}
	if parsed.Scheme == "https" {
		return "https://" + parsed.Host
	}
	return ""
}

func writeSameOriginError(w stdhttp.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(stdhttp.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":{"code":"csrf_rejected","message":"cross-site request rejected"}}`))
}
