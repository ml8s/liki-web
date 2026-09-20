package liki

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"liki-web/internal/sqliteutil"
)

func TestLikiStoresMigrateAndPreserveData(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	// Simulate a database created by the legacy DROP/CREATE startup path. The
	// current migration must never discard this row when it runs against an
	// already-populated table.
	_, err = db.Exec(`
CREATE TABLE products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    skills TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1
);
CREATE TABLE entitlements (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    email TEXT NOT NULL,
    product_id TEXT NOT NULL REFERENCES products(id),
    skills TEXT NOT NULL,
    status TEXT NOT NULL,
    source TEXT NOT NULL,
    source_id TEXT NOT NULL,
    starts_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(source, source_id, product_id)
);
INSERT INTO products VALUES ('liki-chat', 'Liki Chat', '["liki"]', 1);
INSERT INTO entitlements VALUES
('legacy', 'user-1', 'user@example.com', 'liki-chat', '["liki"]',
 'active', 'legacy', 'legacy', '2026-01-01T00:00:00Z',
 '2027-01-01T00:00:00Z', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
`)
	if err != nil {
		t.Fatal(err)
	}

	entitlements := NewEntitlementStore(db)
	for i := 0; i < 2; i++ {
		if err := entitlements.Migrate(ctx); err != nil {
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

	granted, err := entitlements.GrantForProduct(
		ctx, "user-1", "User@example.com", ProductChat, "test", "source-1", 30*24*time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	if granted.Status != StatusActive || granted.ProductID != ProductChat {
		t.Fatalf("granted = %+v", granted)
	}

	orders := NewOrderStore(db)
	if err := orders.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	order, err := orders.CreateOrder(ctx, "user-1", "user@example.com", ProductChat, "CNY")
	if err != nil {
		t.Fatal(err)
	}
	if err := orders.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := orders.GetOrderForUser(ctx, "user-1", order.ID); err != nil {
		t.Fatalf("order disappeared after idempotent migration: %v", err)
	}

	chats := NewChatStore(db)
	for i := 0; i < 2; i++ {
		if err := chats.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	session, err := chats.CreateSession(ctx, "user-1", string(SkillLiki), "Test", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chats.AppendMessage(
		ctx, "user-1", session.ID, "user", KindText, "hello",
	); err != nil {
		t.Fatal(err)
	}
	messages, err := chats.ListMessages(ctx, "user-1", session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].Content != "hello" {
		t.Fatalf("messages = %+v", messages)
	}
}
