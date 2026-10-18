package migrate_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/chat"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/records"
	"github.com/ml8s/liki-web/internal/sales"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

// 共享 liki.db 的 store 必须各自完成迁移。migrate 以 (liki, id) 为键：两个
// store 一旦使用相同 ID，后者的迁移会被静默跳过、产物缺失——本测试把
// main.go 在 liki.db 上建立的 store 全部跑两遍（第二遍验幂等），再逐一断言
// 产物存在，把这类撞车挡在 CI（fail-closed）。
func TestSharedLikiDBMigrationsDoNotCollide(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	stores := []struct {
		name    string
		migrate func(context.Context) error
	}{
		{"entitlement", entitlement.NewStore(db).Migrate},
		{"subscriptions", sales.NewSubscriptionStore(db).Migrate},
		{"chat", chat.NewChatStore(db).Migrate},
		{"records", records.NewStore(db).Migrate},
		{"orders", sales.NewOrderStore(db).Migrate},
	}
	for pass := 1; pass <= 2; pass++ {
		for _, store := range stores {
			if err := store.migrate(ctx); err != nil {
				t.Fatalf("pass %d: %s migrate: %v", pass, store.name, err)
			}
		}
	}

	artifacts := []struct{ kind, name, owner string }{
		{"table", "entitlements", "entitlement"},
		{"table", "sales_subscriptions", "subscriptions"},
		{"table", "sales_orders", "orders"},
		{"table", "chat_sessions", "chat 001"},
		{"column", "chat_sessions.deleted_at", "chat 004"},
		{"column", "chat_sessions.expert", "chat 005"},
		{"table", "chat_messages", "chat 001"},
		{"column", "chat_messages.kind", "chat 002"},
		{"table", "tool_records", "records"},
	}
	for _, a := range artifacts {
		var count int
		var err error
		switch a.kind {
		case "table":
			err = db.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, a.name).Scan(&count)
		case "column":
			table, column, _ := strings.Cut(a.name, ".")
			err = db.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?`, table, column).Scan(&count)
		}
		if err != nil || count == 0 {
			t.Fatalf("%s: %s %s missing — 共享库迁移 ID 撞车被静默跳过？err=%v",
				a.owner, a.kind, a.name, err)
		}
	}
}
