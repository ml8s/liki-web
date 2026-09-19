package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"liki-web/internal/agent"
	"liki-web/internal/auth"
	"liki-web/internal/entitlement"
	"liki-web/internal/llm"
	"liki-web/internal/profile"

	_ "modernc.org/sqlite"
)

type fakeRunner struct {
	prompt    string
	messages  []llm.Message
	calls     int
	active    int
	maxActive int
}

func (r *fakeRunner) Execute(ctx context.Context, msgs []llm.Message, onEvent func(agent.ChatEvent), _ func() string) ([]llm.Message, error) {
	r.calls++
	r.active++
	if r.active > r.maxActive {
		r.maxActive = r.active
	}
	time.Sleep(20 * time.Millisecond)
	r.active--
	if onEvent != nil {
		onEvent(agent.ChatEvent{Type: agent.EventTextDelta, Content: "助手"})
	}
	r.prompt = msgs[0].Content
	r.messages = msgs
	return append(msgs, llm.Message{Role: llm.RoleAssistant, Content: "助手回复"}), nil
}

func (r *fakeRunner) Prompt() string { return "unified prompt" }

func openChatDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newChatFixture(t *testing.T) (*Service, *Store, *entitlement.Store, *fakeRunner, string) {
	t.Helper()
	db := openChatDB(t)
	authStore := auth.NewStore(db, []byte("test-hmac-key"))
	if err := authStore.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	entitlements := entitlement.NewStore(db)
	if err := entitlements.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	userID := "user_platform"
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.ExecContext(context.Background(), `INSERT INTO users (id,email,status,locale,created_at,updated_at) VALUES (?,?, 'active','zh-Hant',?,?)`, userID, "user@example.com", now, now); err != nil {
		t.Fatal(err)
	}
	granted, err := entitlements.GrantForProduct(context.Background(), userID, "user@example.com", entitlement.ProductChat, "payment_intent", "pay_1", 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{}
	return NewService(store, entitlements, nil, map[Skill]Runner{"liki": runner}), store, entitlements, runner, granted.UserID
}

func TestChatServiceCreateAndSendMessage(t *testing.T) {
	ctx := context.Background()
	service, store, _, runner, userID := newChatFixture(t)

	var events []agent.ChatEvent
	session, err := service.CreateSession(ctx, userID, "liki", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if session.Skill != "liki" || session.Title == "" {
		t.Fatalf("session = %+v", session)
	}
	userMessage, response, gotSession, err := service.SendMessage(ctx, userID, session.ID, " 请帮我起名 ", func(event agent.ChatEvent) { events = append(events, event) })
	if err != nil {
		t.Fatal(err)
	}
	if response.Content != "助手回复" || gotSession.ID != session.ID || userMessage.Content != "请帮我起名" {
		t.Fatalf("user=%+v response=%+v session=%+v", userMessage, response, gotSession)
	}
	if runner.calls != 1 || !strings.Contains(runner.prompt, "unified prompt") {
		t.Fatalf("runner calls=%d prompt=%q", runner.calls, runner.prompt)
	}
	if len(events) != 2 || events[0].Type != agent.EventThinking || events[1].Type != agent.EventTextDelta || events[1].Content != "助手" {
		t.Fatalf("events = %+v", events)
	}
	messages, err := store.ListMessages(ctx, userID, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || messages[0].Role != "user" || messages[0].Content != "请帮我起名" || messages[1].Role != "assistant" {
		t.Fatalf("messages = %+v", messages)
	}
}

func TestChatServiceSerializesSameSession(t *testing.T) {
	ctx := context.Background()
	service, _, _, runner, userID := newChatFixture(t)
	session, err := service.CreateSession(ctx, userID, "liki", "", "")
	if err != nil {
		t.Fatal(err)
	}

	const requests = 3
	done := make(chan error, requests)
	for i := 0; i < requests; i++ {
		go func() {
			_, _, _, err := service.SendMessage(ctx, userID, session.ID, "并发消息", nil)
			done <- err
		}()
	}
	for i := 0; i < requests; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if runner.calls != requests || runner.maxActive != 1 {
		t.Fatalf("calls=%d maxActive=%d; want %d, 1", runner.calls, runner.maxActive, requests)
	}
}

func TestChatServiceRejectsUnlicensedSkillAndEmptyMessage(t *testing.T) {
	ctx := context.Background()
	service, _, _, runner, userID := newChatFixture(t)

	// One Chat SKU unlocks every internal skill.
	if _, err := service.CreateSession(ctx, userID, "liki", "", ""); err != nil {
		t.Fatalf("create divination session: %v", err)
	}
	session, err := service.CreateSession(ctx, userID, "liki", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := service.SendMessage(ctx, userID, session.ID, "   ", nil); err != ErrMessageEmpty {
		t.Fatalf("send error = %v, want ErrMessageEmpty", err)
	}
	if runner.calls != 0 {
		t.Fatalf("runner calls = %d, want 0", runner.calls)
	}
	// A legacy product id is never treated as an internal skill.
	if ok, err := service.entitlements.HasSkill(ctx, userID, "mingshu"); err != nil || ok {
		t.Fatalf("legacy skill access = %v, %v; want false, nil", ok, err)
	}
	if _, err := service.runnerFor("mingshu"); err != ErrRunnerUnavailable {
		t.Fatalf("legacy runner = %v, want ErrRunnerUnavailable", err)
	}
}

func TestChatHandlerContract(t *testing.T) {
	ctx := context.Background()
	service, store, _, runner, userID := newChatFixture(t)
	session, err := service.CreateSession(ctx, userID, "liki", "起名", "")
	if err != nil {
		t.Fatal(err)
	}

	user := auth.User{ID: userID, Email: "user@example.com", Status: auth.UserActive}
	handler := NewHandler(service, func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Test-Auth") == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "unauthorized"}})
				return
			}
			next(w, r.WithContext(auth.WithAuth(r.Context(), user, auth.AuthSession{ID: "session"}, "token")))
		}
	}, nil)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, nil)

	request := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		var reader *strings.Reader
		if body == "" {
			reader = strings.NewReader("")
		} else {
			reader = strings.NewReader(body)
		}
		req := httptest.NewRequest(method, path, reader)
		req.Header.Set("X-Test-Auth", "1")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	unauth := httptest.NewRequest(http.MethodGet, "/api/chat/sessions", nil)
	unauthRec := httptest.NewRecorder()
	mux.ServeHTTP(unauthRec, unauth)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthRec.Code)
	}

	created := request(http.MethodPost, "/api/chat/sessions", `{"skill":"liki","title":"新会话"}`)
	if created.Code != http.StatusOK || !strings.Contains(created.Body.String(), `"user_id"`) {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	detail := request(http.MethodGet, "/api/chat/sessions/"+session.ID, "")
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), `"title":"起名"`) {
		t.Fatalf("detail = %d %s", detail.Code, detail.Body.String())
	}
	send := request(http.MethodPost, "/api/chat/sessions/"+session.ID+"/messages", `{"content":"你好"}`)
	if send.Code != http.StatusOK {
		t.Fatalf("send = %d %s", send.Code, send.Body.String())
	}
	if got := send.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("send content type = %q", got)
	}
	for _, want := range []string{`"type":"thinking"`, `"type":"text-delta"`, `"type":"done"`, `"assistant_message"`} {
		if !strings.Contains(send.Body.String(), want) {
			t.Fatalf("send body missing %s: %s", want, send.Body.String())
		}
	}
	messages := request(http.MethodGet, "/api/chat/sessions/"+session.ID+"/messages", "")
	if messages.Code != http.StatusOK || !strings.Contains(messages.Body.String(), `"session_id"`) {
		t.Fatalf("messages = %d %s", messages.Code, messages.Body.String())
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE unified_chat_sessions SET skill='mingshu' WHERE id=?`, session.ID); err != nil {
		t.Fatal(err)
	}
	forbiddenDetail := request(http.MethodGet, "/api/chat/sessions/"+session.ID, "")
	if forbiddenDetail.Code != http.StatusForbidden {
		t.Fatalf("forbidden detail = %d %s", forbiddenDetail.Code, forbiddenDetail.Body.String())
	}
	forbiddenMessages := request(http.MethodGet, "/api/chat/sessions/"+session.ID+"/messages", "")
	if forbiddenMessages.Code != http.StatusForbidden {
		t.Fatalf("forbidden messages = %d %s", forbiddenMessages.Code, forbiddenMessages.Body.String())
	}
	if runner.calls != 1 {
		t.Fatalf("runner calls = %d", runner.calls)
	}
}

func TestChatSessionDetailAccess(t *testing.T) {
	ctx := context.Background()
	service, store, _, _, userID := newChatFixture(t)
	session, err := service.CreateSession(ctx, userID, "liki", "", "")
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := service.Session(ctx, userID, session.ID)
	if err != nil {
		t.Fatalf("active session read err = %v", err)
	}
	if loaded.ID != session.ID {
		t.Fatalf("loaded = %+v", loaded)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE entitlements SET expires_at='2000-01-01T00:00:00Z' WHERE user_id=?`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Session(ctx, userID, session.ID); err != ErrSkillForbidden {
		t.Fatalf("expired session read err = %v, want ErrSkillForbidden", err)
	}
}

func TestAssistantContentPreservesAllToolLoopSegments(t *testing.T) {
	got := assistantContent([]llm.Message{
		{Role: llm.RoleSystem, Content: "system"},
		{Role: llm.RoleAssistant, Content: "先查命盘。"},
		{Role: llm.RoleTool, Content: "{}"},
		{Role: llm.RoleAssistant, Content: "结论。"},
	})
	if got != "先查命盘。\n\n结论。" {
		t.Fatalf("assistant content = %q", got)
	}
}

func TestChatTimestampsRemainLexicographicallyOrdered(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	first := formatTime(base)
	second := formatTime(base.Add(time.Nanosecond))
	if first >= second {
		t.Fatalf("timestamps are not ordered: %q >= %q", first, second)
	}
	if !strings.HasSuffix(first, ".000000000Z") {
		t.Fatalf("timestamp lacks fixed nanoseconds: %q", first)
	}
}

type failingRunner struct {
	events []agent.ChatEvent
}

func (r *failingRunner) Execute(ctx context.Context, msgs []llm.Message, onEvent func(agent.ChatEvent), _ func() string) ([]llm.Message, error) {
	for _, event := range r.events {
		if onEvent != nil {
			onEvent(event)
		}
	}
	return nil, errors.New("llm unavailable")
}

func (r *failingRunner) Prompt() string { return "unified prompt" }

func TestFailedGenerationPersistsUserButNotPartialAssistant(t *testing.T) {
	ctx := context.Background()
	db := openChatDB(t)
	authStore := auth.NewStore(db, []byte("test-hmac-key"))
	if err := authStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	entitlements := entitlement.NewStore(db)
	if err := entitlements.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	userID := "user_platform"
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO users (id,email,status,locale,created_at,updated_at) VALUES (?,?, 'active','zh-Hant',?,?)`, userID, "user@example.com", now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := entitlements.GrantForProduct(ctx, userID, "user@example.com", entitlement.ProductChat, "payment_intent", "pay_1", 24*time.Hour); err != nil {
		t.Fatal(err)
	}
	store := NewStore(db)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	runner := &failingRunner{events: []agent.ChatEvent{
		{Type: agent.EventTextDelta, Content: "半个"},
	}}
	service := NewService(store, entitlements, nil, map[Skill]Runner{"liki": runner})
	session, err := service.CreateSession(ctx, userID, "liki", "", "")
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, err = service.SendMessage(ctx, userID, session.ID, "你好", nil)
	if err == nil {
		t.Fatal("expected generation failure")
	}
	messages, err := store.ListMessages(ctx, userID, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].Role != "user" || messages[0].Content != "你好" {
		t.Fatalf("partial generation persisted: %+v", messages)
	}
}

func TestClosedSessionIsUnavailable(t *testing.T) {
	ctx := context.Background()
	service, store, _, _, userID := newChatFixture(t)
	session, err := service.CreateSession(ctx, userID, "liki", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE unified_chat_sessions SET status='closed' WHERE id=?`, session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Session(ctx, userID, session.ID); err != ErrSessionNotFound {
		t.Fatalf("closed session err = %v, want ErrSessionNotFound", err)
	}
	if _, _, _, err := service.SendMessage(ctx, userID, session.ID, "你好", nil); err != ErrSessionNotFound {
		t.Fatalf("closed session send err = %v, want ErrSessionNotFound", err)
	}
}

func TestChatServiceProfileBinding(t *testing.T) {
	ctx := context.Background()
	_, store, entitlements, runner, userID := newChatFixture(t)

	profileStore := profile.NewStore(store.db)
	if err := profileStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	p, err := profileStore.Create(ctx, userID, profile.Input{
		Nickname: "妈妈", Gender: "female", Calendar: "lunar",
		BirthDate: "1958-闰5-12", BirthTime: "06:00", BirthTimeKnown: "known",
		BirthPlace: "上海",
	})
	if err != nil {
		t.Fatal(err)
	}

	service := NewService(store, entitlements, profileStore, map[Skill]Runner{"liki": runner})
	session, err := service.CreateSession(ctx, userID, "liki", "", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if session.ProfileID != p.ID {
		t.Fatalf("session profile_id = %q, want %q", session.ProfileID, p.ID)
	}

	// First message must be the system-written profile card.
	messages, err := store.ListMessages(ctx, userID, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].Kind != KindSystemProfile {
		t.Fatalf("expected system_profile first message, got %+v", messages)
	}

	// Cross-user profile must be rejected.
	if _, err := service.CreateSession(ctx, "other_user", "liki", "", p.ID); err == nil {
		t.Fatal("cross-user profile binding must fail")
	}

	// System prompt must include the profile user info.
	if _, _, _, err := service.SendMessage(ctx, userID, session.ID, "你好", nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(runner.prompt, "妈妈") || !strings.Contains(runner.prompt, "1958-闰5-12") {
		t.Fatalf("system prompt lacks profile context: %q", runner.prompt)
	}
}

func TestUpdateProfileBirthTimeTool(t *testing.T) {
	ctx := context.Background()

	// Mock the platform API that the tool calls over the network, the same
	// endpoint the frontend uses. It authenticates via the session cookie.
	profileJSON := `{"id":"p_1","user_id":"user_1","version":1,"nickname":"妈妈","name":"","gender":"female","calendar":"lunar","birth_date":"1958-闰5-12","birth_time":"06:00","birth_time_known":"known","birth_place":"上海","note":"","third_party_consent":true}`
	patchedJSON := `{"id":"p_1","user_id":"user_1","version":2,"nickname":"妈妈","name":"","gender":"female","calendar":"lunar","birth_date":"1958-闰5-12","birth_time":"07:00","birth_time_known":"known","birth_place":"上海","note":"","third_party_consent":true}`
	var sentVersion int
	var receivedCookie string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/profiles/p_1", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("liki_session")
		if err != nil || cookie.Value != "token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		receivedCookie = cookie.Value
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"data":` + profileJSON + `}`))
		case http.MethodPatch:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if v, ok := body["expected_version"].(float64); ok {
				sentVersion = int(v)
			}
			_, _ = w.Write([]byte(`{"data":` + patchedJSON + `}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	handler := UpdateProfileBirthTimeHandler(srv.URL, "liki_session", srv.Client())
	args, _ := json.Marshal(map[string]any{"profile_id": "p_1", "birth_time": "07:00", "birth_time_known": "known"})

	// Without a session binding the tool must refuse before any network call.
	if _, err := handler(ctx, args); err == nil {
		t.Fatal("tool must refuse without a session binding")
	}

	authCtx := auth.WithAuth(ctx, auth.User{ID: "user_1"}, auth.AuthSession{ID: "s_1"}, "token")
	toolCtx := withSessionBinding(authCtx, sessionBinding{UserID: "user_1", SessionID: "s_1", ProfileID: "p_1"})
	out, err := handler(toolCtx, args)
	if err != nil {
		t.Fatalf("bound update failed: %v", err)
	}
	if sentVersion != 1 {
		t.Fatalf("expected_version sent = %d, want 1", sentVersion)
	}
	if receivedCookie != "token" {
		t.Fatalf("session cookie not forwarded, got %q", receivedCookie)
	}
	if !strings.Contains(string(out), "07:00") {
		t.Fatalf("tool output missing updated time: %s", out)
	}

	// A profile other than the session-bound one must be rejected before any
	// network call.
	otherArgs, _ := json.Marshal(map[string]any{"profile_id": "other", "birth_time": "05:00", "birth_time_known": "known"})
	if _, err := handler(toolCtx, otherArgs); err == nil {
		t.Fatal("tool must refuse to update a non-bound profile")
	}
}

func TestUpdateProfileBirthTimeToolRateLimit(t *testing.T) {
	ctx := context.Background()
	profileJSON := `{"id":"p_1","user_id":"user_1","version":1,"nickname":"妈妈","gender":"female","calendar":"lunar","birth_date":"1958-闰5-12","birth_time":"06:00","birth_time_known":"known"}`
	mux := http.NewServeMux()
	mux.HandleFunc("/api/profiles/p_1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"data":` + profileJSON + `}`))
		case http.MethodPatch:
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":"rate_limited","message":"too many"}}`))
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	handler := UpdateProfileBirthTimeHandler(srv.URL, "liki_session", srv.Client())
	authCtx := auth.WithAuth(ctx, auth.User{ID: "user_1"}, auth.AuthSession{ID: "s_1"}, "token")
	toolCtx := withSessionBinding(authCtx, sessionBinding{UserID: "user_1", SessionID: "s_1", ProfileID: "p_1"})
	args, _ := json.Marshal(map[string]any{"profile_id": "p_1", "birth_time": "07:00", "birth_time_known": "known"})
	_, err := handler(toolCtx, args)
	if err == nil || !strings.Contains(err.Error(), "太频繁") {
		t.Fatalf("expected rate-limit hint, got %v", err)
	}
}

func TestStore_AnonymizeUserMessages(t *testing.T) {
	ctx := context.Background()
	_, store, _, _, userID := newChatFixture(t)
	session, err := store.CreateSession(ctx, userID, "liki", "标题", "p_1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendMessage(ctx, userID, session.ID, "assistant", KindSystemProfile, `{"nickname":"妈妈","birth_date":"1958-闰5-12"}`); err != nil {
		t.Fatal(err)
	}
	n, err := store.AnonymizeUserMessages(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want 1 message cleared, got %d", n)
	}
	msgs, err := store.ListMessages(ctx, userID, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Content != "" {
		t.Fatalf("system_profile PII not cleared: %+v", msgs)
	}
}
