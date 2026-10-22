package runcontrol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"

	"github.com/ml8s/liki-web/internal/chat"
	"github.com/ml8s/liki-web/internal/httputil"
)

const (
	completionEventType = "hk.liki.run.completed"
	callbackBodyLimit   = 1 << 20
)

// validRunStatuses reuses the A2A TaskState terminal vocabulary.
var validRunStatuses = map[string]bool{"completed": true, "failed": true, "canceled": true}

// SessionHistory persists the run exchange into the chat session. On the
// direct path the callback is the single authoritative writer of chat history.
// Implemented by *chat.ChatStore. The exchange is written as one batch so a
// failed write never leaves a partial turn for the retry to duplicate.
type SessionHistory interface {
	GetSession(ctx context.Context, userID, sessionID string) (chat.Session, error)
	AppendExchange(ctx context.Context, userID, sessionID string, messages ...chat.PendingMessage) ([]chat.Message, error)
}

// CallbackConfig wires the run-completed receiver (contract C2).
type CallbackConfig struct {
	Secret  string
	Store   *Store
	History SessionHistory
	Logger  *slog.Logger
}

// NewCallbackHandler returns the receiver for runtime run-completed callbacks.
// The body is verified with Standard Webhooks before anything is parsed or
// recorded; delivery is idempotent by run_id.
func NewCallbackHandler(cfg CallbackConfig) (http.HandlerFunc, error) {
	if cfg.Store == nil {
		return nil, errors.New("run callback store is required")
	}
	if cfg.History == nil {
		return nil, errors.New("run callback history writer is required")
	}
	webhook, err := standardwebhooks.NewWebhook(cfg.Secret)
	if err != nil {
		return nil, fmt.Errorf("run callback secret: %w", err)
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, callbackBodyLimit))
		if err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_body", "unreadable callback body")
			return
		}
		if err := webhook.Verify(body, r.Header); err != nil {
			httputil.RespondError(w, http.StatusUnauthorized, "invalid_signature", "signature verification failed")
			return
		}
		event, err := parseCompletionEvent(body)
		if err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_callback", err.Error())
			return
		}
		if key := r.Header.Get("Idempotency-Key"); key != event.Data.RunID {
			httputil.RespondError(w, http.StatusBadRequest, "idempotency_mismatch", "Idempotency-Key must equal run_id")
			return
		}
		inserted, err := cfg.Store.InsertRun(r.Context(), event.Data.record())
		if err != nil {
			logger.Error("record run callback", "run_id", event.Data.RunID, "error", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to record run")
			return
		}
		if inserted {
			if err := cfg.persistExchange(r.Context(), event.Data, logger); err != nil {
				// 补偿：删除 run 行，让重试完整重放（历史与审计保持一致）。
				if deleteErr := cfg.Store.DeleteRun(r.Context(), event.Data.RunID); deleteErr != nil {
					logger.Error("run callback compensation failed",
						"run_id", event.Data.RunID, "error", deleteErr)
				}
				logger.Error("persist run exchange", "run_id", event.Data.RunID, "error", err)
				httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to persist run exchange")
				return
			}
		}
		status := "duplicate"
		if inserted {
			status = "recorded"
		}
		logger.Info("run completed",
			"run_id", event.Data.RunID,
			"user_id", event.Data.UserID,
			"profile", event.Data.Profile,
			"status", event.Data.Status,
			"duplicate", !inserted,
		)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": status})
	}, nil
}

// persistExchange writes the run exchange (user message, plus the answer when
// completed) into the chat session. A missing session keeps the audit row and
// is not retried; a transient failure bubbles up so the delivery is replayed.
func (cfg CallbackConfig) persistExchange(ctx context.Context, data completionData, logger *slog.Logger) error {
	if _, err := cfg.History.GetSession(ctx, data.UserID, data.ThreadID); err != nil {
		if errors.Is(err, chat.ErrSessionNotFound) {
			logger.Warn("run callback session unmatched",
				"run_id", data.RunID, "thread_id", data.ThreadID)
			return nil
		}
		return fmt.Errorf("load session: %w", err)
	}
	exchange := []chat.PendingMessage{{Role: "user", Kind: chat.KindText, Content: data.UserMessage}}
	if data.Status == "completed" && data.Answer != "" {
		exchange = append(exchange, chat.PendingMessage{Role: "assistant", Kind: chat.KindText, Content: data.Answer})
	}
	if _, err := cfg.History.AppendExchange(ctx, data.UserID, data.ThreadID, exchange...); err != nil {
		return fmt.Errorf("append run exchange: %w", err)
	}
	return nil
}

type completionEvent struct {
	SpecVersion string         `json:"specversion"`
	ID          string         `json:"id"`
	Source      string         `json:"source"`
	Type        string         `json:"type"`
	Time        string         `json:"time"`
	Data        completionData `json:"data"`
}

type completionData struct {
	RunID       string           `json:"run_id"`
	UserID      string           `json:"user_id"`
	ThreadID    string           `json:"thread_id"`
	Profile     string           `json:"profile"`
	Status      string           `json:"status"`
	UserMessage string           `json:"user_message"`
	Answer      string           `json:"answer"`
	Usage       *completionUsage `json:"usage"`
	StartedAt   string           `json:"started_at"`
	FinishedAt  string           `json:"finished_at"`
	Error       string           `json:"error"`

	startedAt  time.Time
	finishedAt time.Time
}

type completionUsage struct {
	LLMCalls         int `json:"llm_calls"`
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	ThoughtTokens    int `json:"thought_tokens"`
}

func (d completionData) record() RunRecord {
	record := RunRecord{
		RunID:      d.RunID,
		UserID:     d.UserID,
		ThreadID:   d.ThreadID,
		Profile:    d.Profile,
		Status:     d.Status,
		Answer:     d.Answer,
		Error:      d.Error,
		StartedAt:  d.startedAt,
		FinishedAt: d.finishedAt,
	}
	if d.Usage != nil {
		record.LLMCalls = d.Usage.LLMCalls
		record.PromptTokens = d.Usage.PromptTokens
		record.CompletionTokens = d.Usage.CompletionTokens
		record.ThoughtTokens = d.Usage.ThoughtTokens
	}
	return record
}

// parseCompletionEvent validates the CloudEvents envelope and the data against
// contract C2 before anything is recorded.
func parseCompletionEvent(body []byte) (completionEvent, error) {
	var event completionEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return completionEvent{}, errors.New("callback body is not valid JSON")
	}
	if event.SpecVersion != "1.0" {
		return completionEvent{}, errors.New("specversion must be 1.0")
	}
	if event.Type != completionEventType {
		return completionEvent{}, fmt.Errorf("type must be %s", completionEventType)
	}
	if strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.Source) == "" {
		return completionEvent{}, errors.New("id and source are required")
	}
	if _, err := time.Parse(time.RFC3339, event.Time); err != nil {
		return completionEvent{}, errors.New("time must be RFC 3339")
	}
	data := &event.Data
	if data.RunID == "" || data.UserID == "" || data.ThreadID == "" || data.UserMessage == "" {
		return completionEvent{}, errors.New("run_id, user_id, thread_id and user_message are required")
	}
	if !chat.ValidExpert(data.Profile) {
		return completionEvent{}, errors.New("profile is not a deployed expert")
	}
	if !validRunStatuses[data.Status] {
		return completionEvent{}, errors.New("status must be completed, failed or canceled")
	}
	startedAt, err := time.Parse(time.RFC3339, data.StartedAt)
	if err != nil {
		return completionEvent{}, errors.New("started_at must be RFC 3339")
	}
	finishedAt, err := time.Parse(time.RFC3339, data.FinishedAt)
	if err != nil {
		return completionEvent{}, errors.New("finished_at must be RFC 3339")
	}
	if finishedAt.Before(startedAt) {
		return completionEvent{}, errors.New("finished_at must not precede started_at")
	}
	if data.Usage != nil {
		if data.Usage.LLMCalls < 0 || data.Usage.PromptTokens < 0 ||
			data.Usage.CompletionTokens < 0 || data.Usage.ThoughtTokens < 0 {
			return completionEvent{}, errors.New("usage counters must not be negative")
		}
	}
	data.startedAt = startedAt
	data.finishedAt = finishedAt
	return event, nil
}
