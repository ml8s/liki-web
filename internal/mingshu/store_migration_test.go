package mingshu

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestStoreMigratesHistoricalLifebookProductID(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "mingshu.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	NewStore(db)
	if _, err := db.ExecContext(ctx, `
INSERT INTO orders (id, product, amount, currency, email, birth, locale)
VALUES ('order-legacy', 'lifebook', 990, 'CNY', 'user@example.com', 'birth', 'zh-Hant')`); err != nil {
		t.Fatal(err)
	}

	NewStore(db)
	var productID string
	if err := db.QueryRowContext(ctx, `SELECT product FROM orders WHERE id='order-legacy'`).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	if productID != "mingshu" {
		t.Fatalf("product = %q, want mingshu", productID)
	}
}
