package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestAgentOpsURLPointsAtTheOpsListener(t *testing.T) {
	cases := map[string]string{
		"http://experts-natal:8083":  "http://experts-natal:8085",
		"http://experts-natal":       "http://experts-natal:8085",
		"https://experts-natal:8083": "https://experts-natal:8085",
	}
	for input, want := range cases {
		if got := expertOpsURL(input, "8085"); got != want {
			t.Fatalf("expertOpsURL(%q) = %q, want %q", input, got, want)
		}
	}
	// 无法解析或缺少端口配置时原样返回（由就绪检查显式报错）。
	if got := expertOpsURL("::bad::", "8085"); got != "::bad::" {
		t.Fatalf("expertOpsURL(invalid) = %q", got)
	}
	if got := expertOpsURL("http://experts-natal:8083", ""); got != "http://experts-natal:8083" {
		t.Fatalf("expertOpsURL(no ops port) = %q", got)
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
