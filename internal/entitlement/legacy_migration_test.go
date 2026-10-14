package entitlement

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestEntitlementMigrationPreservesLegacyData(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`
CREATE TABLE products (
 id TEXT PRIMARY KEY, name TEXT NOT NULL, skills TEXT NOT NULL,
 active INTEGER NOT NULL DEFAULT 1);
CREATE TABLE entitlements (
 id TEXT PRIMARY KEY, user_id TEXT NOT NULL, email TEXT NOT NULL,
 product_id TEXT NOT NULL REFERENCES products(id), skills TEXT NOT NULL,
 status TEXT NOT NULL, source TEXT NOT NULL, source_id TEXT NOT NULL,
 starts_at TEXT NOT NULL, expires_at TEXT NOT NULL, created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL, UNIQUE(source, source_id, product_id));
INSERT INTO products VALUES ('liki-chat', 'Liki product.Chat', '["liki"]', 1);
INSERT INTO entitlements VALUES
('legacy', 'user-1', 'user@example.com', 'liki-chat', '["liki"]',
 'active', 'legacy', 'legacy', '2026-01-01T00:00:00Z',
 '2027-01-01T00:00:00Z', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');`)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(db)
	for range 2 {
		if err := store.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM entitlements WHERE id='legacy'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("legacy entitlement count = %d, want 1", count)
	}
	granted, err := store.GrantForProduct(ctx, "user-1", "User@example.com", product.Chat, "test", "source-1", 30*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if granted.Status != Active || granted.ProductID != product.Chat {
		t.Fatalf("granted = %+v", granted)
	}
}
