package chat

import (
	"context"
	"errors"
	"testing"
)

// 越权拒绝矩阵：用户 B 不能读/改/删用户 A 的会话与消息。
// 作用域（user_id）是 chat 域的核心安全不变量；本测试是它的回归护栏。
func TestCrossUserAccessDenied(t *testing.T) {
	ctx := context.Background()
	chats := newExchangeTestStore(t)

	session, err := chats.CreateSession(ctx, "user-a", "liki", ExpertNatal, "A 的会话", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chats.AppendMessage(ctx, "user-a", session.ID, "user", KindText, "hello"); err != nil {
		t.Fatal(err)
	}

	t.Run("get session", func(t *testing.T) {
		if _, err := chats.GetSession(ctx, "user-b", session.ID); !errors.Is(err, ErrSessionNotFound) {
			t.Fatalf("want ErrSessionNotFound, got %v", err)
		}
	})

	t.Run("rename session", func(t *testing.T) {
		if _, err := chats.RenameSession(ctx, "user-b", session.ID, "被改"); !errors.Is(err, ErrSessionNotFound) {
			t.Fatalf("want ErrSessionNotFound, got %v", err)
		}
	})

	t.Run("soft delete session", func(t *testing.T) {
		if err := chats.SoftDeleteSession(ctx, "user-b", session.ID); !errors.Is(err, ErrSessionNotFound) {
			t.Fatalf("want ErrSessionNotFound, got %v", err)
		}
	})

	t.Run("list messages", func(t *testing.T) {
		messages, cursor, err := chats.ListMessages(ctx, "user-b", session.ID, "", 50)
		if err != nil {
			t.Fatal(err)
		}
		if len(messages) != 0 || cursor != "" {
			t.Fatalf("foreign messages leaked: %d messages, cursor %q", len(messages), cursor)
		}
	})

	t.Run("list sessions", func(t *testing.T) {
		sessions, err := chats.ListSessions(ctx, "user-b")
		if err != nil {
			t.Fatal(err)
		}
		if len(sessions) != 0 {
			t.Fatalf("foreign sessions leaked: %d", len(sessions))
		}
	})

	t.Run("owner still works", func(t *testing.T) {
		if _, err := chats.GetSession(ctx, "user-a", session.ID); err != nil {
			t.Fatalf("owner access broken: %v", err)
		}
	})
}
