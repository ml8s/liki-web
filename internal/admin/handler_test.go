package admin

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/naming"

	_ "modernc.org/sqlite"
)

func newTestAdminService(t *testing.T) *naming.AdminService {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := naming.NewStore(db)
	return naming.NewAdminService(store)
}

func requestWithJWT(t *testing.T, method, path, body, email string) *http.Request {
	t.Helper()
	r, err := http.NewRequest(method, path, strings.NewReader(body))
	if err != nil {
		t.Fatal("NewRequest:", err)
	}
	r.Header.Set("Content-Type", "application/json")
	return r
}

// ── HandleListOrders ──

func TestHandleListOrders_MissingEmail(t *testing.T) {
	t.Parallel()
	svc := newTestAdminService(t)
	h := NewHandler(svc, nil)
	r := requestWithJWT(t, "GET", "/api/admin/orders", "", "")
	w := httptest.NewRecorder()
	h.HandleListOrders()(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleListOrders without email: status=%d, want 400, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleListOrders_NoOrders(t *testing.T) {
	t.Parallel()
	svc := newTestAdminService(t)
	h := NewHandler(svc, nil)
	r := requestWithJWT(t, "GET", "/api/admin/orders?email=nonexistent@test.com", "", "")
	w := httptest.NewRecorder()
	h.HandleListOrders()(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("HandleListOrders: status=%d, want 200, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Orders []any `json:"orders"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if len(resp.Data.Orders) != 0 {
		t.Errorf("expected empty orders array, got %d items", len(resp.Data.Orders))
	}
}

// ── HandleCreateOrder ──

func TestHandleCreateOrderIsRetired(t *testing.T) {
	t.Parallel()
	h := NewHandler(newTestAdminService(t), nil)
	r := requestWithJWT(t, "POST", "/api/admin/orders", `{"product":"naming"}`, "")
	w := httptest.NewRecorder()
	h.HandleCreateOrder()(w, r)
	if w.Code != http.StatusGone {
		t.Fatalf("HandleCreateOrder status = %d, want %d, body=%s", w.Code, http.StatusGone, w.Body.String())
	}
}

// ── HandleUpdateExpire ──

func TestHandleUpdateExpire_MissingOrderID(t *testing.T) {
	t.Parallel()
	svc := newTestAdminService(t)
	h := NewHandler(svc, nil)
	r := requestWithJWT(t, "PATCH", "/api/admin/orders//expire", `{"expires_at":"2026-12-31 00:00:00"}`, "")
	w := httptest.NewRecorder()
	h.HandleUpdateExpire()(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleUpdateExpire no id: status=%d, want 400", w.Code)
	}
}

func TestHandleUpdateExpire_MissingExpiresAt(t *testing.T) {
	t.Parallel()
	svc := newTestAdminService(t)
	h := NewHandler(svc, nil)
	r := requestWithJWT(t, "PATCH", "/api/admin/orders/test-id/expire", `{}`, "")
	r.SetPathValue("id", "test-id")
	w := httptest.NewRecorder()
	h.HandleUpdateExpire()(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleUpdateExpire no expires_at: status=%d, want 400", w.Code)
	}
}

func TestHandleUpdateExpire_BadJSON(t *testing.T) {
	t.Parallel()
	svc := newTestAdminService(t)
	h := NewHandler(svc, nil)
	r := requestWithJWT(t, "PATCH", "/api/admin/orders/test-id/expire", `{broken}`, "")
	r.SetPathValue("id", "test-id")
	w := httptest.NewRecorder()
	h.HandleUpdateExpire()(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleUpdateExpire bad JSON: status=%d, want 400", w.Code)
	}
}

// ── HandleGenerateCodes ──

func TestHandleGenerateCodesIsRetired(t *testing.T) {
	t.Parallel()
	h := NewHandler(newTestAdminService(t), nil)
	r := requestWithJWT(t, "POST", "/api/admin/verification-codes", `{"count":3}`, "")
	w := httptest.NewRecorder()
	h.HandleGenerateCodes()(w, r)
	if w.Code != http.StatusGone {
		t.Fatalf("HandleGenerateCodes status = %d, want %d, body=%s", w.Code, http.StatusGone, w.Body.String())
	}
}

// ── HandleListCodes ──

func TestHandleListCodes_Success(t *testing.T) {
	t.Parallel()
	svc := newTestAdminService(t)
	h := NewHandler(svc, nil)
	r := requestWithJWT(t, "GET", "/api/admin/verification-codes", "", "")
	w := httptest.NewRecorder()
	h.HandleListCodes()(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("HandleListCodes: status=%d, want 200, body=%s", w.Code, w.Body.String())
	}
}
