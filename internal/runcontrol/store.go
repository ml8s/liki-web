package runcontrol

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// RunRecord is one completed run as reported by the runtime callback
// (contract C2). Status reuses the A2A TaskState vocabulary:
// completed | failed | canceled.
type RunRecord struct {
	RunID            string
	UserID           string
	ThreadID         string
	Profile          string
	Status           string
	Answer           string
	Error            string
	LLMCalls         int
	PromptTokens     int
	CompletionTokens int
	ThoughtTokens    int
	StartedAt        time.Time
	FinishedAt       time.Time
	ReceivedAt       time.Time
}

// Store owns the run registry table of the direct-connect control plane.
type Store struct {
	db  *sql.DB
	now func() time.Time
}

// NewStore creates the run registry store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db, now: time.Now}
}

// Migrate creates the run registry schema.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS run_records (
  run_id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  thread_id TEXT NOT NULL,
  profile TEXT NOT NULL,
  status TEXT NOT NULL,
  answer TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT '',
  llm_calls INTEGER NOT NULL DEFAULT 0,
  prompt_tokens INTEGER NOT NULL DEFAULT 0,
  completion_tokens INTEGER NOT NULL DEFAULT 0,
  thought_tokens INTEGER NOT NULL DEFAULT 0,
  started_at TEXT NOT NULL,
  finished_at TEXT NOT NULL,
  received_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_run_records_thread ON run_records(thread_id);
`); err != nil {
		return fmt.Errorf("migrate run records: %w", err)
	}
	return nil
}

// InsertRun records a completed run. It reports false when run_id was already
// recorded, which makes callback retries idempotent.
func (s *Store) InsertRun(ctx context.Context, record RunRecord) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
INSERT INTO run_records
  (run_id,user_id,thread_id,profile,status,answer,error,llm_calls,prompt_tokens,completion_tokens,thought_tokens,started_at,finished_at,received_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(run_id) DO NOTHING`,
		record.RunID, record.UserID, record.ThreadID, record.Profile, record.Status, record.Answer, record.Error,
		record.LLMCalls, record.PromptTokens, record.CompletionTokens, record.ThoughtTokens,
		runFormatTime(record.StartedAt), runFormatTime(record.FinishedAt), runFormatTime(s.now()),
	)
	if err != nil {
		return false, fmt.Errorf("insert run record: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("insert run record: %w", err)
	}
	return affected > 0, nil
}

// DeleteRun removes a run row. The callback receiver uses it to compensate a
// delivery whose history write failed so the retry can replay the exchange.
func (s *Store) DeleteRun(ctx context.Context, runID string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM run_records WHERE run_id=?`, runID); err != nil {
		return fmt.Errorf("delete run record: %w", err)
	}
	return nil
}

// AnonymizeUser clears generated answers and provider/runtime errors while
// retaining pseudonymous counters and timestamps for operational audit.
func (s *Store) AnonymizeUser(ctx context.Context, userID string) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
UPDATE run_records SET answer='', error='' WHERE user_id=?`, userID)
	if err != nil {
		return 0, fmt.Errorf("anonymize run records: %w", err)
	}
	return result.RowsAffected()
}

func runFormatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
