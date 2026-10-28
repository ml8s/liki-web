package chat

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestChatMigrationAllowsFreshDatabaseWithoutLegacyRunRecords(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := NewChatStore(db).Migrate(ctx); err != nil {
		t.Fatalf("migrate fresh chat store: %v", err)
	}

	var exists bool
	if err := db.QueryRowContext(ctx, `
SELECT EXISTS (SELECT 1 FROM sqlite_master WHERE type='table' AND name='run_records')
`).Scan(&exists); err != nil {
		t.Fatalf("check run_records table: %v", err)
	}
	if exists {
		t.Fatal("fresh database must not recreate the retired run_records writer table")
	}
}

func TestChatMigrationSanitizesLegacyRunRecordsWithoutDroppingUsageAudit(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.ExecContext(ctx, `
CREATE TABLE run_records (
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
INSERT INTO run_records VALUES (
  'run_1','user_1','session_1','natal','completed','sensitive answer','provider error',
  2,100,40,8,'2026-10-01T00:00:00Z','2026-10-01T00:01:00Z','2026-10-01T00:01:01Z'
);`); err != nil {
		t.Fatalf("seed legacy run records: %v", err)
	}

	if err := NewChatStore(db).Migrate(ctx); err != nil {
		t.Fatalf("migrate chat store: %v", err)
	}

	var exists bool
	if err := db.QueryRowContext(ctx, `
SELECT EXISTS (SELECT 1 FROM sqlite_master WHERE type='table' AND name='run_records')
`).Scan(&exists); err != nil {
		t.Fatalf("check run_records table: %v", err)
	}
	if !exists {
		t.Fatal("legacy run_records audit table was dropped")
	}

	var answer, errorMessage string
	var llmCalls, promptTokens, completionTokens, thoughtTokens int
	if err := db.QueryRowContext(ctx, `
SELECT answer,error,llm_calls,prompt_tokens,completion_tokens,thought_tokens
FROM run_records WHERE run_id='run_1'
`).Scan(&answer, &errorMessage, &llmCalls, &promptTokens, &completionTokens, &thoughtTokens); err != nil {
		t.Fatalf("read sanitized run record: %v", err)
	}
	if answer != "" || errorMessage != "" {
		t.Fatalf("PII not cleared: answer=%q error=%q", answer, errorMessage)
	}
	if llmCalls != 2 || promptTokens != 100 || completionTokens != 40 || thoughtTokens != 8 {
		t.Fatalf("usage audit changed: llm=%d prompt=%d completion=%d thought=%d",
			llmCalls, promptTokens, completionTokens, thoughtTokens)
	}
}
