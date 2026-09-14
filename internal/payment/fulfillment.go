package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type FulfillmentStatus string

const (
	FulfillmentQueued    FulfillmentStatus = "queued"
	FulfillmentRunning   FulfillmentStatus = "running"
	FulfillmentSucceeded FulfillmentStatus = "succeeded"
	FulfillmentFailed    FulfillmentStatus = "failed"
)

const maxFulfillmentAttempts = 5

type FulfillmentTask struct {
	ID              string
	IntentID        string
	BusinessOrderID string
	Product         string
	Status          FulfillmentStatus
	Attempts        int
	RunAt           time.Time
	LastError       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

var ErrNoRunnableFulfillment = errors.New("payment: no runnable fulfillment task")

type FulfillmentTaskStore struct {
	db *sql.DB
}

func NewFulfillmentTaskStore(db *sql.DB) *FulfillmentTaskStore {
	return &FulfillmentTaskStore{db: db}
}

func (s *FulfillmentTaskStore) Migrate(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS fulfillment_tasks (
    id TEXT PRIMARY KEY,
    payment_intent_id TEXT NOT NULL UNIQUE,
    business_order_id TEXT NOT NULL,
    product TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued','running','succeeded','failed')),
    attempts INTEGER NOT NULL DEFAULT 0,
    run_at TEXT NOT NULL,
    last_error TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_fulfillment_tasks_due
    ON fulfillment_tasks(status, run_at);
`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// EnsureQueued creates the one fulfillment task for a paid intent. A terminal
// or running task is never reset by a duplicate webhook or reconciliation.
func (s *FulfillmentTaskStore) EnsureQueued(ctx context.Context, intent PaymentIntent) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO fulfillment_tasks
    (id, payment_intent_id, business_order_id, product, status, attempts, run_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, 0, ?, ?, ?)
ON CONFLICT(payment_intent_id) DO NOTHING`,
		uuid.NewString(), intent.ID, intent.BusinessOrderID, string(intent.Product),
		string(FulfillmentQueued), now, now, now)
	if err != nil {
		return fmt.Errorf("payment: enqueue fulfillment: %w", err)
	}
	return nil
}

// RecoverRunning returns tasks orphaned by a process restart to the queue.
func (s *FulfillmentTaskStore) RecoverRunning(ctx context.Context) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
UPDATE fulfillment_tasks
SET status=?, run_at=?, updated_at=?
WHERE status=?`, string(FulfillmentQueued), now, now, string(FulfillmentRunning))
	return err
}

func (s *FulfillmentTaskStore) ClaimNext(ctx context.Context, now time.Time) (FulfillmentTask, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FulfillmentTask{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var task FulfillmentTask
	var productKey, status, runAt, createdAt, updatedAt string
	var lastError sql.NullString
	err = tx.QueryRowContext(ctx, `
SELECT id, payment_intent_id, business_order_id, product, status, attempts, run_at, last_error, created_at, updated_at
FROM fulfillment_tasks
WHERE status=? AND run_at<=?
ORDER BY run_at
LIMIT 1`,
		string(FulfillmentQueued), now.Format(time.RFC3339),
	).Scan(&task.ID, &task.IntentID, &task.BusinessOrderID, &productKey, &status, &task.Attempts,
		&runAt, &lastError, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return FulfillmentTask{}, ErrNoRunnableFulfillment
	}
	if err != nil {
		return FulfillmentTask{}, err
	}

	claimedAt := now.Format(time.RFC3339)
	res, err := tx.ExecContext(ctx, `
UPDATE fulfillment_tasks
SET status=?, attempts=attempts+1, updated_at=?
WHERE id=? AND status=?`, string(FulfillmentRunning), claimedAt, task.ID, string(FulfillmentQueued))
	if err != nil {
		return FulfillmentTask{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return FulfillmentTask{}, ErrNoRunnableFulfillment
	}
	if err := tx.Commit(); err != nil {
		return FulfillmentTask{}, err
	}

	task.Product = string(productKey)
	task.Status = FulfillmentRunning
	task.Attempts++
	task.LastError = lastError.String
	task.RunAt = parseTime(runAt)
	task.CreatedAt = parseTime(createdAt)
	task.UpdatedAt = parseTime(claimedAt)
	return task, nil
}

func (s *FulfillmentTaskStore) GetByIntent(ctx context.Context, intentID string) (FulfillmentTask, bool, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, payment_intent_id, business_order_id, product, status, attempts, run_at, last_error, created_at, updated_at
FROM fulfillment_tasks WHERE payment_intent_id=?`, intentID)
	var task FulfillmentTask
	var productKey, status, runAt, createdAt, updatedAt string
	var lastError sql.NullString
	err := row.Scan(&task.ID, &task.IntentID, &task.BusinessOrderID, &productKey, &status, &task.Attempts,
		&runAt, &lastError, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return FulfillmentTask{}, false, nil
	}
	if err != nil {
		return FulfillmentTask{}, false, err
	}
	task.Product = string(productKey)
	task.Status = FulfillmentStatus(status)
	task.LastError = lastError.String
	task.RunAt = parseTime(runAt)
	task.CreatedAt = parseTime(createdAt)
	task.UpdatedAt = parseTime(updatedAt)
	return task, true, nil
}

func (s *FulfillmentTaskStore) MarkSucceeded(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
UPDATE fulfillment_tasks SET status=?, last_error=NULL, updated_at=? WHERE id=?`,
		string(FulfillmentSucceeded), now, id)
	return err
}

func (s *FulfillmentTaskStore) MarkFailed(ctx context.Context, id string, failure error) error {
	now := time.Now().UTC()
	status := FulfillmentQueued
	attempts, err := s.attempts(ctx, id)
	if err != nil {
		return err
	}
	if attempts >= maxFulfillmentAttempts {
		status = FulfillmentFailed
	}
	runAt := now
	if status != FulfillmentFailed {
		delays := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, 4 * time.Hour}
		runAt = now.Add(delays[min(attempts, len(delays)-1)])
	}
	_, err = s.db.ExecContext(ctx, `
UPDATE fulfillment_tasks
SET status=?, run_at=?, last_error=?, updated_at=?
WHERE id=?`,
		string(status), runAt.Format(time.RFC3339),
		failure.Error(), now.Format(time.RFC3339), id)
	return err
}

func (s *FulfillmentTaskStore) attempts(ctx context.Context, id string) (int, error) {
	var attempts int
	err := s.db.QueryRowContext(ctx,
		`SELECT attempts FROM fulfillment_tasks WHERE id=?`, id).Scan(&attempts)
	return attempts, err
}

func parseTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339, value)
	return parsed
}
