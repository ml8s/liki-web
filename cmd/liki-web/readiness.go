package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var readinessClient = &http.Client{Timeout: 2 * time.Second}

func checkEngineMCPReadiness(ctx context.Context, baseURL string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		return fmt.Errorf("build engine readiness request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := readinessClient.Do(request)
	return checkReadinessResponse(response, err, "engine")
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
