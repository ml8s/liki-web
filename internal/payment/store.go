package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrIntentNotFound = errors.New("payment: intent not found")

type IntentStore struct {
	db *sql.DB
}

func NewIntentStore(db *sql.DB) *IntentStore { return &IntentStore{db: db} }

func (s *IntentStore) Migrate(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS payment_intents (
    id TEXT PRIMARY KEY,
    business_order_id TEXT NOT NULL,
    user_id TEXT NOT NULL DEFAULT '',
    product TEXT NOT NULL,
    provider TEXT NOT NULL,
    status TEXT NOT NULL,
    amount_minor INTEGER NOT NULL,
    currency TEXT NOT NULL,
    email TEXT NOT NULL,
    locale TEXT NOT NULL DEFAULT 'zh-Hant',
    metadata TEXT NOT NULL DEFAULT '{}',
    provider_session_id TEXT,
    provider_payment_id TEXT,
    checkout_url TEXT,
    qr_code_url TEXT,
    return_url TEXT,
    success_url TEXT,
    failure_url TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    paid_at TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_intents_provider_payment
    ON payment_intents(provider, provider_payment_id)
    WHERE provider_payment_id IS NOT NULL AND provider_payment_id != '';
CREATE INDEX IF NOT EXISTS idx_payment_intents_business_order
    ON payment_intents(business_order_id, product);

CREATE TABLE IF NOT EXISTS payment_refunds (
    id TEXT PRIMARY KEY,
    payment_intent_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_refund_id TEXT NOT NULL,
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    currency TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    notified_at TEXT,
    UNIQUE (provider, provider_refund_id)
);
CREATE INDEX IF NOT EXISTS idx_payment_refunds_intent
    ON payment_refunds(payment_intent_id, created_at);
`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return err
	}
	if err := s.ensureColumn(ctx, "payment_intents", "user_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	// SQLite drivers differ in how PRAGMA rows surface newly-added columns;
	// tolerate only the expected duplicate-column outcome.
	if _, err := s.db.ExecContext(ctx, `ALTER TABLE payment_intents ADD COLUMN refunded_at TEXT`); err != nil &&
		!strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `ALTER TABLE payment_intents ADD COLUMN refunded_amount_minor INTEGER NOT NULL DEFAULT 0`); err != nil &&
		!strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `
INSERT INTO payment_refunds
    (id,payment_intent_id,provider,provider_refund_id,amount_minor,currency,created_at,updated_at)
SELECT provider||':'||provider_payment_id,id,provider,provider_payment_id,amount_minor,currency,
       COALESCE(refunded_at,updated_at),updated_at
FROM payment_intents
WHERE status=? AND provider_payment_id IS NOT NULL AND provider_payment_id != ''
ON CONFLICT(provider,provider_refund_id) DO NOTHING`,
		string(PaymentStatusRefunded)); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `
UPDATE payment_intents SET refunded_amount_minor=amount_minor
WHERE status=? AND refunded_amount_minor=0`, string(PaymentStatusRefunded)); err != nil {
		return err
	}
	return nil
}

type RefundRecord struct {
	ID       string
	New      bool
	Notified bool
	Amount   int64
}

// RecordRefund stores one provider refund fact and advances the intent to
// refunded only when the cumulative refunded amount reaches the sale amount.
// Notified stays false until the downstream observer confirms durable intake.
func (s *IntentStore) RecordRefund(ctx context.Context, intent PaymentIntent, providerRefundID string, amountMinor int64, currency string) (PaymentIntent, RefundRecord, error) {
	if intent.ID == "" {
		return PaymentIntent{}, RefundRecord{}, ErrIntentNotFound
	}
	if providerRefundID == "" {
		providerRefundID = "refund-" + intent.ID
	}
	// amountMinor <= 0 means "refund the remaining balance"; the exact value
	// is re-derived inside the transaction below so concurrent refunds can
	// never overshoot.
	if strings.TrimSpace(currency) == "" {
		currency = intent.Currency
	}
	if intent.Currency != "" && !strings.EqualFold(currency, intent.Currency) {
		return PaymentIntent{}, RefundRecord{}, fmt.Errorf("payment: refund currency mismatch for intent %s: got %s want %s",
			intent.ID, currency, intent.Currency)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PaymentIntent{}, RefundRecord{}, err
	}
	defer func() { _ = tx.Rollback() }()
	// Re-read the authoritative refunded total inside the transaction. The
	// intent snapshot held by the caller may be stale under concurrent
	// refunds; relying on it allowed cumulative overdrafts.
	var refundedNow int64
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(refunded_amount_minor,0) FROM payment_intents WHERE id=?`,
		intent.ID).Scan(&refundedNow); err != nil {
		return PaymentIntent{}, RefundRecord{}, err
	}
	if amountMinor <= 0 {
		amountMinor = intent.AmountMinor - refundedNow
	}
	if amountMinor <= 0 || refundedNow+amountMinor > intent.AmountMinor {
		return PaymentIntent{}, RefundRecord{}, fmt.Errorf("payment: refund %d exceeds remaining %d for intent %s",
			amountMinor, intent.AmountMinor-refundedNow, intent.ID)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	refundID := string(intent.Provider) + ":" + providerRefundID
	res, err := tx.ExecContext(ctx, `
INSERT INTO payment_refunds
    (id,payment_intent_id,provider,provider_refund_id,amount_minor,currency,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?)
ON CONFLICT(provider,provider_refund_id) DO NOTHING`,
		refundID, intent.ID, string(intent.Provider), providerRefundID, amountMinor,
		strings.ToUpper(currency), now, now)
	if err != nil {
		return PaymentIntent{}, RefundRecord{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return PaymentIntent{}, RefundRecord{}, err
	}
	if n == 0 {
		var notified sql.NullString
		if err := tx.QueryRowContext(ctx, `SELECT notified_at FROM payment_refunds WHERE id=?`, refundID).Scan(&notified); err != nil {
			return PaymentIntent{}, RefundRecord{}, err
		}
		intent, err = s.GetIntent(ctx, intent.ID)
		return intent, RefundRecord{ID: refundID, Notified: notified.Valid}, err
	}
	res, err = tx.ExecContext(ctx, `
UPDATE payment_intents
SET refunded_amount_minor=refunded_amount_minor+?,
    status=CASE WHEN refunded_amount_minor+?>=amount_minor THEN ? ELSE ? END,
    refunded_at=COALESCE(refunded_at,?),
    updated_at=?
WHERE id=? AND refunded_amount_minor+?<=amount_minor`,
		amountMinor, amountMinor, string(PaymentStatusRefunded), string(IntentPaid), now, now, intent.ID, amountMinor)
	if err != nil {
		return PaymentIntent{}, RefundRecord{}, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return PaymentIntent{}, RefundRecord{}, err
	} else if n == 0 {
		// A concurrent refund consumed the remaining balance first.
		return PaymentIntent{}, RefundRecord{}, fmt.Errorf("payment: refund %d exceeds remaining balance for intent %s",
			amountMinor, intent.ID)
	}
	if err := tx.Commit(); err != nil {
		return PaymentIntent{}, RefundRecord{}, err
	}
	intent, err = s.GetIntent(ctx, intent.ID)
	return intent, RefundRecord{ID: refundID, New: true}, err
}

// MarkRefundNotified records successful hand-off. A crash before this point
// makes the next webhook retry re-deliver the fact to the observer.
func (s *IntentStore) MarkRefundNotified(ctx context.Context, refundID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
UPDATE payment_refunds SET notified_at=?,updated_at=? WHERE id=?`, now, now, refundID)
	return err
}

func (s *IntentStore) ensureColumn(ctx context.Context, table, column, definition string) error {
	// 白名单校验表名，防止 SQL 注入（虽然当前调用方传硬编码值，但保持防御性）。
	switch table {
	case "payment_intents", "payment_completion_tasks":
	default:
		return fmt.Errorf("unexpected table name: %q", table)
	}
	switch column {
	case "user_id", "webhook_event", "refunded_amount_minor":
	default:
		return fmt.Errorf("unexpected column name: %q", column)
	}
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}

func (s *IntentStore) CreateIntent(ctx context.Context, intent PaymentIntent) error {
	metadata, err := json.Marshal(intent.Metadata)
	if err != nil {
		return fmt.Errorf("payment: marshal metadata: %w", err)
	}
	now := time.Now().UTC()
	if intent.CreatedAt.IsZero() {
		intent.CreatedAt = now
	}
	if intent.UpdatedAt.IsZero() {
		intent.UpdatedAt = now
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO payment_intents (
    id, business_order_id, user_id, product, provider, status, amount_minor, currency, email,
    locale, metadata, return_url, success_url, failure_url, created_at, updated_at
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		intent.ID, intent.BusinessOrderID, intent.UserID, string(intent.Product), string(intent.Provider), string(intent.Status),
		intent.AmountMinor, intent.Currency, intent.Email, intent.Locale, string(metadata),
		intent.ReturnURL, intent.SuccessURL, intent.FailureURL,
		intent.CreatedAt.Format(time.RFC3339), intent.UpdatedAt.Format(time.RFC3339))
	return err
}

func (s *IntentStore) GetIntent(ctx context.Context, id string) (PaymentIntent, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, business_order_id, user_id, product, provider, status, amount_minor, currency, email,
       locale, metadata, provider_session_id, provider_payment_id, checkout_url, qr_code_url,
       return_url, success_url, failure_url, created_at, updated_at, paid_at, refunded_at, refunded_amount_minor
FROM payment_intents WHERE id=?`, id)
	return scanIntent(row.Scan)
}

// ListIntentsForReconciliation returns the open intents a provider can still
// answer for. Intents without a provider handle (a checkout that never opened)
// have nothing to fetch: asking anyway produced one ERROR per intent per cycle
// until the expiry job cleaned them up, and enough of them would starve the
// intents that do have a provider handle.
func (s *IntentStore) ListIntentsForReconciliation(ctx context.Context, limit int) ([]PaymentIntent, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, business_order_id, user_id, product, provider, status, amount_minor, currency, email,
       locale, metadata, provider_session_id, provider_payment_id, checkout_url, qr_code_url,
       return_url, success_url, failure_url, created_at, updated_at, paid_at, refunded_at, refunded_amount_minor
FROM payment_intents
WHERE status IN (?,?)
  AND (provider_session_id <> '' OR provider_payment_id <> '')
ORDER BY updated_at
LIMIT ?`, string(IntentCreated), string(IntentCheckoutCreated), limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var intents []PaymentIntent
	for rows.Next() {
		intent, scanErr := scanIntent(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		intents = append(intents, intent)
	}
	return intents, rows.Err()
}

// MarkStaleIntentsExpired closes unpaid checkout intents so reconciliation and
// retention jobs stop considering abandoned checkouts active.
func (s *IntentStore) MarkStaleIntentsExpired(ctx context.Context, older time.Duration) (int64, error) {
	before := time.Now().UTC().Add(-older).Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
UPDATE payment_intents
SET status=?, updated_at=?
WHERE status IN (?,?) AND updated_at < ?`,
		string(IntentExpired), time.Now().UTC().Format(time.RFC3339),
		string(IntentCreated), string(IntentCheckoutCreated), before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *IntentStore) MarkCheckoutCreated(ctx context.Context, id string, result CheckoutResult) (PaymentIntent, error) {
	res, err := s.db.ExecContext(ctx, `
UPDATE payment_intents
SET status=?, provider_session_id=?, checkout_url=?, qr_code_url=?, updated_at=?
WHERE id=? AND status IN (?,?)`,
		string(IntentCheckoutCreated), result.CheckoutSessionID, result.CheckoutURL, result.QRCodeURL,
		time.Now().UTC().Format(time.RFC3339), id, string(IntentCreated), string(IntentCheckoutCreated))
	if err != nil {
		return PaymentIntent{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return PaymentIntent{}, err
	}
	if n == 0 {
		return PaymentIntent{}, ErrIntentNotFound
	}
	return s.GetIntent(ctx, id)
}

func (s *IntentStore) MarkPaid(ctx context.Context, id, providerPaymentID string) (PaymentIntent, bool, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
UPDATE payment_intents
SET status=?, provider_payment_id=?, paid_at=?, updated_at=?
WHERE id=? AND status IN (?,?)`,
		string(IntentPaid), providerPaymentID, now, now, id,
		string(IntentCreated), string(IntentCheckoutCreated))
	if err != nil {
		return PaymentIntent{}, false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return PaymentIntent{}, false, err
	}
	if n == 0 {
		intent, getErr := s.GetIntent(ctx, id)
		if getErr != nil {
			return PaymentIntent{}, false, getErr
		}
		return intent, false, nil
	}
	intent, err := s.GetIntent(ctx, id)
	return intent, true, err
}

func scanIntent(scan func(dest ...any) error) (PaymentIntent, error) {
	var intent PaymentIntent
	var productKey, provider, status, metadata string
	var userID string
	var providerCheckoutSessionID, providerPaymentID, checkoutURL, qrCodeURL, returnURL, successURL, failureURL, paidAt, refundedAt sql.NullString
	var createdAt, updatedAt string
	err := scan(
		&intent.ID, &intent.BusinessOrderID, &userID, &productKey, &provider, &status,
		&intent.AmountMinor, &intent.Currency, &intent.Email, &intent.Locale, &metadata,
		&providerCheckoutSessionID, &providerPaymentID, &checkoutURL, &qrCodeURL,
		&returnURL, &successURL, &failureURL, &createdAt, &updatedAt, &paidAt, &refundedAt, &intent.RefundedAmountMinor,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return PaymentIntent{}, ErrIntentNotFound
	}
	if err != nil {
		return PaymentIntent{}, err
	}
	intent.Product = productKey
	intent.UserID = userID
	intent.Provider = Provider(provider)
	intent.Status = PaymentStatus(status)
	intent.ProviderCheckoutSessionID = providerCheckoutSessionID.String
	intent.ProviderPaymentID = providerPaymentID.String
	intent.CheckoutURL = checkoutURL.String
	intent.QRCodeURL = qrCodeURL.String
	intent.ReturnURL = returnURL.String
	intent.SuccessURL = successURL.String
	intent.FailureURL = failureURL.String
	_ = json.Unmarshal([]byte(metadata), &intent.Metadata)
	if intent.Metadata == nil {
		intent.Metadata = map[string]string{}
	}
	if parsed, parseErr := time.Parse(time.RFC3339, createdAt); parseErr == nil {
		intent.CreatedAt = parsed
	}
	if parsed, parseErr := time.Parse(time.RFC3339, updatedAt); parseErr == nil {
		intent.UpdatedAt = parsed
	}
	if paidAt.Valid {
		if parsed, parseErr := time.Parse(time.RFC3339, paidAt.String); parseErr == nil {
			intent.PaidAt = &parsed
		}
	}
	if refundedAt.Valid {
		if parsed, parseErr := time.Parse(time.RFC3339, refundedAt.String); parseErr == nil {
			intent.RefundedAt = &parsed
		}
	}
	return intent, nil
}
