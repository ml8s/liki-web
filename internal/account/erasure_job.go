package account

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/partner"
)

// ErasureStore persists erasure requests in platform.db. HandleErasure runs
// the domains inline (best effort) and records completed steps; on any
// failure the request stays queued and ErasureWorker resumes the remaining
// steps, so the erasure always completes even across restarts.
type ErasureStore struct {
	db *sql.DB
}

func NewErasureStore(db *sql.DB) *ErasureStore {
	return &ErasureStore{db: db}
}

func (s *ErasureStore) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS erasure_requests (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    email TEXT NOT NULL,
    tombstone TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued','running','done','failed')),
    attempts INTEGER NOT NULL DEFAULT 0,
    completed_steps TEXT NOT NULL DEFAULT '[]',
    last_error TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_erasure_requests_status
    ON erasure_requests(status, updated_at);
`)
	if err != nil {
		return fmt.Errorf("account: migrate erasure_requests: %w", err)
	}
	return nil
}

type ErasureRequest struct {
	ID             string
	UserID         string
	Email          string
	Tombstone      string
	Status         string
	Attempts       int
	CompletedSteps []string
	LastError      sql.NullString
}

const maxErasureAttempts = 5

func (s *ErasureStore) Enqueue(ctx context.Context, userID, email, tombstone string) (string, error) {
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO erasure_requests (id,user_id,email,tombstone,status,attempts,completed_steps,created_at,updated_at)
VALUES (?,?,?,?,'queued',0,'[]',?,?)`, id, userID, email, tombstone, now, now)
	if err != nil {
		return "", fmt.Errorf("account: enqueue erasure: %w", err)
	}
	return id, nil
}

// Claim moves one runnable request (queued, or failed with attempts left)
// to running so the worker never double-runs it.
func (s *ErasureStore) Claim(ctx context.Context) (ErasureRequest, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ErasureRequest{}, false, err
	}
	defer func() { _ = tx.Rollback() }()

	var req ErasureRequest
	var completed, lastError sql.NullString
	err = tx.QueryRowContext(ctx, `
SELECT id,user_id,email,tombstone,status,attempts,completed_steps,last_error
FROM erasure_requests
WHERE status='queued' OR (status='failed' AND attempts<?)
ORDER BY updated_at ASC LIMIT 1`, maxErasureAttempts).
		Scan(&req.ID, &req.UserID, &req.Email, &req.Tombstone, &req.Status,
			&req.Attempts, &completed, &lastError)
	if errors.Is(err, sql.ErrNoRows) {
		return ErasureRequest{}, false, nil
	}
	if err != nil {
		return ErasureRequest{}, false, err
	}
	req.CompletedSteps = parseSteps(completed)
	req.LastError = lastError
	if _, err := tx.ExecContext(ctx, `
UPDATE erasure_requests SET status='running', attempts=attempts+1, updated_at=?
WHERE id=?`, time.Now().UTC().Format(time.RFC3339), req.ID); err != nil {
		return ErasureRequest{}, false, err
	}
	return req, true, tx.Commit()
}

func (s *ErasureStore) MarkStep(ctx context.Context, id, step string) error {
	var completed sql.NullString
	if err := s.db.QueryRowContext(ctx,
		`SELECT completed_steps FROM erasure_requests WHERE id=?`, id).Scan(&completed); err != nil {
		return err
	}
	steps := parseSteps(completed)
	if slices.Contains(steps, step) {
		return nil
	}
	steps = append(steps, step)
	encoded, err := json.Marshal(steps)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE erasure_requests SET completed_steps=?, updated_at=? WHERE id=?`,
		string(encoded), time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *ErasureStore) MarkDone(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE erasure_requests SET status='done', updated_at=? WHERE id=?`,
		time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *ErasureStore) MarkFailed(ctx context.Context, id string, errStr string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE erasure_requests SET status='failed', last_error=?, updated_at=?
WHERE id=?`, errStr, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func parseSteps(value sql.NullString) []string {
	if !value.Valid || value.String == "" {
		return []string{}
	}
	var steps []string
	if err := json.Unmarshal([]byte(value.String), &steps); err != nil {
		return []string{}
	}
	return steps
}

// ErasureWorker drains queued/failed erasure requests. The step
// implementation is Handler.runErasureSteps — inline pass and worker share
// one implementation and cannot drift.
type ErasureWorker struct {
	Queue    *ErasureStore
	Handler  *Handler
	Interval time.Duration
}

// Start blocks until ctx is cancelled; run it in a goroutine.
func (w *ErasureWorker) Start(ctx context.Context) {
	interval := w.Interval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := w.RunPending(ctx); err != nil {
				slog.Error("account: erasure worker", "err", err)
			}
		}
	}
}

// RunPending processes every runnable erasure request once.
func (w *ErasureWorker) RunPending(ctx context.Context) (int, error) {
	processed := 0
	for {
		if ctx.Err() != nil {
			return processed, ctx.Err()
		}
		req, found, err := w.Queue.Claim(ctx)
		if err != nil {
			return processed, err
		}
		if !found {
			return processed, nil
		}
		user := auth.User{ID: req.UserID, Email: req.Email, Status: auth.UserActive}
		if _, err := w.Handler.runErasureSteps(ctx, user, req.Tombstone, req.ID); err != nil {
			errStr := runErrString(err)
			if markErr := w.Queue.MarkFailed(ctx, req.ID, errStr); markErr != nil {
				return processed, markErr
			}
			processed++
			continue
		}
		if err := w.Queue.MarkDone(ctx, req.ID); err != nil {
			return processed, err
		}
		processed++
	}
}

func runErrString(err error) string {
	if errors.Is(err, partner.ErrReferralFinancialHold) {
		return "partner_financial_hold"
	}
	return err.Error()
}

type erasureOutcome struct {
	ReferralResult  partner.ErasureResult
	ProfilesCleared int64
	Erased          map[string]int64
	MessagesCleared int64
	AuthResult      auth.ErasureResult
}
