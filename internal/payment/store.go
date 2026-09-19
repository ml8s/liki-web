package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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
UPDATE payment_intents SET product='mingshu' WHERE product='lifebook';
`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return err
	}
	return s.ensureColumn(ctx, "payment_intents", "user_id", "TEXT NOT NULL DEFAULT ''")
}

func (s *IntentStore) ensureColumn(ctx context.Context, table, column, definition string) error {
	// 白名单校验表名，防止 SQL 注入（虽然当前调用方传硬编码值，但保持防御性）。
	switch table {
	case "payment_intents", "fulfillment_tasks":
	default:
		return fmt.Errorf("unexpected table name: %q", table)
	}
	switch column {
	case "user_id", "webhook_event":
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
       return_url, success_url, failure_url, created_at, updated_at, paid_at
FROM payment_intents WHERE id=?`, id)
	return scanIntent(row.Scan)
}

func (s *IntentStore) ListIntentsForReconciliation(ctx context.Context, limit int) ([]PaymentIntent, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, business_order_id, user_id, product, provider, status, amount_minor, currency, email,
       locale, metadata, provider_session_id, provider_payment_id, checkout_url, qr_code_url,
       return_url, success_url, failure_url, created_at, updated_at, paid_at
FROM payment_intents
WHERE status IN (?,?)
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
		string(IntentCheckoutCreated), result.SessionID, result.CheckoutURL, result.QRCodeURL,
		time.Now().UTC().Format(time.RFC3339), id, string(IntentCreated), string(IntentCheckoutCreated))
	if err != nil {
		return PaymentIntent{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
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
	var providerSessionID, providerPaymentID, checkoutURL, qrCodeURL, returnURL, successURL, failureURL, paidAt sql.NullString
	var createdAt, updatedAt string
	err := scan(
		&intent.ID, &intent.BusinessOrderID, &userID, &productKey, &provider, &status,
		&intent.AmountMinor, &intent.Currency, &intent.Email, &intent.Locale, &metadata,
		&providerSessionID, &providerPaymentID, &checkoutURL, &qrCodeURL,
		&returnURL, &successURL, &failureURL, &createdAt, &updatedAt, &paidAt,
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
	intent.ProviderSessionID = providerSessionID.String
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
	return intent, nil
}
