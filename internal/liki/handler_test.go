package liki

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestHandleProductsReturnsServerSelectedDisplayCurrency(t *testing.T) {
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	entitlements := NewEntitlementStore(db)
	if err := entitlements.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(nil, entitlements, nil, nil, nil)

	request := httptest.NewRequest(http.MethodGet, "/api/chat/products", nil)
	request.Header.Set("CF-IPCountry", "US")
	request = request.WithContext(auth.WithAuth(request.Context(), auth.User{ID: "user-1"}, auth.AuthSession{}, ""))
	recorder := httptest.NewRecorder()
	handler.HandleProducts(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			Products []struct {
				Prices             map[string]int64 `json:"prices"`
				DisplayCurrency    string           `json:"display_currency"`
				DisplayAmountMinor int64            `json:"display_amount_minor"`
			} `json:"products"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data.Products) != 1 {
		t.Fatalf("products = %#v", response.Data.Products)
	}
	product := response.Data.Products[0]
	if product.DisplayCurrency != "USD" || product.DisplayAmountMinor != product.Prices["USD"] {
		t.Fatalf("display pricing = %s/%d, want USD/%d", product.DisplayCurrency, product.DisplayAmountMinor, product.Prices["USD"])
	}
}
