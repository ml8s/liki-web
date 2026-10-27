package chat

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/agent"
	"github.com/ml8s/liki-web/internal/birthprofile"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/llm"
	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

type scriptedChatRunner struct {
	prompt   string
	result   ExecutionResult
	err      error
	calls    int
	lastMsgs []llm.Message
}

func (r *scriptedChatRunner) Prompt() string { return r.prompt }

func (r *scriptedChatRunner) Execute(_ context.Context, msgs []llm.Message, onEvent func(agent.ChatEvent)) (ExecutionResult, error) {
	r.calls++
	r.lastMsgs = msgs
	if onEvent != nil {
		onEvent(agent.ChatEvent{Type: agent.EventTextDelta, Content: "stream"})
	}
	if r.err != nil {
		return ExecutionResult{}, r.err
	}
	return r.result, nil
}

func newAgentModeTestService(t *testing.T) (*Service, *ChatStore, *scriptedChatRunner) {
	t.Helper()
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	entitlements := entitlement.NewStore(db)
	if err := entitlements.Migrate(ctx); err != nil {
		t.Fatalf("migrate entitlements: %v", err)
	}
	if _, err := entitlements.GrantForProduct(ctx, "user-1", "user@example.com", product.Chat, "test", "test-1", 24*time.Hour); err != nil {
		t.Fatalf("grant entitlement: %v", err)
	}
	profiles := birthprofile.NewStore(db)
	if err := profiles.Migrate(ctx); err != nil {
		t.Fatalf("migrate profiles: %v", err)
	}
	chats := NewChatStore(db)
	if err := chats.Migrate(ctx); err != nil {
		t.Fatalf("migrate chats: %v", err)
	}

	service := NewService(chats, entitlements, profiles)
	remote := &scriptedChatRunner{result: ExecutionResult{Messages: []llm.Message{{Role: llm.RoleAssistant, Content: "agent answer"}}}}
	service.SetAgentRunners(map[string]Runner{ExpertNatal: remote})
	return service, chats, remote
}

func TestServiceAgentModePersistsAuthoritativeAgentResult(t *testing.T) {
	service, chats, remote := newAgentModeTestService(t)
	ctx := context.Background()
	session, err := service.CreateSession(ctx, "user-1", string(product.SkillConsultation), ExpertNatal, "test", "")
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
	if remote.calls != 1 {
		t.Fatalf("runner calls remote=%d", remote.calls)
	}
	if len(remote.lastMsgs) == 0 || remote.lastMsgs[0].Role != llm.RoleUser {
		t.Fatalf("agent runner first message = %+v", remote.lastMsgs)
	}
	messages, _, err := chats.ListMessages(ctx, "user-1", session.ID, "", 50)
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
	service, chats, remote := newAgentModeTestService(t)
	remote.err = errors.New("engine unavailable")
	ctx := context.Background()
	session, err := service.CreateSession(ctx, "user-1", string(product.SkillConsultation), ExpertNatal, "test", "")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	_, _, _, err = service.SendMessage(ctx, "user-1", session.ID, "hello", nil)
	if err == nil || err.Error() != "engine unavailable" {
		t.Fatalf("SendMessage error = %v", err)
	}
	if remote.calls != 1 {
		t.Fatalf("runner calls remote=%d", remote.calls)
	}
	messages, _, err := chats.ListMessages(ctx, "user-1", session.ID, "", 50)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 1 || messages[0].Role != "user" {
		t.Fatalf("messages = %+v", messages)
	}
}

func TestServiceAgentModeRoutesProfileBoundSessionToAgentRunner(t *testing.T) {
	service, chats, remote := newAgentModeTestService(t)
	ctx := context.Background()
	profileStore := service.profiles
	created, err := profileStore.Create(ctx, "user-1", birthprofile.BirthProfileDraft{
		Nickname:          "Test",
		ThirdPartyConsent: true,
		Birth: birthprofile.BirthFactsInput{
			Gender: "male", Date: "1990-06-01", Time: "12:00",
			Location: birthprofile.BirthLocationInput{
				Place:     "北京",
				Timezone:  &[]string{"Asia/Shanghai"}[0],
				Longitude: &[]float64{116.4074}[0],
			},
		},
	})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	session, err := service.CreateSession(ctx, "user-1", string(product.SkillConsultation), ExpertNatal, "test", created.ID)
	if err != nil {
		t.Fatalf("create profile session: %v", err)
	}

	_, assistant, _, err := service.SendMessage(ctx, "user-1", session.ID, "hello", nil)
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if assistant.Content != "agent answer" {
		t.Fatalf("assistant = %+v", assistant)
	}
	if remote.calls != 1 {
		t.Fatalf("runner calls remote=%d", remote.calls)
	}
	messages, _, err := chats.ListMessages(ctx, "user-1", session.ID, "", 50)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 3 {
		t.Fatalf("message count = %d, want user/profile/assistant", len(messages))
	}
}
