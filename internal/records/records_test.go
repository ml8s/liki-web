package records

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store
}

func TestCreateListRenameDeleteLifecycle(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	created, err := store.Create(ctx, "user-1", CreateInput{
		Kind:   KindLiuyao,
		Title:  "问卦 · 项目能成吗",
		Input:  json.RawMessage(`{"question":"项目能成吗"}`),
		Output: json.RawMessage(`{"board":{"name":"水火既济"}}`),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Kind != KindLiuyao || string(created.Input) != `{"question":"项目能成吗"}` {
		t.Fatalf("created = %+v", created)
	}

	other, err := store.Create(ctx, "user-1", CreateInput{
		Kind:  KindBazhai,
		Title: "八宅 · 妈妈",
		Input: json.RawMessage(`{"profile_id":"p_1"}`),
	})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	if _, err := store.Create(ctx, "user-2", CreateInput{
		Kind:  KindLiuyao,
		Title: "他人记录",
		Input: json.RawMessage(`{}`),
	}); err != nil {
		t.Fatalf("create foreign: %v", err)
	}

	items, err := store.List(ctx, "user-1", KindLiuyao, 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || items[0].ID != created.ID {
		t.Fatalf("liuyao list = %+v", items)
	}
	all, err := store.List(ctx, "user-1", "", 50)
	if err != nil || len(all) != 2 {
		t.Fatalf("all list = %+v err=%v", all, err)
	}

	renamed, err := store.Rename(ctx, "user-1", created.ID, "改名了")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed.Title != "改名了" || !renamed.UpdatedAt.Equal(created.UpdatedAt) {
		t.Fatalf("renamed = %+v (created updated_at %v)", renamed, created.UpdatedAt)
	}
	if _, err := store.Rename(ctx, "user-2", created.ID, "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign rename err = %v", err)
	}

	if err := store.SoftDelete(ctx, "user-1", other.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := store.SoftDelete(ctx, "user-1", other.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("double delete err = %v", err)
	}
	all, err = store.List(ctx, "user-1", "", 50)
	if err != nil || len(all) != 1 || all[0].ID != created.ID {
		t.Fatalf("list after delete = %+v err=%v", all, err)
	}
}

func TestCreateValidatesInput(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	cases := map[string]CreateInput{
		"unknown kind": {Kind: "qimen", Title: "x", Input: json.RawMessage(`{}`)},
		"empty title":  {Kind: KindBond, Title: "  ", Input: json.RawMessage(`{}`)},
		"long title":   {Kind: KindBond, Title: strings.Repeat("字", 61), Input: json.RawMessage(`{}`)},
		"non object":   {Kind: KindBond, Title: "x", Input: json.RawMessage(`[1]`)},
		"bad output":   {Kind: KindBond, Title: "x", Input: json.RawMessage(`{}`), Output: json.RawMessage(`"x"`)},
	}
	for name, in := range cases {
		if _, err := store.Create(ctx, "user-1", in); err == nil {
			t.Fatalf("%s: expected validation error", name)
		}
	}
}

func recordRequest(method, target, body, userID, recordID string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.SetPathValue("id", recordID)
	return request.WithContext(auth.WithAuth(request.Context(), auth.User{ID: userID}, auth.AuthSession{}, ""))
}

func TestHandlerRoutesAndOwnership(t *testing.T) {
	store := newTestStore(t)
	handler := NewHandler(store, nil, nil)
	ctx := context.Background()
	record, err := store.Create(ctx, "user-1", CreateInput{
		Kind:  KindHuangli,
		Title: "择日 · 婚礼",
		Input: json.RawMessage(`{"event":"wedding","start_date":"2026-10-01","end_date":"2026-10-08"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler.HandleList(recorder, recordRequest(http.MethodGet, "/api/records?kind=huangli", "", "user-1", record.ID))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"择日 · 婚礼"`) {
		t.Fatalf("list = %d %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), `"user_id"`) {
		t.Fatalf("record response must not expose user_id: %s", recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	createBody := `{"kind":"huangli","title":"新建","input":{"event":"test"},"unknown":true}`
	handler.HandleCreate(recorder, recordRequest(http.MethodPost, "/api/records", createBody, "user-1", ""))
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"invalid_request"`) {
		t.Fatalf("unknown create field = %d %s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.HandleCreate(recorder, recordRequest(http.MethodPost, "/api/records", `{"kind":"huangli","title":"新建","input":{"event":"test"}}`, "user-1", ""))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), `"user_id"`) {
		t.Fatalf("created record response must not expose user_id: %s", recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.HandleList(recorder, recordRequest(http.MethodGet, "/api/records?kind=qimen", "", "user-1", record.ID))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("bad kind = %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.HandleRename(recorder, recordRequest(http.MethodPatch, "/api/records/rec-1", `{"title":"  "}`, "user-1", record.ID))
	if recorder.Code != http.StatusUnprocessableEntity || !strings.Contains(recorder.Body.String(), `"validation_error"`) {
		t.Fatalf("invalid rename = %d %s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.HandleRename(recorder, recordRequest(http.MethodPatch, "/api/records/rec-1", `{"title":"新标题"}`, "user-2", record.ID))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("foreign rename = %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.HandleDelete(recorder, recordRequest(http.MethodDelete, "/api/records/rec-1", "", "user-1", record.ID))
	if recorder.Code != http.StatusOK {
		t.Fatalf("delete = %d %s", recorder.Code, recorder.Body.String())
	}
	if _, err := store.Get(ctx, "user-1", record.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get after delete err = %v", err)
	}
}
