package chat

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
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
)

// chatExecutionTimeout must remain below the HTTP server WriteTimeout. The
// service timeout covers local tool loops and remote AG-UI runs alike.
const chatExecutionTimeout = 600 * time.Second

// Runner abstracts the LLM execution engine for one skill.
type Runner interface {
	Execute(ctx context.Context, msgs []llm.Message, onEvent func(agent.ChatEvent), phaseMsg func() string) ([]llm.Message, error)
}

// Service orchestrates chat sessions with entitlement checks.
type Service struct {
	store        *ChatStore
	authorizer   AccessAuthorizer
	profiles     *birthprofile.Store
	agentRunner  Runner
	locks        sync.Mutex
	sessionLocks sync.Map // sessionID -> *sync.Mutex
}

func NewService(store *ChatStore, authorizer AccessAuthorizer, profiles *birthprofile.Store) *Service {
	return &Service{store: store, authorizer: authorizer, profiles: profiles, sessionLocks: sync.Map{}}
}

// SetAgentRunner enables AG-UI routing for all chat sessions. The in-process
// local runner has been retired; every session (profile-bound or not) is
// executed by the remote liki-agents runtime.
func (s *Service) SetAgentRunner(runner Runner) {
	s.agentRunner = runner
}

func (s *Service) runnerForSession(skill product.Skill, profileID string) (Runner, error) {
	if s.agentRunner != nil {
		return s.agentRunner, nil
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
	if _, err := s.runnerForSession(product.Skill(session.Skill), session.ProfileID); err != nil {
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

func (s *Service) CreateSession(ctx context.Context, userID, skill, title, profileID string) (Session, error) {
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
	session, err := s.store.CreateSession(ctx, userID, skill, title, profileID)
	if err != nil {
		return Session{}, err
	}
	if profileID != "" && s.profiles != nil {
		if err := s.profiles.TouchLastUsed(ctx, userID, profileID); err != nil {
			slog.Warn("liki: touch profile last_used", "profile_id", profileID, "err", err)
		}
		if p, err := s.profiles.Get(ctx, userID, profileID); err == nil {
			if content, err := systemProfileContent(p); err == nil {
				if _, err := s.store.AppendMessage(ctx, userID, session.ID, "assistant", KindSystemProfile, content); err != nil {
					slog.Warn("liki: write system_profile message", "session_id", session.ID, "err", err)
				}
			}
		}
	}
	return session, nil
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
	chatRunner, err := s.runnerForSession(product.Skill(session.Skill), session.ProfileID)
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
		binding.Profile = &AgentProfileContext{
			Calendar:       p.Calendar,
			Gender:         p.Gender,
			BirthDate:      p.BirthDate,
			BirthTime:      p.BirthTime,
			BirthTimeKnown: p.BirthTimeKnown,
			BirthPlace:     p.BirthPlace,
		}
	}
	ctx = withSessionBinding(ctx, binding)

	_, err = s.store.AppendMessage(ctx, userID, sessionID, "user", KindText, content)
	if err != nil {
		return Message{}, Message{}, Session{}, err
	}
	history, err := s.store.ListMessages(ctx, userID, sessionID)
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
	result, err := chatRunner.Execute(ctx, msgs, onEvent, func() string {
		return "正在计算命理数据…"
	})
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
		"nickname":            p.Nickname,
		"name":                p.Name,
		"gender":              p.Gender,
		"calendar":            p.Calendar,
		"birth_date":          p.BirthDate,
		"birth_time":          p.BirthTime,
		"birth_time_known":    p.BirthTimeKnown,
		"birth_place":         p.BirthPlace,
		"third_party_consent": p.ThirdPartyConsent,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
