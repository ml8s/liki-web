package liki

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"liki-web/internal/agent"
	"liki-web/internal/llm"
	"liki-web/internal/profile"
)

var (
	ErrSkillForbidden    = errors.New("liki: skill forbidden")
	ErrMessageEmpty      = errors.New("liki: message empty")
	ErrRunnerUnavailable = errors.New("liki: runner unavailable for skill")
)

// Runner abstracts the LLM execution engine for one skill.
type Runner interface {
	Execute(ctx context.Context, msgs []llm.Message, onEvent func(agent.ChatEvent), phaseMsg func() string) ([]llm.Message, error)
	Prompt() string
}

// Service orchestrates chat sessions with entitlement checks.
type Service struct {
	store        *ChatStore
	entitlements *EntitlementStore
	profiles     *profile.Store
	runners      map[Skill]Runner
	locks        sync.Mutex
	sessionLocks sync.Map // sessionID -> *sync.Mutex
}

func NewService(store *ChatStore, entitlements *EntitlementStore, profiles *profile.Store, runners map[Skill]Runner) *Service {
	if runners == nil {
		runners = make(map[Skill]Runner)
	}
	return &Service{store: store, entitlements: entitlements, profiles: profiles, runners: runners, sessionLocks: sync.Map{}}
}

func (s *Service) runnerFor(skill Skill) (Runner, error) {
	if runner, ok := s.runners[skill]; ok && runner != nil {
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
	ok, err := s.entitlements.HasSkill(ctx, userID, Skill(session.Skill))
	if err != nil {
		return Session{}, err
	}
	if !ok {
		return Session{}, ErrSkillForbidden
	}
	if _, err := s.runnerFor(Skill(session.Skill)); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) lockSession(sessionID string) func() {
	s.locks.Lock()
	lock, _ := s.sessionLocks.LoadOrStore(sessionID, &sync.Mutex{})
	m := lock.(*sync.Mutex)
	s.locks.Unlock()
	m.Lock()
	return func() {
		m.Unlock()
		s.sessionLocks.Delete(sessionID)
	}
}

func (s *Service) CreateSession(ctx context.Context, userID, skill, title, profileID string) (Session, error) {
	ok, err := s.entitlements.HasSkill(ctx, userID, Skill(skill))
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
	ok, err := s.entitlements.HasSkill(ctx, userID, Skill(session.Skill))
	if err != nil {
		return Message{}, Message{}, Session{}, err
	}
	if !ok {
		return Message{}, Message{}, Session{}, ErrSkillForbidden
	}
	chatRunner, err := s.runnerFor(Skill(session.Skill))
	if err != nil {
		return Message{}, Message{}, Session{}, err
	}

	ctx = withSessionBinding(ctx, sessionBinding{UserID: userID, SessionID: session.ID, ProfileID: session.ProfileID})

	_, err = s.store.AppendMessage(ctx, userID, sessionID, "user", KindText, content)
	if err != nil {
		return Message{}, Message{}, Session{}, err
	}
	history, err := s.store.ListMessages(ctx, userID, sessionID)
	if err != nil {
		return Message{}, Message{}, Session{}, err
	}
	userMessage := history[len(history)-1]

	systemPrompt := chatRunner.Prompt()
	if session.ProfileID != "" && s.profiles != nil {
		if p, err := s.profiles.Get(ctx, userID, session.ProfileID); err == nil {
			systemPrompt = profileContext(p) + "\n\n" + systemPrompt
		}
	}

	msgs := make([]llm.Message, 0, len(history)+1)
	msgs = append(msgs, llm.Message{Role: llm.RoleSystem, Content: systemPrompt})
	for _, item := range history {
		if item.Kind == KindSystemProfile {
			continue
		}
		msgs = append(msgs, llm.Message{Role: llm.Role(item.Role), Content: item.Content})
	}

	if onEvent != nil {
		onEvent(agent.ChatEvent{Type: agent.EventThinking})
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
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
func systemProfileContent(p profile.Profile) (string, error) {
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

// profileContext is the user-info block injected into the system prompt so the
// agent computes charts itself. Deliberately no chart/pan payload.
func profileContext(p profile.Profile) string {
	var b strings.Builder
	b.WriteString("系统已载入档案信息（用户信息，供排盘与解读使用）：\n")
	b.WriteString("称呼：" + p.Nickname + "\n")
	if p.Name != "" {
		b.WriteString("姓名：" + p.Name + "\n")
	}
	genderLabel := map[string]string{"male": "男", "female": "女"}[p.Gender]
	if genderLabel == "" {
		genderLabel = "未知"
	}
	b.WriteString("性别：" + genderLabel + "\n")
	b.WriteString("历法：" + p.Calendar + "\n")
	b.WriteString("出生日期：" + p.BirthDate + "\n")
	b.WriteString("出生时间：" + p.BirthTime + "（" + p.BirthTimeKnown + "）\n")
	if p.BirthPlace != "" {
		b.WriteString("出生地：" + p.BirthPlace + "\n")
	}
	if p.Note != "" {
		b.WriteString("用户备注：" + p.Note + "\n")
	}
	b.WriteString("请基于以上出生信息，使用排盘工具计算命盘后再进行解读。")
	return b.String()
}
