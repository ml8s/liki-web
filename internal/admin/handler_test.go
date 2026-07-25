package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"liki-web/internal/httputil"
	"liki-web/internal/naming"

	_ "modernc.org/sqlite"
)

func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("ADMIN_EMAIL", "admin@liki.hk")
	os.Exit(m.Run())
}

func newTestAdminService(t *testing.T) *naming.AdminService {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	store := naming.NewStore(db)
	return naming.NewAdminService(store)
}

func setJWT(t *testing.T, w http.ResponseWriter, email string) {
	t.Helper()
	if err := httputil.SetJWTCookie(w, email, uuid.New().String()); err != nil {
		t.Fatal("SetJWTCookie:", err)
	}
}

func requestWithJWT(t *testing.T, method, path, body, email string) *http.Request {
	t.Helper()
	r, err := http.NewRequest(method, path, strings.NewReader(body))
	if err != nil {
		t.Fatal("NewRequest:", err)
	}
	r.Header.Set("Content-Type", "application/json")
	if email != "" {
		w := httptest.NewRecorder()
		setJWT(t, w, email)
		for _, c := range w.Result().Cookies() {
			if c.Name == "liki_token" {
				r.AddCookie(c)
			}
		}
	}
	return r
}

// ── adminAuth ──

func TestAdminAuth_NoCookie(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "/api/admin/orders", nil)
	if adminAuth(r) {
		t.Error("adminAuth should reject request without cookie")
	}
}

func TestAdminAuth_InvalidJWT(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "/api/admin/orders", nil)
	r.AddCookie(&http.Cookie{Name: "liki_token", Value: "not-a-jwt"})
	if adminAuth(r) {
		t.Error("adminAuth should reject invalid JWT")
	}
}

func TestAdminAuth_WrongEmail(t *testing.T) {
	t.Parallel()
	r := requestWithJWT(t, "GET", "/api/admin/orders", "", "user@example.com")
	if adminAuth(r) {
		t.Error("adminAuth should reject non-admin email")
	}
}

func TestAdminAuth_NoneAlgorithm(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "/api/admin/orders", nil)
	r.AddCookie(&http.Cookie{Name: "liki_token", Value: "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJlbWFpbCI6InRlc3RAZXhhbXBsZS5jb20ifQ."})
	if adminAuth(r) {
		t.Error("adminAuth should reject JWT with alg=none")
	}
}

func TestAdminAuth_ValidAdmin(t *testing.T) {
	t.Parallel()
	r := requestWithJWT(t, "GET", "/api/admin/orders", "", "admin@liki.hk")
	if !adminAuth(r) {
		t.Error("adminAuth should accept valid admin email")
	}
}

// ── adminOnly middleware ──

func TestAdminOnly_NoCookie(t *testing.T) {
	t.Parallel()
	handler := adminOnly(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r := httptest.NewRequest("GET", "/api/admin/orders", nil)
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("adminOnly without cookie: status=%d, want 403", w.Code)
	}
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

func TestHandleCreateOrder_BadJSON(t *testing.T) {
	t.Parallel()
	svc := newTestAdminService(t)
	h := NewHandler(svc, nil)
	r := requestWithJWT(t, "POST", "/api/admin/orders", "{bad}", "")
	w := httptest.NewRecorder()
	h.HandleCreateOrder()(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleCreateOrder bad JSON: status=%d, want 400", w.Code)
	}
}

func TestHandleCreateOrder_MissingFields(t *testing.T) {
	t.Parallel()
	svc := newTestAdminService(t)
	h := NewHandler(svc, nil)
	r := requestWithJWT(t, "POST", "/api/admin/orders", `{"product":"naming"}`, "")
	w := httptest.NewRecorder()
	h.HandleCreateOrder()(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleCreateOrder missing fields: status=%d, want 400", w.Code)
	}
}

func TestHandleCreateOrder_Success(t *testing.T) {
	t.Parallel()
	svc := newTestAdminService(t)
	done := make(chan string, 1)
	h := NewHandler(svc, func(_ context.Context, orderID string) {
		done <- orderID
	})
	body := `{"email":"test@example.com","product":"naming","expires_at":"2099-12-31 00:00:00"}`
	r := requestWithJWT(t, "POST", "/api/admin/orders", body, "")
	w := httptest.NewRecorder()
	h.HandleCreateOrder()(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("HandleCreateOrder: status=%d, want 200, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			OrderID string `json:"order_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.OrderID == "" {
		t.Error("HandleCreateOrder: expected order_id in response")
	}
	select {
	case calledOrderID := <-done:
		if calledOrderID != resp.Data.OrderID {
			t.Errorf("HandleCreateOrder: onPaid called with %q, want %q", calledOrderID, resp.Data.OrderID)
		}
	case <-time.After(time.Second):
		t.Error("HandleCreateOrder: onPaid was not called")
	}
}

func TestHandleCreateOrder_Defaults(t *testing.T) {
	t.Parallel()
	svc := newTestAdminService(t)
	h := NewHandler(svc, nil)
	body := `{"email":"test@example.com","expires_at":"2099-12-31 00:00:00"}`
	r := requestWithJWT(t, "POST", "/api/admin/orders", body, "")
	w := httptest.NewRecorder()
	h.HandleCreateOrder()(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("HandleCreateOrder defaults: status=%d, want 200, body=%s", w.Code, w.Body.String())
	}
}

func TestHandleCreateOrder_USD(t *testing.T) {
	t.Parallel()
	svc := newTestAdminService(t)
	h := NewHandler(svc, nil)
	body := `{"email":"test@example.com","product":"naming","currency":"USD","expires_at":"2099-12-31 00:00:00"}`
	r := requestWithJWT(t, "POST", "/api/admin/orders", body, "")
	w := httptest.NewRecorder()
	h.HandleCreateOrder()(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("HandleCreateOrder USD: status=%d, want 200", w.Code)
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

func TestHandleGenerateCodes_BadJSON(t *testing.T) {
	t.Parallel()
	svc := newTestAdminService(t)
	h := NewHandler(svc, nil)
	r := requestWithJWT(t, "POST", "/api/admin/verification-codes", `not-json`, "")
	w := httptest.NewRecorder()
	h.HandleGenerateCodes()(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleGenerateCodes bad JSON: status=%d, want 400", w.Code)
	}
}

func TestHandleGenerateCodes_TooMany(t *testing.T) {
	t.Parallel()
	svc := newTestAdminService(t)
	h := NewHandler(svc, nil)
	r := requestWithJWT(t, "POST", "/api/admin/verification-codes", `{"count":200}`, "")
	w := httptest.NewRecorder()
	h.HandleGenerateCodes()(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleGenerateCodes count>100: status=%d, want 400", w.Code)
	}
}

func TestHandleGenerateCodes_DefaultCount(t *testing.T) {
	t.Parallel()
	svc := newTestAdminService(t)
	h := NewHandler(svc, nil)
	r := requestWithJWT(t, "POST", "/api/admin/verification-codes", `{"count":0}`, "")
	w := httptest.NewRecorder()
	h.HandleGenerateCodes()(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("HandleGenerateCodes count=0: status=%d, want 200, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Codes []struct {
				Code string `json:"code"`
			} `json:"codes"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if len(resp.Data.Codes) != 1 {
		t.Errorf("expected 1 code (default), got %d", len(resp.Data.Codes))
	}
}

func TestHandleGenerateCodes_Success(t *testing.T) {
	t.Parallel()
	svc := newTestAdminService(t)
	h := NewHandler(svc, nil)
	r := requestWithJWT(t, "POST", "/api/admin/verification-codes", `{"count":3}`, "")
	w := httptest.NewRecorder()
	h.HandleGenerateCodes()(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("HandleGenerateCodes: status=%d, want 200, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Codes []struct {
				Code string `json:"code"`
			} `json:"codes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Data.Codes) != 3 {
		t.Errorf("HandleGenerateCodes: got %d codes, want 3", len(resp.Data.Codes))
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
