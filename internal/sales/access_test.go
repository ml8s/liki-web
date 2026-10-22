package sales

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

// 越权拒绝矩阵：用户 B 不能读/取消用户 A 的订单。
func TestCrossUserOrderAccessDenied(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	orders := NewOrderStoreWithCatalog(db, product.NewCatalog(product.CatalogConfig{
		OneTimePriceMinorCNY: 7900,
		OneTimePriceMinorUSD: 2990,
	}))
	if err := orders.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	order, err := orders.CreateOrder(ctx, "user-a", "a@example.com", product.Chat, "CNY")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("get", func(t *testing.T) {
		if _, err := orders.GetOrderForUser(ctx, "user-b", order.ID); !errors.Is(err, ErrOrderNotFound) {
			t.Fatalf("want ErrOrderNotFound, got %v", err)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		if err := orders.CancelOrder(ctx, "user-b", order.ID); !errors.Is(err, ErrOrderNotFound) {
			t.Fatalf("want ErrOrderNotFound, got %v", err)
		}
	})

	t.Run("owner still works", func(t *testing.T) {
		kept, err := orders.GetOrderForUser(ctx, "user-a", order.ID)
		if err != nil {
			t.Fatalf("owner access broken: %v", err)
		}
		if kept.Status != OrderPending {
			t.Fatalf("order mutated by foreign attempts: %s", kept.Status)
		}
	})
}
