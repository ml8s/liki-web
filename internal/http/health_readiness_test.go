package http

import (
	"context"
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
}

func TestReadyFailsClosedWhenNotConfigured(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/ready", nil)
	rec := httptest.NewRecorder()
	handleReady(nil)(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
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
