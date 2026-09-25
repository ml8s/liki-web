package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckEngineReadinessUsesJSONRPCHealth(t *testing.T) {
	var method string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":{"status":"ok"},"id":1}`))
	}))
	defer server.Close()

	if err := checkEngineReadiness(context.Background(), server.URL+"/jsonrpc"); err != nil {
		t.Fatalf("checkEngineReadiness() = %v", err)
	}
	if method != http.MethodPost {
		t.Fatalf("method = %s, want POST", method)
	}
}

func TestCheckAgentReadinessUsesReadyZ(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := checkAgentReadiness(context.Background(), server.URL+"/"); err != nil {
		t.Fatalf("checkAgentReadiness() = %v", err)
	}
	if path != "/readyz" {
		t.Fatalf("path = %s, want /readyz", path)
	}
}

func TestCheckReadinessRejectsNonSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkReadinessResponse(response, nil, "test"); err == nil {
		t.Fatal("checkReadinessResponse() error = nil, want HTTP failure")
	}
}
