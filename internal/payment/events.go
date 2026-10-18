package payment

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PaymentEventSource string

const (
	PaymentEventWebhook        PaymentEventSource = "webhook"
	PaymentEventReconciliation PaymentEventSource = "reconciliation"
)

type PaymentEventStore struct {
	db *sql.DB
}

func NewPaymentEventStore(db *sql.DB) *PaymentEventStore { return &PaymentEventStore{db: db} }

func (s *PaymentEventStore) Migrate(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS payment_events (
    id TEXT PRIMARY KEY,
    payment_intent_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    event_type TEXT NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('webhook','reconciliation')),
    payload_hash TEXT NOT NULL,
    payment_id TEXT NOT NULL DEFAULT '',
    received_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_events_dedup
    ON payment_events(payment_intent_id, source, event_type, payload_hash);
CREATE INDEX IF NOT EXISTS idx_payment_events_intent_time
    ON payment_events(payment_intent_id, received_at);
`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// Record stores only a payload hash, never provider PII or the raw callback.
// It returns true when a new event was inserted and false for a duplicate.
func (s *PaymentEventStore) Record(ctx context.Context, intentID string, source PaymentEventSource, event WebhookEvent) (bool, error) {
	if intentID == "" {
		return false, fmt.Errorf("payment: event intent id is empty")
	}
	hash, err := HashPaymentEvent(event)
	if err != nil {
		return false, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
INSERT INTO payment_events
    (id, payment_intent_id, provider, event_type, source, payload_hash, payment_id, received_at)
VALUES (?,?,?,?,?,?,?,?)
ON CONFLICT(payment_intent_id, source, event_type, payload_hash) DO NOTHING`,
		uuid.NewString(), intentID, string(event.Provider), event.Type, string(source),
		hash, event.Data.PaymentID, now)
	if err != nil {
		return false, fmt.Errorf("payment: record event: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// HashPaymentEvent uses a stable JSON subset and intentionally excludes email
// and provider metadata from the audit row.
func HashPaymentEvent(event WebhookEvent) (string, error) {
	payload := struct {
		Provider  string `json:"provider"`
		Type      string `json:"type"`
		OrderID   string `json:"order_id"`
		Amount    int64  `json:"amount"`
		Currency  string `json:"currency"`
		PaymentID string `json:"payment_id"`
	}{
		Provider:  string(event.Provider),
		Type:      event.Type,
		OrderID:   event.Data.OrderID,
		Amount:    event.Data.Amount,
		Currency:  event.Data.Currency,
		PaymentID: event.Data.PaymentID,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
