package chat

import (
	"context"
	"strings"
	"testing"
)

// 索引契约（列表查询约定）：
//   - 会话列表按「最近活动」排序必须走表达式索引（无临时排序）；
//   - 消息 keyset 分页必须走覆盖完整排序键的索引（无临时排序）。
//
// 这两个断言把「索引↔排序键对齐」从人脑约定变成可执行契约。
func TestChatListQueriesUseIndexes(t *testing.T) {
	ctx := context.Background()
	chats := newExchangeTestStore(t)

	explain := func(query string, args ...any) string {
		t.Helper()
		rows, err := chats.db.QueryContext(ctx, "EXPLAIN QUERY PLAN "+query, args...)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = rows.Close() }()
		var plan strings.Builder
		for rows.Next() {
			var id, parent, notused int
			var detail string
			if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
				t.Fatal(err)
			}
			plan.WriteString(detail + "\n")
		}
		return plan.String()
	}

	sessions := explain(`
SELECT id FROM chat_sessions WHERE user_id=? AND status='active' AND deleted_at IS NULL
ORDER BY COALESCE(last_message_at,updated_at) DESC LIMIT ?`, "u", sessionListLimit)
	if !strings.Contains(sessions, "idx_chat_sessions_user_recent") {
		t.Fatalf("sessions list must use the recent index, plan:\n%s", sessions)
	}
	if strings.Contains(sessions, "TEMP B-TREE") {
		t.Fatalf("sessions list must not sort in a temp b-tree, plan:\n%s", sessions)
	}

	messages := explain(`
SELECT id FROM chat_messages
WHERE session_id=? AND user_id=? AND (created_at < ? OR (created_at = ? AND id < ?))
ORDER BY created_at DESC, id DESC LIMIT ?`, "s", "u", "t", "t", "i", 201)
	if !strings.Contains(messages, "idx_chat_messages_session_created") {
		t.Fatalf("messages keyset must use the session index, plan:\n%s", messages)
	}
	if strings.Contains(messages, "TEMP B-TREE") {
		t.Fatalf("messages keyset must not sort in a temp b-tree, plan:\n%s", messages)
	}
}
