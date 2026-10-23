package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	aguiclient "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/client/sse"
	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	aguitypes "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"

	"github.com/ml8s/liki-web/internal/agent"
	"github.com/ml8s/liki-web/internal/llm"
)

const agentProtocolPath = "/ag-ui"

// AgentRunnerConfig configures the server-to-server AG-UI client. The browser
// never receives these values; liki-web remains the only caller of liki-agent.
type AgentRunnerConfig struct {
	BaseURL string
	Token   string
	Timeout time.Duration
	Product string
	Locale  string
}

// AgentRunner adapts liki-web's existing chat Runner interface to liki-agent's
// official AG-UI SSE protocol. It owns no prompts, tools, product database, or
// entitlement policy: those remain behind the liki-web BFF boundary.
type AgentRunner struct {
	client  *aguiclient.Client
	decoder *aguievents.EventDecoder
	config  AgentRunnerConfig
}

func NewAgentRunner(config AgentRunnerConfig) (*AgentRunner, error) {
	if strings.TrimSpace(config.BaseURL) == "" {
		return nil, fmt.Errorf("liki: agent base URL is required")
	}
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Minute
	}
	if config.Product == "" {
		config.Product = "liki"
	}
	if config.Locale == "" {
		config.Locale = "zh-Hant"
	}
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logger.SetLevel(logrus.ErrorLevel)

	return &AgentRunner{
		client: aguiclient.NewClient(aguiclient.Config{
			Endpoint:       endpointFor(config.BaseURL),
			APIKey:         config.Token,
			ConnectTimeout: 10 * time.Second,
			ReadTimeout:    config.Timeout,
			Logger:         logger,
		}),
		decoder: aguievents.NewEventDecoder(nil),
		config:  config,
	}, nil
}

// Execute sends the persisted conversation to liki-agent and translates only
// the AG-UI events used by the existing product SSE interface.
func (r *AgentRunner) Execute(
	ctx context.Context,
	msgs []llm.Message,
	onEvent func(agent.ChatEvent),
) ([]llm.Message, error) {
	binding, _ := sessionBindingFrom(ctx)
	locale := chatLocaleFrom(ctx)
	if locale == "" {
		locale = r.config.Locale
	}
	input, err := r.newRunInput(msgs, binding, locale)
	if err != nil {
		return nil, err
	}

	frames, errorCh, err := r.client.Stream(aguiclient.StreamOptions{
		Context: ctx,
		Payload: input,
		Headers: map[string]string{
			"X-Liki-User-ID": binding.UserID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("liki: start agent run: %w", err)
	}

	state := agentRunState{}
	for frame := range frames {
		// The official HTTP client exposes raw data frames while the official
		// event decoder expects the SSE event name. AG-UI redundantly places
		// the same discriminator in JSON, so read it without hand-rolling SSE.
		var envelope struct {
			Type aguievents.EventType `json:"type"`
		}
		if err := json.Unmarshal(frame.Data, &envelope); err != nil {
			return nil, fmt.Errorf("liki: identify agent event: %w", err)
		}
		event, decodeErr := r.decoder.DecodeEvent(string(envelope.Type), frame.Data)
		if decodeErr != nil {
			return nil, fmt.Errorf("liki: decode agent event: %w", decodeErr)
		}
		if err := state.observe(event, onEvent); err != nil {
			return nil, err
		}
		if state.done {
			break
		}
	}
	// The official stream closes the error channel after the frame channel.
	if streamErr := <-errorCh; streamErr != nil {
		return nil, fmt.Errorf("liki: agent stream: %w", streamErr)
	}
	if err := state.finalError(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	answer := strings.TrimSpace(state.answer)
	if answer == "" {
		return nil, fmt.Errorf("liki: agent returned no answer")
	}
	return []llm.Message{{Role: llm.RoleAssistant, Content: answer}}, nil
}

func (r *AgentRunner) newRunInput(
	msgs []llm.Message,
	binding sessionBinding,
	locale string,
) (aguitypes.RunAgentInput, error) {
	threadID := binding.SessionID
	if threadID == "" {
		threadID = uuid.NewString()
	}
	forwardedProps := map[string]any{
		"product": r.config.Product,
		"locale":  locale,
	}
	if binding.Profile != nil {
		forwardedProps["profile"] = binding.Profile
	}
	if len(binding.ConsultationContext) > 0 {
		var consultationContext map[string]any
		if err := json.Unmarshal(binding.ConsultationContext, &consultationContext); err != nil {
			return aguitypes.RunAgentInput{}, fmt.Errorf("liki: decode consultation context: %w", err)
		}
		forwardedProps["consultation_context"] = consultationContext
	}
	input := aguitypes.RunAgentInput{
		ThreadID:       threadID,
		RunID:          uuid.NewString(),
		Messages:       make([]aguitypes.Message, 0, len(msgs)),
		ForwardedProps: forwardedProps,
	}

	for index, item := range msgs {
		if item.Role != llm.RoleUser && item.Role != llm.RoleAssistant {
			continue
		}
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		input.Messages = append(input.Messages, aguitypes.Message{
			ID:      fmt.Sprintf("%s-%d", threadID, index),
			Role:    aguitypes.Role(item.Role),
			Content: content,
		})
	}
	if len(input.Messages) == 0 || input.Messages[len(input.Messages)-1].Role != aguitypes.RoleUser {
		return aguitypes.RunAgentInput{}, fmt.Errorf("liki: agent run requires a latest user message")
	}
	return input, nil
}

func endpointFor(baseURL string) string {
	return strings.TrimSuffix(strings.TrimSuffix(baseURL, "/"), "/ag-ui") + agentProtocolPath
}

// agentRunState accumulates protocol facts for one server-to-server run. The
// streamed text is only forwarded to the browser; RUN_FINISHED.result.answer
// remains the authoritative value that liki-web persists.
type agentRunState struct {
	done     bool
	answer   string
	runError error
	lastTool string
}

func (s *agentRunState) observe(event aguievents.Event, onEvent func(agent.ChatEvent)) error {
	switch ev := event.(type) {
	case *aguievents.RunStartedEvent:
		emitChatEvent(onEvent, agent.ChatEvent{Type: agent.EventThinking})
	case *aguievents.TextMessageContentEvent:
		emitChatEvent(onEvent, agent.ChatEvent{Type: agent.EventTextDelta, Content: ev.Delta})
	case *aguievents.ToolCallStartEvent:
		// 只转发原始工具名（事实）；显示文案由前端 i18n 渲染。连续同名去重。
		tool := ev.ToolCallName
		if tool != s.lastTool {
			s.lastTool = tool
			emitChatEvent(onEvent, agent.ChatEvent{Type: agent.EventPhase, Content: tool})
		}
	case *aguievents.ToolCallEndEvent:
		// phase 已在 Start 时发出，End 不重复
	case *aguievents.RunFinishedEvent:
		if ev.Result != nil {
			raw, err := json.Marshal(ev.Result)
			if err != nil {
				return fmt.Errorf("liki: encode agent result: %w", err)
			}
			var result struct {
				Answer string `json:"answer"`
			}
			if err := json.Unmarshal(raw, &result); err != nil {
				return fmt.Errorf("liki: decode agent result: %w", err)
			}
			s.answer = result.Answer
		}
		s.done = true
	case *aguievents.RunErrorEvent:
		message := strings.TrimSpace(ev.Message)
		if message == "" {
			message = "agent run failed"
		}
		s.runError = fmt.Errorf("%s", message)
		s.done = true
	default:
		// AG-UI is intentionally forward-compatible. liki-web consumes only
		// the product chat profile and ignores protocol extensions it does
		// not understand.
	}
	return nil
}

func (s *agentRunState) finalError() error {
	return s.runError
}

func emitChatEvent(fn func(agent.ChatEvent), event agent.ChatEvent) {
	if fn != nil {
		fn(event)
	}
}

// compile-time guard for the existing Runner contract.
var _ Runner = (*AgentRunner)(nil)
