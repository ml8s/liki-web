package paymentflow

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"liki-web/internal/hepan"
	"liki-web/internal/mingshu"
	"liki-web/internal/naming"
	"liki-web/internal/payment"
	"liki-web/internal/product"
)

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", t.TempDir()+"/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestNamingFlowPreparesAndActivates(t *testing.T) {
	ctx := context.Background()
	store := naming.NewStore(newDB(t))
	if err := store.CreateOrder(ctx, "n-1", "naming", 990, "TWD", "user@example.com", "zh-Hant"); err != nil {
		t.Fatal(err)
	}
	flow := NewNamingFlow(store, 15)

	order, err := flow.PrepareCheckout(ctx, "n-1")
	if err != nil {
		t.Fatal(err)
	}
	if order.AmountMinor != 990 || order.Currency != "TWD" || order.Status != "pending" {
		t.Fatalf("order = %+v", order)
	}
	intent := payment.PaymentIntent{ID: "pi-1", BusinessOrderID: "n-1", AmountMinor: 990}
	if got := flow.SuccessRedirect(intent); got != "/zh-Hant/naming/chat?order_id=n-1" {
		t.Fatalf("redirect = %q", got)
	}
	if err := flow.OnPaid(ctx, intent); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetOrder(ctx, "n-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "paid" || got.ChatExpiresAt == "" {
		t.Fatalf("naming order = %+v", got)
	}
}

func TestMingShuFlowPreparesAndMarksPaid(t *testing.T) {
	ctx := context.Background()
	store := mingshu.NewStore(newDB(t))
	if err := store.CreateOrder(ctx, "m-1", string(product.ProductMingShu), 990, "TWD", "user@example.com", "birth", "zh-Hant"); err != nil {
		t.Fatal(err)
	}
	flow := NewMingShuFlow(store, nil, nil, "")

	order, err := flow.PrepareCheckout(ctx, "m-1")
	if err != nil {
		t.Fatal(err)
	}
	if order.AmountMinor != 990 || order.Status != "pending" {
		t.Fatalf("order = %+v", order)
	}
	intent := payment.PaymentIntent{ID: "pi-2", BusinessOrderID: "m-1", AmountMinor: 990}
	if got := flow.SuccessRedirect(intent); got != "/zh-Hant/mingshu/report?order_id=m-1" {
		t.Fatalf("redirect = %q", got)
	}
	if err := flow.OnPaid(ctx, intent); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetOrder(ctx, "m-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "paid" {
		t.Fatalf("mingshu order = %+v", got)
	}
}

func TestHepanFlowPreparesAndMarksPaid(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)
	if _, err := db.Exec(hepan.Schema); err != nil {
		t.Fatal(err)
	}
	store := hepan.NewStore(db)
	personA := `{"name":"A"}`
	personB := `{"name":"B"}`
	if err := store.CreateOrder(ctx, "h-1", string(product.Product("hepan")), 990, "TWD", "user@example.com", personA, personB, "zh-Hant"); err != nil {
		t.Fatal(err)
	}
	flow := NewHepanFlow(store, nil, nil, "")

	order, err := flow.PrepareCheckout(ctx, "h-1")
	if err != nil {
		t.Fatal(err)
	}
	if order.AmountMinor != 990 || order.Status != "pending" {
		t.Fatalf("order = %+v", order)
	}
	intent := payment.PaymentIntent{ID: "pi-3", BusinessOrderID: "h-1", AmountMinor: 990}
	if got := flow.SuccessRedirect(intent); got != "/zh-Hant/hepan/report?order_id=h-1" {
		t.Fatalf("redirect = %q", got)
	}
	if err := flow.OnPaid(ctx, intent); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetOrder(ctx, "h-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "paid" {
		t.Fatalf("hepan order = %+v", got)
	}
}
