package account

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"liki-web/internal/auth"
	"liki-web/internal/chat"
	"liki-web/internal/profile"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestHandleErasure(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	profileStore := profile.NewStore(db)
	if err := profileStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	chatStore := chat.NewStore(db)
	if err := chatStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	// user_1 建档案 + 会话 + system_profile 消息（含 PII）
	p, err := profileStore.Create(ctx, "user_1", profile.Input{
		Nickname: "妈妈", Gender: "female", Calendar: "lunar",
		BirthDate: "1958-闰5-12", BirthTime: "06:00", BirthTimeKnown: "known",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := chatStore.CreateSession(ctx, "user_1", "liki-bazi", "标题", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chatStore.AppendMessage(ctx, "user_1", session.ID, "assistant", chat.KindSystemProfile, `{"birth_date":"1958-闰5-12"}`); err != nil {
		t.Fatal(err)
	}

	handler := NewHandler(profileStore, chatStore, func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user := auth.User{ID: "user_1", Email: "u@example.com", Status: auth.UserActive}
			r = r.WithContext(auth.WithAuth(r.Context(), user, auth.AuthSession{ID: "s_1"}, "token"))
			next(w, r)
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/api/account/erasure", nil)
	w := httptest.NewRecorder()
	handler.requireUser(handler.HandleErasure)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Status          string `json:"status"`
			ProfilesCleared int64  `json:"profiles_cleared"`
			MessagesCleared int64  `json:"messages_cleared"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Data.Status != "erased" || resp.Data.ProfilesCleared != 1 || resp.Data.MessagesCleared != 1 {
		t.Fatalf("bad erasure result: %+v", resp.Data)
	}

	// 档案 PII 已清 + 软删；会话消息内容已清
	if _, err := profileStore.Get(ctx, "user_1", p.ID); err != profile.ErrNotFound {
		t.Fatalf("profile should be gone, got err=%v", err)
	}
	msgs, err := chatStore.ListMessages(ctx, "user_1", session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || strings.TrimSpace(msgs[0].Content) != "" {
		t.Fatalf("message PII not cleared: %+v", msgs)
	}
}
