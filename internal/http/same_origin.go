package http

import (
	stdhttp "net/http"
	"net/url"
	"strings"

	"github.com/ml8s/liki-web/internal/httputil"
)

// SameOrigin protects cookie-authenticated state changes. SameSite=Lax is the
// primary cookie boundary; Origin/Referer verification is the application
// boundary. Requests carrying an explicit credential header are exempt: the
// browser never attaches those automatically, so a cross-site page cannot forge
// them (standard CSRF exemption for non-cookie API clients).
func SameOrigin(devMode bool, next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if !unsafeMethod(r.Method) || csrfExempt(r) {
			next.ServeHTTP(w, r)
			return
		}
		// An exact Origin/Referer + forwarded-host match is safe in every
		// environment. The dev-origin allowlist is an additional convenience
		// for direct local servers (for example, Vite on :5173).
		if sameOriginExpectedHost(r) || (devMode && sameOriginAllowed(r, devOrigins)) {
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

// csrfExempt reports whether a state-changing request does not need same-origin
// verification. CSRF only affects credentials the browser attaches implicitly
// (cookies); explicit header credentials and anonymous public JSON are safe.
func csrfExempt(r *stdhttp.Request) bool {
	// Explicit credentials: the browser does not add these cross-site.
	// - Authorization: Bearer token / API key (feedback API and machine clients)
	// - Payment-Proof: aipay 402 redemption credential
	if r.Header.Get("Authorization") != "" || r.Header.Get("Payment-Proof") != "" {
		return true
	}
	// Provider webhooks authenticate with signatures, not browser cookies.
	// `/api/aipay` is a non-browser 402 flow: the first request has no
	// Payment-Proof yet (it returns 402 with the bill), so it cannot be gated
	// on that header. Internal run-control endpoints are server-to-server only.
	if r.URL.Path == "/api/payments/webhook" || r.URL.Path == "/api/aipay" ||
		strings.HasPrefix(r.URL.Path, "/api/internal/") {
		return true
	}
	// Feedback is public anonymous JSON without cookies. A non-browser script
	// has no browser-supplied Origin/Referer, so the missing-header case is
	// allowed (covers POST and PATCH under /api/feedback). Cross-site browser
	// requests still carry Origin and fall through to the origin check.
	// Rate limiting and payload validation remain the abuse boundary.
	if strings.HasPrefix(r.URL.Path, "/api/feedback") &&
		r.Header.Get("Origin") == "" && r.Header.Get("Referer") == "" {
		return true
	}
	return false
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
