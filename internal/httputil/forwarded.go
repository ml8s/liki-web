package httputil

import (
	"net"
	"net/http"
	"strings"
)

// ForwardedHost resolves the public Host presented by a configured number of
// trusted reverse proxies. Host headers use the same rightmost rule as
// ClientIP so an attacker cannot choose an arbitrary forwarded value.
func ForwardedHost(r *http.Request) string {
	return ForwardedHostWithTrustedProxies(r, trustedProxyHops())
}

func ForwardedHostWithTrustedProxies(r *http.Request, trustedProxies int) string {
	values := r.Header.Values("X-Forwarded-Host")
	if trustedProxies <= 0 || len(values) == 0 {
		return canonicalHost(r.Host)
	}
	entries := make([]string, 0, len(values))
	for _, value := range values {
		for item := range strings.SplitSeq(value, ",") {
			if item = canonicalHost(item); item != "" {
				entries = append(entries, item)
			}
		}
	}
	if len(entries) == 0 {
		return canonicalHost(r.Host)
	}
	index := max(len(entries)-trustedProxies, 0)
	return entries[index]
}

func canonicalHost(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(value); err == nil {
		return value
	}
	if strings.ContainsAny(value, " \t\r\n\\") || strings.Count(value, ":") > 1 {
		return ""
	}
	return value
}
