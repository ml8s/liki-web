package mingshu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/product"
)

func lbSeedOrder(t *testing.T, s *Store) string {
	t.Helper()
	oid := uuid.New().String()
	birth := "张三 | 1990-05-20 12:00 | 上海 | male"
	if err := s.CreateOrder(context.Background(), oid, string(product.ProductMingShu), 990, "CNY", "test@t.com", birth, "zh-Hans"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return oid
}

func lbSeedPaidOrder(t *testing.T, s *Store) string {
	t.Helper()
	oid := lbSeedOrder(t, s)
	s.MarkPaidIdempotent(context.Background(), oid, "pay-1")             //nolint:errcheck
	s.SaveReport(context.Background(), oid, `{"content":"test report"}`) //nolint:errcheck
	return oid
}

// ── HandleCreateOrder ──

func TestHandleCreateOrderIsRetired(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	h := HandleCreateOrder(s)

	body := `{"birth":{"name":"张三","year":1990,"month":5,"day":20,"hour":12,"minute":0,"city":"上海","gender":"male"},"email":"a@b.co"}`
	r := httptest.NewRequest("POST", "/api/mingshu/orders", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusGone {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusGone, w.Body.String())
	}
}

// ── HandlePaymentReturn ──

func lbAccessToken(t *testing.T, oid string) string {
	t.Helper()
	token, err := httputil.NewOrderAccessToken(string(product.ProductMingShu), oid, time.Now(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// ── HandleGetReport ──

func TestLifebookHandleGetReport_Success(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := lbSeedPaidOrder(t, s)

	h := HandleGetReport(s)
	r := httptest.NewRequest("GET", "/api/mingshu/orders/"+oid+"/report?access_token="+lbAccessToken(t, oid), nil)
	r.SetPathValue("id", oid)
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.Contains(resp.Data.Content, "test report") {
		t.Errorf("content = %q, want 'test report'", resp.Data.Content)
	}
}

func TestLifebookHandleGetReport_NotFound(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	h := HandleGetReport(s)

	r := httptest.NewRequest("GET", "/api/mingshu/orders/no-such/report", nil)
	r.SetPathValue("id", "no-such")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleGetReport_RequiresPaidAndAuthorization(t *testing.T) {
	t.Parallel()
	s := newLBTestStore(t)
	oid := lbSeedPaidOrder(t, s)
	h := HandleGetReport(s)

	r := httptest.NewRequest("GET", "/api/mingshu/orders/"+oid+"/report", nil)
	r.SetPathValue("id", oid)
	w := httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("unauthorized = %d, want 404", w.Code)
	}

	pending := lbSeedOrder(t, s)
	token := lbAccessToken(t, pending)
	r = httptest.NewRequest("GET", "/api/mingshu/orders/"+pending+"/report?access_token="+token, nil)
	r.SetPathValue("id", pending)
	w = httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("pending = %d, want 404", w.Code)
	}
}
