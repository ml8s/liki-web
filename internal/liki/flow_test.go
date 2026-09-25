package liki

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestFulfillerDisplayName(t *testing.T) {
	t.Parallel()
	fulfiller := NewFulfiller(nil, nil, ProductChat, 0)
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
	orders := NewOrderStore(db)
	entitlements := NewEntitlementStore(db)
	for _, migrate := range []func(context.Context) error{orders.Migrate, entitlements.Migrate} {
		if err := migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	order, err := orders.CreateOrder(ctx, "user-1", "user@example.com", ProductChat, "USD")
	if err != nil {
		t.Fatal(err)
	}
	if err := orders.AttachIntent(ctx, "user-1", order.ID, "pi-1"); err != nil {
		t.Fatal(err)
	}
	intent := payment.PaymentIntent{
		ID: "pi-1", BusinessOrderID: order.ID, UserID: "user-1", Product: string(ProductChat),
	}
	fulfiller := NewFulfiller(orders, entitlements, ProductChat, 30*24*time.Hour)
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
