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

func TestAdminSubscriptionsEndpoint(t *testing.T) {
	ctx := context.Background()
	authorizer := openAdminAuth(t)
	subscriptionDB, err := sqliteutil.Open(filepath.Join(t.TempDir(), "sales.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = subscriptionDB.Close() })
	subscriptions := sales.NewSubscriptionStore(subscriptionDB)
	if err := subscriptions.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	active, err := subscriptions.CreatePending(ctx, "user-1", "customer@example.com", product.Chat, product.USD)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := subscriptionDB.ExecContext(ctx,
		`UPDATE sales_subscriptions SET status='active', provider_subscription_id='sub_123' WHERE id=?`,
		active.ID); err != nil {
		t.Fatal(err)
	}

	salesStores := sales.NewSalesStores(nil, subscriptions)
	mux := http.NewServeMux()
	RegisterRoutes(mux, nil, authorizer, identityFor(activeAdmin()), nil, nil, nil, salesStores, nil, nil)

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/admin/sales/subscriptions?search=sub_123", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			Subscriptions []struct {
				ID                     string `json:"id"`
				Email                  string `json:"email"`
				Status                 string `json:"status"`
				ProviderSubscriptionID string `json:"provider_subscription_id"`
			} `json:"subscriptions"`
			Page  int   `json:"page"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	items := response.Data.Subscriptions
	if len(items) != 1 || items[0].Email != "customer@example.com" || items[0].Status != "active" {
		t.Fatalf("unexpected subscriptions: %+v", items)
	}
	if items[0].ProviderSubscriptionID != "sub_123" {
		t.Fatalf("provider subscription id lost: %+v", items[0])
	}
	if response.Data.Total != 1 || response.Data.Page != 1 {
		t.Fatalf("unexpected pagination: page=%d total=%d", response.Data.Page, response.Data.Total)
	}

	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/admin/sales/subscriptions?status=pending", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("filter status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"subscriptions":[]`) {
		t.Fatalf("pending filter should be empty: %s", recorder.Body.String())
	}
}
