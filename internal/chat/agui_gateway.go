package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"

	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
	aguisse "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"

	"github.com/ml8s/liki-web/internal/agent"
	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
)

// HandleAGUIMessage exposes the existing product conversation behind a
// browser-safe AG-UI endpoint. It is deliberately not a proxy to liki-agent:
// liki-web performs auth, entitlement, quota, persistence, and verified identity
// injection before any agent traffic is generated.
func (h *Handler) HandleAGUIMessage(w http.ResponseWriter, r *http.Request) {
	input, ok := httputil.Decode[aguitypes.RunAgentInput](w, r)
	if !ok {
		return
	}
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	sessionID := r.PathValue("id")
	if input.ThreadID != "" && input.ThreadID != sessionID {
		httputil.RespondError(w, http.StatusUnprocessableEntity, "invalid_request", "threadId must match the session")
		return
	}
	content, err := latestAGUIUserText(input)
	if err != nil {
		httputil.RespondError(w, http.StatusUnprocessableEntity, "invalid_request", err.Error())
		return
	}
	if _, err := h.service.Session(r.Context(), user.ID, sessionID); err != nil {
		writeSessionError(r.Context(), w, err)
		return
	}
	requestCtx := withChatLocale(r.Context(), localeOrDefault(firstNonEmpty(
		localeFromForwardedProps(input.ForwardedProps),
		user.Locale,
	)))

	if _, ok := httputil.BeginSSE(w); !ok {
		httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "服务器不支持流式响应")
		return
	}
	gateway := newAGUIGateway(w, r.Context(), sessionID, input.RunID)
	gateway.start()
	userMessage, assistantMessage, session, runErr := h.service.SendMessage(
		requestCtx, user.ID, sessionID, content,
		gateway.translate,
	)
	if runErr == nil {
		gateway.finish(SendMessageResult{
			Session:          session,
			ActiveSkill:      session.Skill,
			UserMessage:      userMessage,
			AssistantMessage: assistantMessage,
		})
		return
	}
	if !errors.Is(runErr, context.Canceled) {
		slog.ErrorContext(r.Context(), "liki: send agent message", "session_id", sessionID, "err", runErr)
	}
	gateway.fail("服务暂时不可用，请稍后重试")
}

func localeFromForwardedProps(value any) string {
	props, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	locale, _ := props["locale"].(string)
	return locale
}

type aguiGateway struct {
	writer      *aguisse.SSEWriter
	w           http.ResponseWriter
	ctx         context.Context
	sessionID   string
	runID       string
	messageID   string
	textStarted bool
	failed      bool
}

func newAGUIGateway(
	w http.ResponseWriter,
	ctx context.Context,
	sessionID string,
	runID string,
) *aguiGateway {
	if strings.TrimSpace(runID) == "" {
		runID = uuid.NewString()
	}
	return &aguiGateway{
		writer:    aguisse.NewSSEWriter(),
		w:         w,
		ctx:       ctx,
		sessionID: sessionID,
		runID:     runID,
		messageID: runID + ":assistant",
	}
}

func (g *aguiGateway) emit(event aguievents.Event) {
	if err := event.Validate(); err != nil {
		slog.ErrorContext(g.ctx, "liki: invalid browser AG-UI event", "err", err)
		return
	}
	if err := g.writer.WriteEvent(g.ctx, g.w, event); err != nil {
		slog.WarnContext(g.ctx, "liki: write browser AG-UI event", "err", err)
	}
}

func (g *aguiGateway) start() {
	g.emit(aguievents.NewRunStartedEvent(g.sessionID, g.runID))
}

// 浏览器自定义事件采用反域名命名；载荷只承载事实（工具名/信号），
// 用户可见文案一律由前端 i18n 渲染。
const (
	browserPhaseEvent = "hk.liki.phase"
)

// translate 把内部 ChatEvent 映射为浏览器 AG-UI 事件：
// phase 只带原始工具名；EventThinking 不产生浏览器事件（前端自带本地化默认
// 文案，推理内容不外发）。
func (g *aguiGateway) translate(event agent.ChatEvent) {
	switch event.Type {
	case agent.EventPhase:
		g.phase(event.Content)
	case agent.EventTextDelta:
		g.text(event.Content)
	default:
	}
}

func (g *aguiGateway) phase(tool string) {
	if strings.TrimSpace(tool) == "" {
		return
	}
	g.custom(browserPhaseEvent, tool)
}

func (g *aguiGateway) custom(name, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	g.emit(aguievents.NewCustomEvent(name, aguievents.WithValue(value)))
}

func (g *aguiGateway) text(delta string) {
	if delta == "" {
		return
	}
	if !g.textStarted {
		if err := g.emitStarted(); err != nil {
			return
		}
	}
	g.emit(aguievents.NewTextMessageContentEvent(g.messageID, delta))
}

func (g *aguiGateway) emitStarted() error {
	if err := aguievents.NewTextMessageStartEvent(g.messageID, aguievents.WithRole(string(aguitypes.RoleAssistant))).Validate(); err != nil {
		return err
	}
	g.textStarted = true
	g.emit(aguievents.NewTextMessageStartEvent(g.messageID, aguievents.WithRole(string(aguitypes.RoleAssistant))))
	return nil
}

func (g *aguiGateway) finish(result SendMessageResult) {
	if g.failed {
		return
	}
	if g.textStarted {
		g.textStarted = false
		g.emit(aguievents.NewTextMessageEndEvent(g.messageID))
	}
	finished := aguievents.NewRunFinishedEventWithOptions(
		g.sessionID,
		g.runID,
		aguievents.WithSuccessOutcome(),
	)
	finished.Result = map[string]any{
		"session":           result.Session,
		"active_skill":      result.ActiveSkill,
		"user_message":      result.UserMessage,
		"assistant_message": result.AssistantMessage,
	}
	g.emit(finished)
}

func (g *aguiGateway) fail(message string) {
	if g.failed {
		return
	}
	g.failed = true
	if strings.TrimSpace(message) == "" {
		message = "服务暂时不可用，请稍后重试"
	}
	if g.textStarted {
		g.textStarted = false
		g.emit(aguievents.NewTextMessageEndEvent(g.messageID))
	}
	g.emit(aguievents.NewRunErrorEvent(message, aguievents.WithRunID(g.runID)))
}

func latestAGUIUserText(input aguitypes.RunAgentInput) (string, error) {
	for _, message := range slices.Backward(input.Messages) {

		if message.Role != aguitypes.RoleUser {
			continue
		}
		switch content := message.Content.(type) {
		case string:
			content = strings.TrimSpace(content)
			if content == "" {
				return "", fmt.Errorf("latest user message is empty")
			}
			return content, nil
		default:
			return "", fmt.Errorf("latest user message must contain text")
		}
	}
	return "", fmt.Errorf("AG-UI request must contain a user message")
}
