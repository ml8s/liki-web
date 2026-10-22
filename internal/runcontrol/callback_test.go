package runcontrol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"

	"github.com/ml8s/liki-web/internal/chat"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

var testCallbackKey = []byte("0123456789abcdef0123456789abcdef") // 32 bytes

const testCallbackSecret = "whsec_MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

type fakeAppend struct {
	userID    string
	sessionID string
	role      string
	content   string
}

// fakeHistory stands in for *chat.ChatStore: the callback is the authoritative
// writer of the run exchange on the direct path.
type fakeHistory struct {
	appends    []fakeAppend
	failAppend bool
	unmatched  bool
}

func (f *fakeHistory) GetSession(_ context.Context, userID, sessionID string) (chat.Session, error) {
	if f.unmatched {
		return chat.Session{}, chat.ErrSessionNotFound
	}
	return chat.Session{ID: sessionID, UserID: userID}, nil
}

func (f *fakeHistory) AppendExchange(_ context.Context, userID, sessionID string, messages ...chat.PendingMessage) ([]chat.Message, error) {
	if f.failAppend {
		f.failAppend = false
		return nil, errors.New("database unavailable")
	}
	written := make([]chat.Message, 0, len(messages))
	for _, pending := range messages {
		f.appends = append(f.appends, fakeAppend{userID: userID, sessionID: sessionID, role: pending.Role, content: pending.Content})
		written = append(written, chat.Message{SessionID: sessionID, UserID: userID, Role: pending.Role, Kind: pending.Kind, Content: pending.Content})
	}
	return written, nil
}

func newCallbackHarness(t *testing.T) (http.HandlerFunc, *Store, *fakeHistory) {
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
	history := &fakeHistory{}
	handler, err := NewCallbackHandler(CallbackConfig{Secret: testCallbackSecret, Store: store, History: history})
	if err != nil {
		t.Fatalf("NewCallbackHandler() error = %v", err)
	}
	return handler, store, history
}

func callbackPayload(runID, status, answer string) string {
	now := time.Now().UTC()
	payload := map[string]any{
		"specversion": "1.0",
		"id":          "evt_1",
		"source":      "spiffe://liki/runtime/natal",
		"type":        "hk.liki.run.completed",
		"time":        now.Format(time.RFC3339),
		"data": map[string]any{
			"run_id":       runID,
			"user_id":      "user_1",
			"thread_id":    "session_1",
			"profile":      "natal",
			"status":       status,
			"user_message": "你好",
			"answer":       answer,
			"usage":        map[string]any{"llm_calls": 2, "prompt_tokens": 10, "completion_tokens": 20, "thought_tokens": 1},
			"started_at":   now.Add(-time.Minute).Format(time.RFC3339),
			"finished_at":  now.Format(time.RFC3339),
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func signedRequest(t *testing.T, body string, signedAt time.Time) *http.Request {
	t.Helper()
	webhook, err := standardwebhooks.NewWebhookRaw(testCallbackKey)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := webhook.Sign("evt_1", signedAt, []byte(body))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/internal/runs/completed", strings.NewReader(body))
	request.Header.Set(standardwebhooks.HeaderWebhookID, "evt_1")
	request.Header.Set(standardwebhooks.HeaderWebhookTimestamp, fmt.Sprint(signedAt.Unix()))
	request.Header.Set(standardwebhooks.HeaderWebhookSignature, signature)
	request.Header.Set("Idempotency-Key", runIDFromBody(body))
	return request
}

// runIDFromBody mirrors what the runtime sends: Idempotency-Key = run_id.
func runIDFromBody(body string) string {
	var envelope struct {
		Data struct {
			RunID string `json:"run_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(body), &envelope)
	return envelope.Data.RunID
}

func decodeStatus(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return body.Status
}

func mutatePayload(t *testing.T, body string, mutate func(envelope map[string]any, data map[string]any)) string {
	t.Helper()
	var envelope map[string]any
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatal(err)
	}
	data, _ := envelope["data"].(map[string]any)
	mutate(envelope, data)
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestCallbackRecordsCompletedRun(t *testing.T) {
	handler, store, history := newCallbackHarness(t)
	body := callbackPayload("run_1", "completed", "权威答案")

	recorder := httptest.NewRecorder()
	handler(recorder, signedRequest(t, body, time.Now()))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if got := decodeStatus(t, recorder); got != "recorded" {
		t.Fatalf("response status = %q, want recorded", got)
	}

	record, found, err := store.GetRun(context.Background(), "run_1")
	if err != nil || !found {
		t.Fatalf("GetRun() found=%v err=%v", found, err)
	}
	if record.Status != "completed" || record.UserID != "user_1" || record.ThreadID != "session_1" ||
		record.Profile != "natal" || record.Answer != "权威答案" ||
		record.LLMCalls != 2 || record.PromptTokens != 10 || record.CompletionTokens != 20 || record.ThoughtTokens != 1 {
		t.Fatalf("record = %+v", record)
	}
	// 回调是直连路径下聊天历史的唯一权威写入者：用户消息 + 助手回答。
	want := []fakeAppend{
		{userID: "user_1", sessionID: "session_1", role: "user", content: "你好"},
		{userID: "user_1", sessionID: "session_1", role: "assistant", content: "权威答案"},
	}
	if len(history.appends) != len(want) {
		t.Fatalf("appends = %+v, want %+v", history.appends, want)
	}
	for index := range want {
		if history.appends[index] != want[index] {
			t.Fatalf("append[%d] = %+v, want %+v", index, history.appends[index], want[index])
		}
	}
}

func TestCallbackIsIdempotentByRunID(t *testing.T) {
	handler, store, history := newCallbackHarness(t)
	body := callbackPayload("run_1", "completed", "答案")

	for delivery, want := range []string{"recorded", "duplicate"} {
		recorder := httptest.NewRecorder()
		handler(recorder, signedRequest(t, body, time.Now()))
		if recorder.Code != http.StatusOK {
			t.Fatalf("delivery %d: status = %d, want 200", delivery+1, recorder.Code)
		}
		if got := decodeStatus(t, recorder); got != want {
			t.Fatalf("delivery %d: response = %q, want %q", delivery+1, got, want)
		}
	}

	if len(history.appends) != 2 {
		t.Fatalf("appends after retry = %d, want 2 (no duplicate history)", len(history.appends))
	}
	record, found, err := store.GetRun(context.Background(), "run_1")
	if err != nil || !found || record.Answer != "答案" {
		t.Fatalf("record after retries = %+v found=%v err=%v", record, found, err)
	}
}

func TestCallbackRecordsCanceledRunWithoutAnswer(t *testing.T) {
	handler, store, history := newCallbackHarness(t)
	body := callbackPayload("run_2", "canceled", "")

	recorder := httptest.NewRecorder()
	handler(recorder, signedRequest(t, body, time.Now()))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	record, found, err := store.GetRun(context.Background(), "run_2")
	if err != nil || !found {
		t.Fatalf("GetRun() found=%v err=%v", found, err)
	}
	if record.Status != "canceled" || record.Answer != "" {
		t.Fatalf("record = %+v", record)
	}
	// 取消：用户消息保留，助手回答不落库。
	if len(history.appends) != 1 || history.appends[0].role != "user" {
		t.Fatalf("appends = %+v, want only the user message", history.appends)
	}
}

func TestCallbackKeepsAuditWhenSessionUnmatched(t *testing.T) {
	handler, store, history := newCallbackHarness(t)
	history.unmatched = true

	recorder := httptest.NewRecorder()
	handler(recorder, signedRequest(t, callbackPayload("run_1", "completed", "答案"), time.Now()))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (no retry loop for unmatched sessions)", recorder.Code)
	}
	if _, found, _ := store.GetRun(context.Background(), "run_1"); !found {
		t.Fatal("unmatched session must still keep the run audit row")
	}
	if len(history.appends) != 0 {
		t.Fatalf("appends = %+v, want none", history.appends)
	}
}

func TestCallbackCompensatesOnHistoryFailureAndRetrySucceeds(t *testing.T) {
	handler, store, history := newCallbackHarness(t)
	body := callbackPayload("run_1", "completed", "答案")
	history.failAppend = true

	failed := httptest.NewRecorder()
	handler(failed, signedRequest(t, body, time.Now()))
	if failed.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", failed.Code)
	}
	// 补偿：run 行被删除，且失败的批次不得留下部分写入（批次事务语义），
	// 重试可完整重放（历史与审计一致）。
	if _, found, _ := store.GetRun(context.Background(), "run_1"); found {
		t.Fatal("failed delivery must not leave a run row behind")
	}
	if len(history.appends) != 0 {
		t.Fatalf("failed delivery must not leave partial history: %+v", history.appends)
	}

	retry := httptest.NewRecorder()
	handler(retry, signedRequest(t, body, time.Now()))
	if retry.Code != http.StatusOK || decodeStatus(t, retry) != "recorded" {
		t.Fatalf("retry = %d %s, want recorded", retry.Code, retry.Body.String())
	}
	if len(history.appends) != 2 {
		t.Fatalf("appends after retry = %+v, want the full exchange", history.appends)
	}
}

func TestCallbackRejectsInvalidSignature(t *testing.T) {
	handler, store, history := newCallbackHarness(t)
	request := signedRequest(t, callbackPayload("run_1", "completed", "答案"), time.Now())
	// 签名后篡改请求体：验签必须失败且不落库。
	request.Body = io.NopCloser(strings.NewReader(callbackPayload("run_1", "completed", "被篡改")))

	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	if _, found, _ := store.GetRun(context.Background(), "run_1"); found {
		t.Fatal("a tampered callback must not be recorded")
	}
	if len(history.appends) != 0 {
		t.Fatal("a tampered callback must not write history")
	}
}

func TestCallbackRejectsStaleSignature(t *testing.T) {
	handler, store, _ := newCallbackHarness(t)
	body := callbackPayload("run_1", "completed", "答案")

	recorder := httptest.NewRecorder()
	handler(recorder, signedRequest(t, body, time.Now().Add(-10*time.Minute)))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	if _, found, _ := store.GetRun(context.Background(), "run_1"); found {
		t.Fatal("a stale callback must not be recorded")
	}
}

func TestCallbackRejectsIdempotencyKeyMismatch(t *testing.T) {
	handler, store, _ := newCallbackHarness(t)
	request := signedRequest(t, callbackPayload("run_1", "completed", "答案"), time.Now())
	request.Header.Set("Idempotency-Key", "other-run")

	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if _, found, _ := store.GetRun(context.Background(), "run_1"); found {
		t.Fatal("a mismatched idempotency key must not be recorded")
	}
}

func TestCallbackRejectsMalformedPayloads(t *testing.T) {
	valid := callbackPayload("run_1", "completed", "答案")
	cases := map[string]string{
		"not json":             "{",
		"wrong spec":           mutatePayload(t, valid, func(envelope, _ map[string]any) { envelope["specversion"] = "0.3" }),
		"wrong type":           mutatePayload(t, valid, func(envelope, _ map[string]any) { envelope["type"] = "hk.liki.run.started" }),
		"missing source":       mutatePayload(t, valid, func(envelope, _ map[string]any) { envelope["source"] = "" }),
		"bad status":           mutatePayload(t, valid, func(_, data map[string]any) { data["status"] = "succeeded" }),
		"unknown profile":      mutatePayload(t, valid, func(_, data map[string]any) { data["profile"] = "unknown" }),
		"missing run id":       mutatePayload(t, valid, func(_, data map[string]any) { data["run_id"] = "" }),
		"missing user id":      mutatePayload(t, valid, func(_, data map[string]any) { data["user_id"] = "" }),
		"missing user message": mutatePayload(t, valid, func(_, data map[string]any) { data["user_message"] = "" }),
		"bad finished at":      mutatePayload(t, valid, func(_, data map[string]any) { data["finished_at"] = "yesterday" }),
	}
	for name, body := range cases {
		handler, store, history := newCallbackHarness(t)
		recorder := httptest.NewRecorder()
		handler(recorder, signedRequest(t, body, time.Now()))

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400: %s", name, recorder.Code, recorder.Body.String())
		}
		if _, found, _ := store.GetRun(context.Background(), "run_1"); found {
			t.Fatalf("%s: malformed callback must not be recorded", name)
		}
		if len(history.appends) != 0 {
			t.Fatalf("%s: malformed callback must not write history", name)
		}
	}
}

func TestStoreMigrateIsIdempotent(t *testing.T) {
	_, store, _ := newCallbackHarness(t)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}
}
