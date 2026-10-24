package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PaymentCompletionStatus string

const (
	PaymentCompletionQueued    PaymentCompletionStatus = "queued"
	PaymentCompletionRunning   PaymentCompletionStatus = "running"
	PaymentCompletionSucceeded PaymentCompletionStatus = "succeeded"
	PaymentCompletionFailed    PaymentCompletionStatus = "failed"
)

const maxFulfillmentAttempts = 5

type PaymentCompletionTask struct {
	ID              string
	IntentID        string
	BusinessOrderID string
	Product         string
	Status          PaymentCompletionStatus
	Attempts        int
	RunAt           time.Time
	LastError       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

var ErrNoRunnablePaymentCompletion = errors.New("payment: no runnable fulfillment task")

type PaymentCompletionTaskStore struct {
	db *sql.DB
}

func NewPaymentCompletionTaskStore(db *sql.DB) *PaymentCompletionTaskStore {
	return &PaymentCompletionTaskStore{db: db}
}

func (s *PaymentCompletionTaskStore) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS payment_completion_tasks (
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
CREATE INDEX IF NOT EXISTS idx_payment_completion_tasks_due
    ON payment_completion_tasks(status, run_at);
`)
	return err
}

// EnsureQueued creates the one fulfillment task for a paid intent. A terminal
// or running task is never reset by a duplicate webhook or reconciliation.
func (s *PaymentCompletionTaskStore) EnsureQueued(ctx context.Context, intent PaymentIntent) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO payment_completion_tasks
    (id, payment_intent_id, business_order_id, product, status, attempts, run_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, 0, ?, ?, ?)
ON CONFLICT(payment_intent_id) DO NOTHING`,
		uuid.NewString(), intent.ID, intent.BusinessOrderID, string(intent.Product),
		string(PaymentCompletionQueued), now, now, now)
	if err != nil {
		return fmt.Errorf("payment: enqueue fulfillment: %w", err)
	}
	return nil
}

// RecoverRunning returns tasks orphaned by a process restart to the queue.
func (s *PaymentCompletionTaskStore) RecoverRunning(ctx context.Context) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
UPDATE payment_completion_tasks
SET status=?, run_at=?, updated_at=?
WHERE status=?`, string(PaymentCompletionQueued), now, now, string(PaymentCompletionRunning))
	return err
}

func (s *PaymentCompletionTaskStore) ClaimNext(ctx context.Context, now time.Time) (PaymentCompletionTask, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PaymentCompletionTask{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var task PaymentCompletionTask
	var productKey, status, runAt, createdAt, updatedAt string
	var lastError sql.NullString
	err = tx.QueryRowContext(ctx, `
SELECT id, payment_intent_id, business_order_id, product, status, attempts, run_at, last_error, created_at, updated_at
FROM payment_completion_tasks
WHERE status=? AND run_at<=?
ORDER BY run_at
LIMIT 1`,
		string(PaymentCompletionQueued), now.Format(time.RFC3339),
	).Scan(&task.ID, &task.IntentID, &task.BusinessOrderID, &productKey, &status, &task.Attempts,
		&runAt, &lastError, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return PaymentCompletionTask{}, ErrNoRunnablePaymentCompletion
	}
	if err != nil {
		return PaymentCompletionTask{}, err
	}

	claimedAt := now.Format(time.RFC3339)
	res, err := tx.ExecContext(ctx, `
UPDATE payment_completion_tasks
SET status=?, attempts=attempts+1, updated_at=?
WHERE id=? AND status=?`, string(PaymentCompletionRunning), claimedAt, task.ID, string(PaymentCompletionQueued))
	if err != nil {
		return PaymentCompletionTask{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return PaymentCompletionTask{}, ErrNoRunnablePaymentCompletion
	}
	if err := tx.Commit(); err != nil {
		return PaymentCompletionTask{}, err
	}

	task.Product = string(productKey)
	task.Status = PaymentCompletionRunning
	task.Attempts++
	task.LastError = lastError.String
	task.RunAt = parseTime(runAt)
	task.CreatedAt = parseTime(createdAt)
	task.UpdatedAt = parseTime(claimedAt)
	return task, nil
}

func (s *PaymentCompletionTaskStore) GetByIntent(ctx context.Context, intentID string) (PaymentCompletionTask, bool, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, payment_intent_id, business_order_id, product, status, attempts, run_at, last_error, created_at, updated_at
FROM payment_completion_tasks WHERE payment_intent_id=?`, intentID)
	var task PaymentCompletionTask
	var productKey, status, runAt, createdAt, updatedAt string
	var lastError sql.NullString
	err := row.Scan(&task.ID, &task.IntentID, &task.BusinessOrderID, &productKey, &status, &task.Attempts,
		&runAt, &lastError, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return PaymentCompletionTask{}, false, nil
	}
	if err != nil {
		return PaymentCompletionTask{}, false, err
	}
	task.Product = string(productKey)
	task.Status = PaymentCompletionStatus(status)
	task.LastError = lastError.String
	task.RunAt = parseTime(runAt)
	task.CreatedAt = parseTime(createdAt)
	task.UpdatedAt = parseTime(updatedAt)
	return task, true, nil
}

func (s *PaymentCompletionTaskStore) MarkSucceeded(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
UPDATE payment_completion_tasks SET status=?, last_error=NULL, updated_at=? WHERE id=?`,
		string(PaymentCompletionSucceeded), now, id)
	return err
}

func (s *PaymentCompletionTaskStore) MarkFailed(ctx context.Context, id string, failure error) error {
	now := time.Now().UTC()
	status := PaymentCompletionQueued
	attempts, err := s.attempts(ctx, id)
	if err != nil {
		return err
	}
	if attempts >= maxFulfillmentAttempts {
		status = PaymentCompletionFailed
	}
	runAt := now
	if status != PaymentCompletionFailed {
		delays := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, 4 * time.Hour}
		runAt = now.Add(delays[min(attempts, len(delays)-1)])
	}
	_, err = s.db.ExecContext(ctx, `
UPDATE payment_completion_tasks
SET status=?, run_at=?, last_error=?, updated_at=?
WHERE id=?`,
		string(status), runAt.Format(time.RFC3339),
		failure.Error(), now.Format(time.RFC3339), id)
	return err
}

func (s *PaymentCompletionTaskStore) attempts(ctx context.Context, id string) (int, error) {
	var attempts int
	err := s.db.QueryRowContext(ctx,
		`SELECT attempts FROM payment_completion_tasks WHERE id=?`, id).Scan(&attempts)
	return attempts, err
}

func parseTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339, value)
	return parsed
}
