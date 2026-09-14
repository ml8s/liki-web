package payment

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestIntentStoreMigrateNormalizesHistoricalMingShu(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "payments.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store := NewIntentStore(db)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	intent := PaymentIntent{
		ID: "pi-lifebook", BusinessOrderID: "biz-lifebook", Product: "lifebook",
		Status: IntentCreated, AmountMinor: 990, Currency: "CNY",
		Email: "user@example.com", Locale: "zh-Hant",
		Metadata: map[string]string{},
	}
	if err := store.CreateIntent(ctx, intent); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE payment_intents SET product='lifebook' WHERE id=?`, intent.ID); err != nil {
		t.Fatal(err)
	}

	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetIntent(ctx, intent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Product != "mingshu" {
		t.Fatalf("product = %q, want %q", got.Product, "mingshu")
	}
}
