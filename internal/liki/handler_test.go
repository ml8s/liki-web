package liki

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestHandleProductsReturnsServerSelectedDisplayCurrency(t *testing.T) {
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	entitlements := entitlement.NewStore(db)
	if err := entitlements.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	catalog := NewCommerceCatalog(CommerceCatalogConfig{
		OneTimePriceMinorCNY:      7900,
		OneTimePriceMinorUSD:      2990,
		SubscriptionPriceMinorUSD: 1990,
	})
	handler := NewHandler(nil, entitlements, nil, nil, nil, WithCommerceCatalog(catalog))

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
				DisplayCurrency    string `json:"display_currency"`
				DisplayAmountMinor int64  `json:"display_amount_minor"`
				Options            []struct {
					BillingMode string `json:"billing_mode"`
					Price       struct {
						CurrencyCode string `json:"currency_code"`
						MinorUnits   int64  `json:"minor_units"`
					} `json:"price"`
				} `json:"purchase_options"`
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
	if product.DisplayCurrency != "USD" || product.DisplayAmountMinor != 2990 {
		t.Fatalf("display pricing = %s/%d, want USD/2990", product.DisplayCurrency, product.DisplayAmountMinor)
	}
	// Subscription options are only offered when a subscription gateway is
	// configured; this test intentionally wires one-time commerce only.
	if len(product.Options) != 2 {
		t.Fatalf("options = %#v", product.Options)
	}
}
