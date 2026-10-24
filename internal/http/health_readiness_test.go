package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadyReportsDatabaseFailure(t *testing.T) {
	t.Parallel()
	handler := handleReady(func(context.Context) error { return errors.New("db unavailable") })
	req := httptest.NewRequest(http.MethodGet, "/api/ready", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "unready") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestReadyReportsConfiguredDatabaseSuccess(t *testing.T) {
	t.Parallel()
	called := false
	handler := handleReady(func(context.Context) error { called = true; return nil })
	req := httptest.NewRequest(http.MethodGet, "/api/ready", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !called {
		t.Fatal("readiness checker was not called")
	}
	var payload struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.Status != "ready" {
		t.Fatalf("status = %q, want ready", payload.Data.Status)
	}
}

func TestReadyFailsClosedWhenNotConfigured(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/ready", nil)
	rec := httptest.NewRecorder()
	handleReady(nil)(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache-control = %q, want no-store", got)
	}
}

func TestReadyRouteIsRateLimited(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	limiter := NewRateLimiter()
	defer limiter.Stop()
	RegisterRoutes(mux, ServerDeps{
		Readiness: func(context.Context) error { return nil },
	}, "test", "test", limiter, limiter)

	var lastStatus int
	for range 11 {
		req := httptest.NewRequest(http.MethodGet, "/api/ready", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		lastStatus = rec.Code
	}
	if lastStatus != http.StatusTooManyRequests {
		t.Fatalf("11th readiness status = %d, want 429", lastStatus)
	}
}

func TestHealthIsLivenessAndNoStore(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	handleHealth()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache-control = %q, want no-store", got)
	}
	var payload struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.Status != "ok" {
		t.Fatalf("status = %q, want ok", payload.Data.Status)
	}
}

func TestReadySetsNoStoreOnSuccessAndFailure(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		check  ReadinessChecker
		status int
	}{
		{name: "ready", check: func(context.Context) error { return nil }, status: http.StatusOK},
		{name: "unready", check: func(context.Context) error { return errors.New("dependency down") }, status: http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/ready", nil)
			rec := httptest.NewRecorder()
			handleReady(tc.check)(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d", rec.Code, tc.status)
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-store" {
				t.Fatalf("cache-control = %q, want no-store", got)
			}
		})
	}
}

func TestHealthStaysLiveWhenReadinessFails(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	limiter := NewRateLimiter()
	defer limiter.Stop()
	RegisterRoutes(mux, ServerDeps{
		Readiness: func(context.Context) error { return errors.New("dependency down") },
	}, "test", "test", limiter, limiter)

	health := httptest.NewRecorder()
	mux.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200 while readiness fails", health.Code)
	}
	if got := health.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("health cache-control = %q, want no-store", got)
	}

	ready := httptest.NewRecorder()
	mux.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/api/ready", nil))
	if ready.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready status = %d, want 503", ready.Code)
	}
}
