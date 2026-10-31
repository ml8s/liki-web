package chat

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func newExchangeTestStore(t *testing.T) *ChatStore {
	t.Helper()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	chats := NewChatStore(db)
	if err := chats.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return chats
}

func TestAppendExchangeWritesBatchInOrder(t *testing.T) {
	ctx := context.Background()
	chats := newExchangeTestStore(t)
	session, err := chats.CreateSession(ctx, "user-1", string(product.SkillConsultation), "natal", "Test", "")
	if err != nil {
		t.Fatal(err)
	}
	// 确定性时钟：验证批次内每条消息取得严格递增的时间戳（列表按
	// created_at 排序，同刻消息的次序会退化到随机 id 上）。
	base := time.Now().UTC()
	step := 0
	chats.nowFunc = func() time.Time {
		now := base.Add(time.Duration(step) * time.Millisecond)
		step++
		return now
	}

	written, err := chats.AppendExchange(ctx, "user-1", session.ID,
		PendingMessage{Role: "user", Kind: KindText, Content: "问题"},
		PendingMessage{Role: "assistant", Kind: KindText, Content: "答案"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 2 || written[0].Content != "问题" || written[1].Content != "答案" {
		t.Fatalf("written = %+v", written)
	}
	if !written[0].CreatedAt.Before(written[1].CreatedAt) {
		t.Fatalf("timestamps must be strictly ordered: %v .. %v", written[0].CreatedAt, written[1].CreatedAt)
	}

	messages, _, err := chats.ListMessages(ctx, "user-1", session.ID, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || messages[0].Content != "问题" || messages[1].Content != "答案" {
		t.Fatalf("messages = %+v", messages)
	}
}

func TestAppendExchangeRollsBackWholeBatch(t *testing.T) {
	ctx := context.Background()
	chats := newExchangeTestStore(t)
	session, err := chats.CreateSession(ctx, "user-1", string(product.SkillConsultation), "natal", "Test", "")
	if err != nil {
		t.Fatal(err)
	}
	// 第二条消息复用同一个 id：主键冲突必须让整批回滚，不得留下半条对话。
	ids := []string{"exchange-dup", "exchange-dup"}
	calls := 0
	chats.idFunc = func() string {
		id := ids[calls]
		calls++
		return id
	}

	if _, err := chats.AppendExchange(ctx, "user-1", session.ID,
		PendingMessage{Role: "user", Kind: KindText, Content: "问题"},
		PendingMessage{Role: "assistant", Kind: KindText, Content: "答案"},
	); err == nil {
		t.Fatal("duplicate message id must fail the batch")
	}
	messages, _, err := chats.ListMessages(ctx, "user-1", session.ID, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 {
		t.Fatalf("rolled-back batch must leave no messages: %+v", messages)
	}
}
