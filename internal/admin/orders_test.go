package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sales"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestAdminOrdersEndpoint(t *testing.T) {
	ctx := context.Background()
	authorizer := openAdminAuth(t)
	orderDB, err := sqliteutil.Open(filepath.Join(t.TempDir(), "sales.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = orderDB.Close() })
	orders := sales.NewOrderStoreWithCatalog(orderDB, product.NewCatalog(product.CatalogConfig{
		OneTimePriceMinorUSD: 2990,
	}))
	if err := orders.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := orders.CreateOrder(ctx, "user-1", "customer@example.com", "liki-chat", "USD"); err != nil {
		t.Fatal(err)
	}

	salesStores := sales.NewSalesStores(orders, nil)
	mux := http.NewServeMux()
	RegisterRoutes(mux, nil, authorizer, identityFor(activeAdmin()), nil, nil, nil, salesStores, nil, nil)

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/admin/sales/orders?search=customer", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			Orders []struct {
				ID     string `json:"id"`
				Email  string `json:"email"`
				Status string `json:"status"`
			} `json:"orders"`
			Page  int   `json:"page"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data.Orders) != 1 || response.Data.Orders[0].Email != "customer@example.com" {
		t.Fatalf("unexpected orders: %+v", response.Data.Orders)
	}
	if response.Data.Total != 1 || response.Data.Page != 1 {
		t.Fatalf("unexpected pagination: page=%d total=%d", response.Data.Page, response.Data.Total)
	}

	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/admin/sales/orders?status=paid", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("filter status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"orders":[]`) {
		t.Fatalf("paid filter should be empty: %s", recorder.Body.String())
	}
}
