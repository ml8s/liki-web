package chat

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestChatMigrationPreservesData(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	chats := NewChatStore(db)
	for range 2 {
		if err := chats.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	session, err := chats.CreateSession(ctx, "user-1", string(product.SkillConsultation), ExpertNatal, "Test", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chats.AppendMessage(ctx, "user-1", session.ID, "user", KindText, "hello"); err != nil {
		t.Fatal(err)
	}
	messages, _, err := chats.ListMessages(ctx, "user-1", session.ID, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].Content != "hello" {
		t.Fatalf("messages = %+v", messages)
	}
}
