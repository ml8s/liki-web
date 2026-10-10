package httputil

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// ClientIP resolves the client address through a configured number of trusted
// reverse proxies. Proxy-supplied forwarding headers are never trusted by
// default; a proxy appends the observable peer to X-Forwarded-For, so the
// client is selected from the right edge of that list.
func ClientIP(r *http.Request) string {
	return ClientIPWithTrustedProxies(r, trustedProxyHops())
}

func trustedProxyHops() int {
	value := strings.TrimSpace(os.Getenv("TRUSTED_PROXY_HOPS"))
	if value == "" {
		// Production web is deployed behind exactly one Caddy edge. Direct
		// requests to the internal port are not public.
		return 1
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 1
	}
	return n
}

func ClientIPWithTrustedProxies(r *http.Request, trustedProxies int) string {
	remote := remoteIP(r)
	if trustedProxies <= 0 {
		return remote
	}

	values := r.Header.Values("X-Forwarded-For")
	if len(values) == 0 {
		return remote
	}
	entries := make([]string, 0, len(values))
	for _, header := range values {
		for item := range strings.SplitSeq(header, ",") {
			if item = strings.TrimSpace(item); item != "" {
				entries = append(entries, item)
			}
		}
	}
	if len(entries) == 0 {
		return remote
	}

	index := max(len(entries)-trustedProxies, 0)
	for i := index; i < len(entries); i++ {
		if ip := normalizeForwardedIP(entries[i]); ip != "" {
			return ip
		}
	}
	return remote
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return strings.TrimSpace(host)
}

func normalizeForwardedIP(value string) string {
	value = strings.TrimSpace(value)
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	if net.ParseIP(value) == nil {
		return ""
	}
	return value
}
