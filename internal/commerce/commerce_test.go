package commerce

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"liki-web/internal/auth"
	"liki-web/internal/entitlement"
	"liki-web/internal/payment"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "commerce.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	authStore := auth.NewStore(db, []byte("test-key"))
	if err := authStore.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	entitlements := entitlement.NewStore(db)
	if err := entitlements.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	store := NewStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO users (id,email,status,locale,created_at,updated_at) VALUES (?,?, 'active','zh-Hant',?,?)`, "user_1", "user@example.com", now, now); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCatalogAndAmounts(t *testing.T) {
	items := Catalog()
	if len(items) != 1 {
		t.Fatalf("catalog = %d", len(items))
	}
	amount, err := AmountMinor(ProductChat, "USD")
	if err != nil || amount != 2990 {
		t.Fatalf("all amount = %d, %v", amount, err)
	}
	if _, err := AmountMinor("mingshu", "CNY"); err == nil {
		t.Fatal("legacy product accepted")
	}
}

func TestOrderLifecycleAndEntitlementFulfillment(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	store := NewStore(db)
	entitlements := entitlement.NewStore(db)
	order, err := store.CreateOrder(ctx, "user_1", "user@example.com", ProductChat, "CNY")
	if err != nil {
		t.Fatal(err)
	}
	intentID := "pi_1"
	if err := store.AttachIntent(ctx, order.UserID, order.ID, intentID); err != nil {
		t.Fatal(err)
	}
	flow := NewFulfiller(store, entitlements, ProductChat, 24*time.Hour)
	if flow.Product() != string(ProductChat) {
		t.Fatalf("product = %q", flow.Product())
	}
	prepared, err := flow.PrepareCheckout(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.UserID != "user_1" || prepared.Product != string(ProductChat) {
		t.Fatalf("prepared = %+v", prepared)
	}

	intent := payment.PaymentIntent{
		ID: intentID, BusinessOrderID: order.ID, UserID: order.UserID,
		Product: string(ProductChat), Email: order.Email,
	}
	if err := flow.OnPaid(ctx, intent); err != nil {
		t.Fatal(err)
	}
	allowed, err := entitlements.AllowedSkills(ctx, "user_1")
	if err != nil {
		t.Fatal(err)
	}
	want := []entitlement.Skill{entitlement.SkillNaming, entitlement.SkillBazi, entitlement.SkillDivination, entitlement.SkillFengshui}
	if len(allowed) != len(want) {
		t.Fatalf("allowed = %+v, want %+v", allowed, want)
	}
	for i := range want {
		if allowed[i] != want[i] {
			t.Fatalf("allowed = %+v, want %+v", allowed, want)
		}
	}

	// Duplicate fulfillment must not duplicate the entitlement term.
	if err := flow.OnPaid(ctx, intent); err != nil {
		t.Fatal(err)
	}
	items, err := entitlements.ListActiveByUser(ctx, "user_1")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Source != "commerce_order" || items[0].SourceID != order.ID {
		t.Fatalf("duplicate entitlements = %+v", items)
	}

	// Ownership is enforced for direct order reads.
	if _, err := store.GetOrderForUser(ctx, "user_other", order.ID); !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("ownership error = %v", err)
	}
}

func TestFulfillmentRejectsCanceledOrder(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	store := NewStore(db)
	entitlements := entitlement.NewStore(db)
	order, err := store.CreateOrder(ctx, "user_1", "user@example.com", ProductChat, "CNY")
	if err != nil {
		t.Fatal(err)
	}
	intentID := "pi_canceled"
	if err := store.AttachIntent(ctx, order.UserID, order.ID, intentID); err != nil {
		t.Fatal(err)
	}
	if err := store.CancelOrder(ctx, order.UserID, order.ID); err != nil {
		t.Fatal(err)
	}

	flow := NewFulfiller(store, entitlements, ProductChat, 24*time.Hour)
	intent := payment.PaymentIntent{
		ID: intentID, BusinessOrderID: order.ID, UserID: order.UserID,
		Product: string(ProductChat), Email: order.Email,
	}
	if err := flow.OnPaid(ctx, intent); err == nil {
		t.Fatal("canceled order fulfillment succeeded")
	}
	items, err := entitlements.ListActiveByUser(ctx, order.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("canceled order granted entitlements: %+v", items)
	}
}
