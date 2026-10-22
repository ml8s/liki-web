package chat

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ml8s/liki-web/internal/agent"
	"github.com/ml8s/liki-web/internal/birthprofile"
	"github.com/ml8s/liki-web/internal/llm"
	"github.com/ml8s/liki-web/internal/product"
)

var (
	ErrSkillForbidden    = errors.New("liki: skill forbidden")
	ErrMessageEmpty      = errors.New("liki: message empty")
	ErrRunnerUnavailable = errors.New("liki: runner unavailable for skill")
	ErrExpertInvalid     = errors.New("liki: invalid expert")
)

// Expert identifies one independently deployed expert app (ADK deployment).
// Every session binds exactly one expert; the runner is chosen by this value.
const (
	ExpertNatal      = "natal"
	ExpertDivination = "divination"
	ExpertFengshui   = "fengshui"
	ExpertNaming     = "naming"
)

// ExpertOrder is the product-facing expert order.
var ExpertOrder = []string{ExpertNatal, ExpertDivination, ExpertFengshui, ExpertNaming}

// ValidExpert reports whether expert names a deployed expert app.
func ValidExpert(expert string) bool {
	return slices.Contains(ExpertOrder, expert)
}

// chatExecutionTimeout must remain below the HTTP server WriteTimeout. The
// service timeout covers local tool loops and remote AG-UI runs alike.
const chatExecutionTimeout = 600 * time.Second

// Runner abstracts the LLM execution engine for one expert deployment.
type Runner interface {
	Execute(ctx context.Context, msgs []llm.Message, onEvent func(agent.ChatEvent)) ([]llm.Message, error)
}

// Service orchestrates chat sessions with entitlement checks.
type Service struct {
	store        *ChatStore
	authorizer   AccessAuthorizer
	profiles     *birthprofile.Store
	agentRunners map[string]Runner
	locks        sync.Mutex
	sessionLocks sync.Map // sessionID -> *sync.Mutex
}

func NewService(store *ChatStore, authorizer AccessAuthorizer, profiles *birthprofile.Store) *Service {
	return &Service{store: store, authorizer: authorizer, profiles: profiles, sessionLocks: sync.Map{}}
}

// SetAgentRunners enables AG-UI routing per expert app. The in-process local
// runner has been retired; every session executes on the remote
// liki-agent-runtime deployment selected by the session's expert. Sessions
// without an expert (pre-expert history) are unavailable.
func (s *Service) SetAgentRunners(runners map[string]Runner) {
	s.agentRunners = runners
}

func (s *Service) runnerForSession(expert string) (Runner, error) {
	if runner, ok := s.agentRunners[expert]; ok && runner != nil {
		return runner, nil
	}
	return nil, ErrRunnerUnavailable
}

// Session loads a session only while its entitlement remains active. Expired
// products lose both write and read access to the conversation.
func (s *Service) Session(ctx context.Context, userID, sessionID string) (Session, error) {
	session, err := s.store.GetSession(ctx, userID, sessionID)
	if err != nil {
		return Session{}, err
	}
	if session.Status != SessionActive {
		return Session{}, ErrSessionNotFound
	}
	ok, err := s.authorizer.HasSkill(ctx, userID, product.Skill(session.Skill))
	if err != nil {
		return Session{}, err
	}
	if !ok {
		return Session{}, ErrSkillForbidden
	}
	if _, err := s.runnerForSession(session.Expert); err != nil {
		return Session{}, err
	}
	return session, nil
}

// refcountedMutex pairs a sync.Mutex with a reference count so a session lock
// can be safely removed from the map once no goroutine holds or waits on it.
// Removing a lock after Unlock would let two goroutines observe different
// mutexes for the same session; refcounting avoids both that race and an
// unbounded map.
type refcountedMutex struct {
	mu      sync.Mutex
	refs    int
	removed bool
}

func (s *Service) lockSession(sessionID string) func() {
	s.locks.Lock()
	entry := s.acquireRef(sessionID)
	s.locks.Unlock()

	entry.mu.Lock()
	return func() {
		entry.mu.Unlock()

		s.locks.Lock()
		entry.refs--
		if entry.refs == 0 && !entry.removed {
			entry.removed = true
			s.sessionLocks.Delete(sessionID)
		}
		s.locks.Unlock()
	}
}

// acquireRef returns the refcounted mutex for sessionID with refs incremented.
// Caller must hold s.locks. It is safe to reuse an entry that is being torn
// down only while refs > 0; a new goroutine arriving after Delete observes a
// fresh entry, which serializes against the old one because the old one has
// refs == 0 (no holder, no waiter) at that point.
func (s *Service) acquireRef(sessionID string) *refcountedMutex {
	raw, ok := s.sessionLocks.Load(sessionID)
	if ok {
		entry := raw.(*refcountedMutex)
		entry.refs++
		return entry
	}
	entry := &refcountedMutex{}
	loaded, _ := s.sessionLocks.LoadOrStore(sessionID, entry)
	loadedEntry := loaded.(*refcountedMutex)
	loadedEntry.refs++
	return loadedEntry
}

func (s *Service) CreateSession(ctx context.Context, userID, skill, expert, title, profileID string) (Session, error) {
	if !ValidExpert(expert) {
		return Session{}, ErrExpertInvalid
	}
	ok, err := s.authorizer.HasSkill(ctx, userID, product.Skill(skill))
	if err != nil {
		return Session{}, err
	}
	if !ok {
		return Session{}, ErrSkillForbidden
	}
	if title == "" {
		title = strings.TrimSpace(skill) + " consultation"
	}
	if profileID != "" && s.profiles != nil {
		if _, err := s.profiles.Get(ctx, userID, profileID); err != nil {
			return Session{}, err
		}
	}
	session, err := s.store.CreateSession(ctx, userID, skill, expert, title, profileID)
	if err != nil {
		return Session{}, err
	}
	if profileID != "" && s.profiles != nil {
		if err := s.profiles.TouchLastUsed(ctx, userID, profileID); err != nil {
			slog.WarnContext(ctx, "liki: touch profile last_used", "profile_id", profileID, "err", err)
		}
		if p, err := s.profiles.Get(ctx, userID, profileID); err == nil {
			if content, err := systemProfileContent(p); err == nil {
				if _, err := s.store.AppendMessage(ctx, userID, session.ID, "assistant", KindSystemProfile, content); err != nil {
					slog.WarnContext(ctx, "liki: write system_profile message", "session_id", session.ID, "err", err)
				}
			}
		}
	}
	return session, nil
}

// RenameSession renames a user-owned session. Renaming is metadata maintenance
// and stays available after a product expires: the session list keeps showing
// past sessions, so users must be able to curate them.
func (s *Service) RenameSession(ctx context.Context, userID, sessionID, title string) (Session, error) {
	return s.store.RenameSession(ctx, userID, sessionID, title)
}

// DeleteSession soft-deletes a user-owned session regardless of entitlement,
// mirroring RenameSession.
func (s *Service) DeleteSession(ctx context.Context, userID, sessionID string) error {
	return s.store.SoftDeleteSession(ctx, userID, sessionID)
}

func (s *Service) SendMessage(ctx context.Context, userID, sessionID, content string, onEvent func(agent.ChatEvent)) (Message, Message, Session, error) {
	unlock := s.lockSession(sessionID)
	defer unlock()

	content = strings.TrimSpace(content)
	if content == "" {
		return Message{}, Message{}, Session{}, ErrMessageEmpty
	}
	session, err := s.store.GetSession(ctx, userID, sessionID)
	if err != nil {
		return Message{}, Message{}, Session{}, err
	}
	if session.Status != SessionActive {
		return Message{}, Message{}, Session{}, ErrSessionNotFound
	}
	ok, err := s.authorizer.HasSkill(ctx, userID, product.Skill(session.Skill))
	if err != nil {
		return Message{}, Message{}, Session{}, err
	}
	if !ok {
		return Message{}, Message{}, Session{}, ErrSkillForbidden
	}
	chatRunner, err := s.runnerForSession(session.Expert)
	if err != nil {
		return Message{}, Message{}, Session{}, err
	}

	binding := sessionBinding{
		UserID:    userID,
		SessionID: session.ID,
		ProfileID: session.ProfileID,
	}
	if session.ProfileID != "" && s.profiles != nil {
		p, err := s.profiles.Get(ctx, userID, session.ProfileID)
		if err != nil {
			return Message{}, Message{}, Session{}, err
		}
		binding.Profile = &AgentProfile{
			Gender:  p.Birth.Gender,
			Instant: p.Birth.Instant,
			Location: &AgentBirthLocation{
				Longitude: p.Birth.Location.Longitude,
				Timezone:  p.Birth.Location.Timezone,
			},
		}
	}
	ctx = withSessionBinding(ctx, binding)

	_, err = s.store.AppendMessage(ctx, userID, sessionID, "user", KindText, content)
	if err != nil {
		return Message{}, Message{}, Session{}, err
	}
	// 代理上下文只取最近 agentHistoryLimit 条：会话无限增长时控制 token 成本与延迟。
	history, _, err := s.store.ListMessages(ctx, userID, sessionID, "", agentHistoryLimit)
	if err != nil {
		return Message{}, Message{}, Session{}, err
	}
	userMessage := history[len(history)-1]

	msgs := make([]llm.Message, 0, len(history)+1)
	for _, item := range history {
		if item.Kind == KindSystemProfile {
			continue
		}
		msgs = append(msgs, llm.Message{Role: llm.Role(item.Role), Content: item.Content})
	}

	if onEvent != nil {
		onEvent(agent.ChatEvent{Type: agent.EventThinking})
	}

	ctx, cancel := context.WithTimeout(ctx, chatExecutionTimeout)
	defer cancel()
	result, err := chatRunner.Execute(ctx, msgs, onEvent)
	if err != nil {
		return Message{}, Message{}, Session{}, err
	}

	response := assistantContent(result)
	if response == "" {
		return Message{}, Message{}, Session{}, errors.New("liki: agent returned no assistant response")
	}
	responseMsg, err := s.store.AppendMessage(ctx, userID, sessionID, "assistant", KindText, response)
	if err != nil {
		return Message{}, Message{}, Session{}, err
	}
	_ = s.store.TouchSession(ctx, sessionID, time.Now().UTC())
	session, err = s.store.GetSession(ctx, userID, sessionID)
	if err != nil {
		return Message{}, Message{}, Session{}, err
	}
	return userMessage, responseMsg, session, nil
}

func assistantContent(messages []llm.Message) string {
	var out strings.Builder
	for _, message := range messages {
		if message.Role != llm.RoleAssistant || strings.TrimSpace(message.Content) == "" {
			continue
		}
		if out.Len() > 0 {
			out.WriteString("\n\n")
		}
		out.WriteString(message.Content)
	}
	return out.String()
}

// systemProfileContent builds the persisted first message shown as a profile
// card. Structured JSON so the frontend can render it without parsing prose.
func systemProfileContent(p birthprofile.BirthProfile) (string, error) {
	payload := map[string]any{
		"nickname": p.Nickname,
		"name":     p.Name,
		"birth": map[string]any{
			"gender":  p.Birth.Gender,
			"instant": p.Birth.Instant,
			"date":    p.Birth.Date,
			"time":    p.Birth.Time,
			"location": map[string]any{
				"place":     p.Birth.Location.Place,
				"longitude": p.Birth.Location.Longitude,
				"timezone":  p.Birth.Location.Timezone,
			},
		},
		"third_party_consent": p.ThirdPartyConsent,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
