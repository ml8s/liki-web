package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIPUsesRightmostTrustedProxyEntry(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.2:12345"
	r.Header.Set("X-Forwarded-For", "203.0.113.10, 198.51.100.20")

	// With one trusted proxy, the proxy-appended value is the real client.
	if got := ClientIPWithTrustedProxies(r, 1); got != "198.51.100.20" {
		t.Fatalf("client IP = %q, want 198.51.100.20", got)
	}
	// With two trusted proxies, the attacker-supplied left entry is ignored.
	if got := ClientIPWithTrustedProxies(r, 2); got != "203.0.113.10" {
		t.Fatalf("two-hop client IP = %q, want 203.0.113.10", got)
	}
}

func TestClientIPIgnoresForwardedHeadersWithoutTrustedProxy(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.1:60000"
	r.Header.Set("X-Forwarded-For", "203.0.113.1")
	r.Header.Set("CF-Connecting-IP", "203.0.113.2")
	r.Header.Set("X-Real-IP", "203.0.113.3")

	if got := ClientIPWithTrustedProxies(r, 0); got != "192.0.2.1" {
		t.Fatalf("client IP = %q, want remote address", got)
	}
}

func TestClientIPRejectsMalformedTrustedEntry(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.2:60000"
	r.Header.Set("X-Forwarded-For", "not-an-ip")

	if got := ClientIPWithTrustedProxies(r, 1); got != "192.0.2.2" {
		t.Fatalf("client IP = %q, want remote address", got)
	}
}
