package liki

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestFulfillerDisplayName(t *testing.T) {
	t.Parallel()
	fulfiller := NewFlow(nil, nil, product.Chat, 0)
	if got := fulfiller.DisplayName("zh-Hant"); got != "Liki Chat" {
		t.Fatalf("DisplayName() = %q, want %q", got, "Liki Chat")
	}
}

func TestFulfillerOnPaidGrantsEntitlementIdempotently(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	orders := NewOrderStoreWithCatalog(db, product.NewCatalog(product.CatalogConfig{
		OneTimePriceMinorCNY: 7900, OneTimePriceMinorUSD: 2990, SubscriptionPriceMinorUSD: 1990,
	}))
	entitlements := entitlement.NewStore(db)
	for _, migrate := range []func(context.Context) error{orders.Migrate, entitlements.Migrate} {
		if err := migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	order, err := orders.CreateOrder(ctx, "user-1", "user@example.com", product.Chat, "USD")
	if err != nil {
		t.Fatal(err)
	}
	if err := orders.AttachIntent(ctx, "user-1", order.ID, "pi-1"); err != nil {
		t.Fatal(err)
	}
	intent := payment.PaymentIntent{
		ID: "pi-1", BusinessOrderID: order.ID, UserID: "user-1", Product: string(product.Chat),
	}
	fulfiller := NewFlow(orders, entitlements, product.Chat, 30*24*time.Hour)
	if err := fulfiller.OnPaid(ctx, intent); err != nil {
		t.Fatalf("first OnPaid: %v", err)
	}
	if err := fulfiller.OnPaid(ctx, intent); err != nil {
		t.Fatalf("repeated OnPaid: %v", err)
	}
	paid, err := orders.GetOrderForUser(ctx, "user-1", order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if paid.Status != OrderPaid {
		t.Fatalf("order status = %q, want paid", paid.Status)
	}
	grants, err := entitlements.ListActiveByUser(ctx, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(grants) != 1 || grants[0].SourceID != order.ID {
		t.Fatalf("grants = %+v, want one commerce_order grant", grants)
	}
}
