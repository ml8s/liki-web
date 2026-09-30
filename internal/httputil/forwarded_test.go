package httputil

import (
	"net/http/httptest"
	"testing"
)

func TestForwardedHostUsesRightmostTrustedProxyEntry(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "/", nil)
	r.Host = "internal.example"
	r.Header.Set("X-Forwarded-Host", "attacker.example, liki.hk")

	if got := ForwardedHostWithTrustedProxies(r, 1); got != "liki.hk" {
		t.Fatalf("ForwardedHostWithTrustedProxies = %q, want liki.hk", got)
	}
}

func TestForwardedHostRejectsUntrustedHeader(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "/", nil)
	r.Host = "liki.hk"
	r.Header.Set("X-Forwarded-Host", "attacker.example")

	if got := ForwardedHostWithTrustedProxies(r, 0); got != "liki.hk" {
		t.Fatalf("ForwardedHostWithTrustedProxies = %q, want liki.hk", got)
	}
}
