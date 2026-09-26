package hepan

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/product"
)

func newTestHandlerStore(t *testing.T) *Store {
	t.Helper()
	return newTestStore(t)
}

// ── HandleCreateOrder ──

func TestHandleCreateOrderIsRetired(t *testing.T) {
	t.Parallel()
	store := newTestHandlerStore(t)
	h := HandleCreateOrder(store)

	body := `{"person_a":{"name":"张三","birth":"1990-05-20","city":"上海","gender":"male"},"person_b":{"name":"李四","birth":"1992-08-15","city":"北京","gender":"female"}}`
	r := httptest.NewRequest("POST", "/api/hepan/orders", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusGone {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusGone, w.Body.String())
	}
}

// ── HandleGetReport ──

func TestHandleGetReport_Success(t *testing.T) {
	t.Parallel()
	store := newTestHandlerStore(t)
	orderID := seedOrder(t, store)
	if _, err := store.MarkPaidIdempotent(context.Background(), orderID, "pay-1"); err != nil {
		t.Fatal(err)
	}
	token, err := httputil.NewOrderAccessToken("hepan", orderID, time.Now(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	report := `{"content":"合盘报告内容"}`
	store.SaveReport(context.Background(), orderID, report) //nolint:errcheck

	h := HandleGetReport(store)
	r := httptest.NewRequest("GET", "/api/hepan/orders/"+orderID+"/report?access_token="+token, nil)
	r.SetPathValue("id", orderID)
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp) //nolint:errcheck
	if !strings.Contains(resp.Data.Content, "合盘报告") {
		t.Errorf("content = %q", resp.Data.Content)
	}
}

func TestHandleGetReport_NotFound(t *testing.T) {
	t.Parallel()
	store := newTestHandlerStore(t)
	h := HandleGetReport(store)

	r := httptest.NewRequest("GET", "/api/hepan/orders/nonexistent/report", nil)
	r.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleGetReport_EmptyID(t *testing.T) {
	t.Parallel()
	store := newTestHandlerStore(t)
	h := HandleGetReport(store)

	r := httptest.NewRequest("GET", "/api/hepan/orders//report", nil)
	r.SetPathValue("id", "")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// HandleCheckout and shared currency detection are covered by payment runtime
// and product package tests for the generic flow.

func TestNewOrderID_IsUnique(t *testing.T) {
	t.Parallel()
	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id := product.NewOrderID()
		if ids[id] {
			t.Fatal("duplicate order ID")
		}
		ids[id] = true
	}
}

func TestLocaleFromAccept(t *testing.T) {
	t.Parallel()
	tests := []struct {
		accept string
		want   string
	}{
		{"zh-CN,zh;q=0.9", "zh-Hans"},
		{"zh-Hans,zh;q=0.9", "zh-Hans"},
		{"zh-TW,zh;q=0.9", "zh-Hant"},
		{"zh-HK,zh;q=0.9", "zh-Hant"},
		{"zh,en;q=0.9", "zh-Hant"},
		{"en,zh;q=0.9", "en"},
		{"fr;q=0.9", "en"},
		{"", "en"},
	}
	for _, tc := range tests {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Accept-Language", tc.accept)
		got := httputil.LocaleFromAccept(r)
		if got != tc.want {
			t.Errorf("httputil.LocaleFromAccept(%q) = %q, want %q", tc.accept, got, tc.want)
		}
	}
}

// ── HandleWebhook ──

func TestHandleGetReport_RequiresPaidAndAuthorization(t *testing.T) {
	t.Parallel()
	store := newTestHandlerStore(t)
	orderID := seedOrder(t, store)
	store.SaveReport(context.Background(), orderID, `{"content":"report"}`) //nolint:errcheck
	h := HandleGetReport(store)

	r := httptest.NewRequest("GET", "/api/hepan/orders/"+orderID+"/report", nil)
	r.SetPathValue("id", orderID)
	w := httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("unauthorized = %d, want 404", w.Code)
	}

	token, err := httputil.NewOrderAccessToken("hepan", orderID, time.Now(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	r = httptest.NewRequest("GET", "/api/hepan/orders/"+orderID+"/report?access_token="+token, nil)
	r.SetPathValue("id", orderID)
	w = httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("pending = %d, want 404", w.Code)
	}
}
