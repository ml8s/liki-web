package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var readinessClient = &http.Client{Timeout: 2 * time.Second}

// expertOpsURL derives the runtime ops surface URL (readiness/metrics) from the
// expert API URL: /readyz lives on the runtime's dedicated ops listener, not on
// the protocol port.
func expertOpsURL(agentURL, opsPort string) string {
	parsed, err := url.Parse(agentURL)
	if err != nil || parsed.Host == "" || opsPort == "" {
		return agentURL
	}
	parsed.Host = net.JoinHostPort(parsed.Hostname(), opsPort)
	return parsed.String()
}

func checkAgentReadiness(ctx context.Context, agentURL string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(agentURL, "/")+"/readyz", nil)
	if err != nil {
		return fmt.Errorf("build agent readiness request: %w", err)
	}
	response, err := readinessClient.Do(request)
	return checkReadinessResponse(response, err, "liki-agent")
}

func checkReadinessResponse(response *http.Response, err error, name string) error {
	if err != nil {
		return fmt.Errorf("%s readiness: %w", name, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%s readiness: HTTP %d", name, response.StatusCode)
	}
	return nil
}
