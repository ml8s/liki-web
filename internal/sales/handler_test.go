package sales

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

// 列表接口只下发本区域可结算的购买选项：国内是 Xunhu 的按次 CNY，境外是 Dodo 的
// 按次 + 订阅 USD。币种与可结算性都由服务端决定，客户端只渲染服务端给的结果。
func TestHandleProductsOffersOnlyRegionPurchasableOptions(t *testing.T) {
	cases := []struct {
		name                  string
		country               string
		noSubscriptionProduct bool
		wantCurrency          string
		wantAmountMinor       int64
		wantOptions           []string
	}{
		{
			name:            "domestic buyer stays on Xunhu one-time",
			country:         "CN",
			wantCurrency:    "CNY",
			wantAmountMinor: 7900,
			wantOptions:     []string{"one_time/CNY/7900"},
		},
		{
			name:            "overseas buyer gets Dodo one-time and subscription",
			country:         "US",
			wantCurrency:    "USD",
			wantAmountMinor: 2990,
			wantOptions:     []string{"one_time/USD/2990", "subscription/USD/1990"},
		},
		{
			name:            "missing geo defaults to Dodo",
			country:         "",
			wantCurrency:    "USD",
			wantAmountMinor: 2990,
			wantOptions:     []string{"one_time/USD/2990", "subscription/USD/1990"},
		},
		{
			// 部署没配订阅商品时不要下发订阅 offer：点下去只会拿到 502。
			name:                  "unconfigured subscription product is not advertised",
			country:               "US",
			noSubscriptionProduct: true,
			wantCurrency:          "USD",
			wantAmountMinor:       2990,
			wantOptions:           []string{"one_time/USD/2990"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.Close() })
			entitlements := entitlement.NewStore(db)
			subscriptions := NewSubscriptionStore(db)
			for _, migrate := range []func(context.Context) error{entitlements.Migrate, subscriptions.Migrate} {
				if err := migrate(ctx); err != nil {
					t.Fatal(err)
				}
			}
			catalog := product.NewCatalog(product.CatalogConfig{
				OneTimePriceMinorCNY:      7900,
				OneTimePriceMinorUSD:      2990,
				SubscriptionPriceMinorUSD: 1990,
			})
			var subscriptionCheckout SubscriptionCheckoutCreator = &fakeSubscriptionCheckoutGateway{}
			if tc.noSubscriptionProduct {
				subscriptionCheckout = nil
			}
			handler := NewHandler(nil, nil, nil, nil,
				WithSkillAuthorizer(entitlements),
				WithCatalog(catalog),
				WithSubscriptions(subscriptions, subscriptionCheckout, &fakeSubscriptionGateway{}, "https://liki.hk"),
			)

			request := httptest.NewRequest(http.MethodGet, "/api/sales/products", nil)
			request.Header.Set("CF-IPCountry", tc.country)
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
			card := response.Data.Products[0]
			if card.DisplayCurrency != tc.wantCurrency || card.DisplayAmountMinor != tc.wantAmountMinor {
				t.Fatalf("display pricing = %s/%d, want %s/%d",
					card.DisplayCurrency, card.DisplayAmountMinor, tc.wantCurrency, tc.wantAmountMinor)
			}
			options := make([]string, 0, len(card.Options))
			for _, option := range card.Options {
				options = append(options, fmt.Sprintf("%s/%s/%d",
					option.BillingMode, option.Price.CurrencyCode, option.Price.MinorUnits))
			}
			if !slices.Equal(options, tc.wantOptions) {
				t.Fatalf("purchase options = %v, want %v", options, tc.wantOptions)
			}
		})
	}
}

func TestHandleCheckoutRejectsUnknownFields(t *testing.T) {
	handler := NewHandler(nil, nil, nil, nil)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/sales/checkouts",
		strings.NewReader(`{"product_id":"liki-chat","billing_mode":"one_time","schema_version":"2"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(auth.WithAuth(request.Context(), auth.User{ID: "user-1"}, auth.AuthSession{}, ""))
	recorder := httptest.NewRecorder()
	handler.HandleCheckout(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s, want 400", recorder.Code, recorder.Body.String())
	}
	var problem struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if problem.Type != "invalid_request" {
		t.Fatalf("problem type = %q, want invalid_request", problem.Type)
	}
}
