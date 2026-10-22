package chat

import (
	"context"
	"testing"
	"time"
)

// 会话列表契约：按最近活动倒序；超过上限只返回最近的 sessionListLimit 条。
func TestListSessionsOrderedAndLimited(t *testing.T) {
	ctx := context.Background()
	chats := newExchangeTestStore(t)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	create := func(touchAt time.Time) Session {
		t.Helper()
		session, err := chats.CreateSession(ctx, "user-1", "liki", ExpertNatal, "会话", "")
		if err != nil {
			t.Fatal(err)
		}
		if err := chats.TouchSession(ctx, session.ID, touchAt); err != nil {
			t.Fatal(err)
		}
		return session
	}

	// 顺序：最近活动在前。
	oldest := create(base)
	middle := create(base.Add(time.Hour))
	newest := create(base.Add(2 * time.Hour))

	sessions, err := chats.ListSessions(ctx, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 3 {
		t.Fatalf("want 3 sessions, got %d", len(sessions))
	}
	got := []string{sessions[0].ID, sessions[1].ID, sessions[2].ID}
	want := []string{newest.ID, middle.ID, oldest.ID}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order mismatch at %d: got %v want %v", i, got, want)
		}
	}

	// 上限：填到超过 sessionListLimit，只返回最近 200 条，最老的被截断。
	for i := 0; i < sessionListLimit; i++ {
		create(base.Add(time.Duration(3+i) * time.Hour))
	}
	sessions, err = chats.ListSessions(ctx, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != sessionListLimit {
		t.Fatalf("want %d sessions, got %d", sessionListLimit, len(sessions))
	}
	for _, session := range sessions {
		if session.ID == oldest.ID {
			t.Fatalf("oldest session must be truncated by the list limit")
		}
	}
}
