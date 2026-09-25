package liki

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/agent"
	"github.com/ml8s/liki-web/internal/llm"
	"github.com/ml8s/liki-web/internal/profile"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

type scriptedChatRunner struct {
	prompt   string
	result   []llm.Message
	err      error
	calls    int
	lastMsgs []llm.Message
}

func (r *scriptedChatRunner) Prompt() string { return r.prompt }

func (r *scriptedChatRunner) Execute(_ context.Context, msgs []llm.Message, onEvent func(agent.ChatEvent), _ func() string) ([]llm.Message, error) {
	r.calls++
	r.lastMsgs = msgs
	if onEvent != nil {
		onEvent(agent.ChatEvent{Type: agent.EventTextDelta, Content: "stream"})
	}
	if r.err != nil {
		return nil, r.err
	}
	return r.result, nil
}

func newAgentModeTestService(t *testing.T) (*Service, *ChatStore, *scriptedChatRunner, *scriptedChatRunner) {
	t.Helper()
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	entitlements := NewEntitlementStore(db)
	if err := entitlements.Migrate(ctx); err != nil {
		t.Fatalf("migrate entitlements: %v", err)
	}
	if _, err := entitlements.GrantForProduct(ctx, "user-1", "user@example.com", ProductChat, "test", "test-1", 24*time.Hour); err != nil {
		t.Fatalf("grant entitlement: %v", err)
	}
	profiles := profile.NewStore(db)
	if err := profiles.Migrate(ctx); err != nil {
		t.Fatalf("migrate profiles: %v", err)
	}
	chats := NewChatStore(db)
	if err := chats.Migrate(ctx); err != nil {
		t.Fatalf("migrate chats: %v", err)
	}

	local := &scriptedChatRunner{result: []llm.Message{{Role: llm.RoleAssistant, Content: "local answer"}}}
	service := NewService(chats, entitlements, profiles, map[Skill]Runner{SkillLiki: local})
	remote := &scriptedChatRunner{result: []llm.Message{{Role: llm.RoleAssistant, Content: "agent answer"}}}
	service.SetAgentRunner(remote)
	return service, chats, local, remote
}

func TestServiceAgentModePersistsAuthoritativeAgentResult(t *testing.T) {
	service, chats, local, remote := newAgentModeTestService(t)
	ctx := context.Background()
	session, err := service.CreateSession(ctx, "user-1", string(SkillLiki), "test", "")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	events := make([]agent.ChatEvent, 0, 2)
	_, assistant, _, err := service.SendMessage(ctx, "user-1", session.ID, "hello", func(event agent.ChatEvent) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if assistant.Content != "agent answer" {
		t.Fatalf("assistant = %+v", assistant)
	}
	if local.calls != 0 || remote.calls != 1 {
		t.Fatalf("runner calls local=%d remote=%d", local.calls, remote.calls)
	}
	if remote.lastMsgs[0].Role == llm.RoleSystem && remote.lastMsgs[0].Content != "" {
		t.Fatalf("agent runner received prompt: %+v", remote.lastMsgs[0])
	}
	messages, err := chats.ListMessages(ctx, "user-1", session.ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 2 || messages[0].Role != "user" || messages[1].Content != "agent answer" {
		t.Fatalf("messages = %+v", messages)
	}
	if len(events) != 2 || events[0].Type != agent.EventThinking || events[1].Type != agent.EventTextDelta {
		t.Fatalf("events = %+v", events)
	}
}

func TestServiceAgentModeLeavesUserMessageWithoutAssistantOnRunError(t *testing.T) {
	service, chats, local, remote := newAgentModeTestService(t)
	remote.err = errors.New("engine unavailable")
	ctx := context.Background()
	session, err := service.CreateSession(ctx, "user-1", string(SkillLiki), "test", "")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	_, _, _, err = service.SendMessage(ctx, "user-1", session.ID, "hello", nil)
	if err == nil || err.Error() != "engine unavailable" {
		t.Fatalf("SendMessage error = %v", err)
	}
	if local.calls != 0 || remote.calls != 1 {
		t.Fatalf("runner calls local=%d remote=%d", local.calls, remote.calls)
	}
	messages, err := chats.ListMessages(ctx, "user-1", session.ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 1 || messages[0].Role != "user" {
		t.Fatalf("messages = %+v", messages)
	}
}

func TestServiceAgentModeRoutesProfileBoundSessionToLocalRunner(t *testing.T) {
	service, chats, local, remote := newAgentModeTestService(t)
	ctx := context.Background()
	profileStore := service.profiles
	created, err := profileStore.Create(ctx, "user-1", profile.Input{
		Nickname:       "Test",
		Gender:         "male",
		Calendar:       "solar",
		BirthDate:      "1990-06-01",
		BirthTimeKnown: "unknown",
	})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	session, err := service.CreateSession(ctx, "user-1", string(SkillLiki), "test", created.ID)
	if err != nil {
		t.Fatalf("create profile session: %v", err)
	}

	_, assistant, _, err := service.SendMessage(ctx, "user-1", session.ID, "hello", nil)
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if assistant.Content != "local answer" {
		t.Fatalf("assistant = %+v", assistant)
	}
	if local.calls != 1 || remote.calls != 0 {
		t.Fatalf("runner calls local=%d remote=%d", local.calls, remote.calls)
	}
	messages, err := chats.ListMessages(ctx, "user-1", session.ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 3 {
		t.Fatalf("message count = %d, want user/profile/assistant", len(messages))
	}
}
