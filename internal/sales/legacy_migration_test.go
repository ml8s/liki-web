package sales

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestOrderMigrationIsIdempotent(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	orders := NewOrderStoreWithCatalog(db, product.NewCatalog(product.CatalogConfig{
		OneTimePriceMinorCNY: 7900, OneTimePriceMinorUSD: 2990, SubscriptionPriceMinorUSD: 1990,
	}))
	if err := orders.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	order, err := orders.CreateOrder(ctx, "user-1", "user@example.com", product.Chat, "CNY")
	if err != nil {
		t.Fatal(err)
	}
	if err := orders.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := orders.GetOrderForUser(ctx, "user-1", order.ID); err != nil {
		t.Fatalf("order disappeared after idempotent migration: %v", err)
	}
}
