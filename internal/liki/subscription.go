package liki

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ml8s/liki-web/internal/migrate"
	"github.com/ml8s/liki-web/internal/payment"
)

type SubscriptionStatus string

const (
	SubscriptionPending   SubscriptionStatus = "pending"
	SubscriptionActive    SubscriptionStatus = "active"
	SubscriptionOnHold    SubscriptionStatus = "on_hold"
	SubscriptionCancelled SubscriptionStatus = "cancelled"
	SubscriptionExpired   SubscriptionStatus = "expired"
	SubscriptionFailed    SubscriptionStatus = "failed"
)

var (
	ErrSubscriptionNotFound = errors.New("liki: subscription not found")
	ErrSubscriptionExists   = errors.New("liki: subscription payment already recorded")
)

type Subscription struct {
	ID                     string             `json:"id"`
	UserID                 string             `json:"user_id"`
	Email                  string             `json:"email"`
	ProductID              Product            `json:"product_id"`
	BillingMode            BillingMode        `json:"billing_mode"`
	Provider               payment.Provider   `json:"provider"`
	ProviderSubscriptionID string             `json:"provider_subscription_id"`
	ProviderCustomerID     string             `json:"provider_customer_id"`
	ProviderCheckoutID     string             `json:"provider_checkout_id"`
	Status                 SubscriptionStatus `json:"status"`
	CurrencyCode           CurrencyCode       `json:"currency_code"`
	CurrentPeriodStart     *time.Time         `json:"current_period_start"`
	CurrentPeriodEnd       *time.Time         `json:"current_period_end"`
	NextBillingAt          *time.Time         `json:"next_billing_at"`
	CancelAtPeriodEnd      bool               `json:"cancel_at_period_end"`
	CancelledAt            *time.Time         `json:"cancelled_at"`
	ProviderSyncedAt       *time.Time         `json:"provider_synced_at"`
	ProviderPeriodCount    int64              `json:"provider_period_count"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
}

type SubscriptionPayment struct {
	ID                 string           `json:"id"`
	SubscriptionID     string           `json:"subscription_id"`
	UserID             string           `json:"user_id"`
	ProductID          Product          `json:"product_id"`
	Provider           payment.Provider `json:"provider"`
	ProviderPaymentID  string           `json:"provider_payment_id"`
	AmountMinorUnits   int64            `json:"amount_minor_units"`
	RefundedMinorUnits int64            `json:"refunded_minor_units"`
	CurrencyCode       CurrencyCode     `json:"currency_code"`
	Status             string           `json:"status"`
	PeriodStart        *time.Time       `json:"period_start"`
	PeriodEnd          *time.Time       `json:"period_end"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
}

type SubscriptionStore struct {
	db      *sql.DB
	nowFunc func() time.Time
	idFunc  func() string
}

func NewSubscriptionStore(db *sql.DB) *SubscriptionStore {
	return &SubscriptionStore{
		db:      db,
		nowFunc: func() time.Time { return time.Now().UTC() },
		idFunc:  uuid.NewString,
	}
}

func (s *SubscriptionStore) Migrate(ctx context.Context) error {
	return migrate.Run(ctx, s.db, []migrate.Migration{{
		ID: "001_chat_subscriptions",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS chat_subscriptions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    email TEXT NOT NULL,
    product_id TEXT NOT NULL,
    billing_mode TEXT NOT NULL CHECK (billing_mode = 'subscription'),
    provider TEXT NOT NULL CHECK (provider = 'dodo'),
    provider_subscription_id TEXT UNIQUE,
    provider_customer_id TEXT,
    provider_checkout_id TEXT,
    status TEXT NOT NULL CHECK (status IN
        ('pending','active','on_hold','cancelled','expired','failed')),
    currency_code TEXT NOT NULL,
    current_period_start TEXT,
    current_period_end TEXT,
    next_billing_at TEXT,
    cancel_at_period_end INTEGER NOT NULL CHECK (cancel_at_period_end IN (0,1)),
    cancelled_at TEXT,
    provider_synced_at TEXT,
    provider_period_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_chat_subscriptions_user
    ON chat_subscriptions(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_subscriptions_status
    ON chat_subscriptions(status, current_period_end);

CREATE TABLE IF NOT EXISTS chat_subscription_payments (
    id TEXT PRIMARY KEY,
    subscription_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    product_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_payment_id TEXT NOT NULL,
    amount_minor_units INTEGER NOT NULL CHECK (amount_minor_units > 0),
    currency_code TEXT NOT NULL,
    period_start TEXT,
    period_end TEXT,
    status TEXT NOT NULL CHECK (status IN ('paid','refunded')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (provider, provider_payment_id)
);
CREATE INDEX IF NOT EXISTS idx_chat_subscription_payments_user
    ON chat_subscription_payments(user_id, created_at DESC);
`)
			return err
		},
	}, {
		ID: "002_chat_subscription_payment_reliability",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
ALTER TABLE chat_subscription_payments ADD COLUMN refunded_minor_units INTEGER NOT NULL DEFAULT 0;
ALTER TABLE chat_subscription_payments ADD COLUMN observer_enqueued_at TEXT;

CREATE TABLE IF NOT EXISTS chat_subscription_payment_refunds (
    id TEXT PRIMARY KEY,
    payment_id TEXT NOT NULL REFERENCES chat_subscription_payments(id),
    provider TEXT NOT NULL,
    provider_refund_id TEXT NOT NULL,
    amount_minor_units INTEGER NOT NULL CHECK (amount_minor_units > 0),
    currency_code TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (provider, provider_refund_id)
);
CREATE INDEX IF NOT EXISTS idx_chat_subscription_payment_refunds_payment
    ON chat_subscription_payment_refunds(payment_id);
`)
			return err
		},
	}})
}

func (s *SubscriptionStore) CreatePending(
	ctx context.Context, userID, email string, productID Product, currency CurrencyCode,
) (Subscription, error) {
	userID = strings.TrimSpace(userID)
	email = strings.ToLower(strings.TrimSpace(email))
	if userID == "" || email == "" {
		return Subscription{}, fmt.Errorf("liki: subscription requires user and email")
	}
	if _, err := ProductDefinitionFor(productID); err != nil {
		return Subscription{}, err
	}
	if currency != CurrencyUSD {
		return Subscription{}, fmt.Errorf("liki: subscription does not support currency %s", currency)
	}
	now := s.nowFunc()
	item := Subscription{
		ID: s.idFunc(), UserID: userID, Email: email, ProductID: productID,
		BillingMode: BillingModeSubscription, Provider: payment.ProviderDodo,
		Status: SubscriptionPending, CurrencyCode: currency,
		CreatedAt: now, UpdatedAt: now,
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO chat_subscriptions
(id,user_id,email,product_id,billing_mode,provider,status,currency_code,
 cancel_at_period_end,created_at,updated_at)
VALUES (?,?,?,?,?,?,?, ?,0,?,?)`,
		item.ID, item.UserID, item.Email, string(item.ProductID), string(item.BillingMode),
		string(item.Provider), string(item.Status), string(item.CurrencyCode),
		formatSubscriptionTime(now), formatSubscriptionTime(now))
	if err != nil {
		return Subscription{}, fmt.Errorf("liki: create subscription: %w", err)
	}
	return item, nil
}

func (s *SubscriptionStore) AttachCheckout(ctx context.Context, userID, subscriptionID, checkoutID string) error {
	res, err := s.db.ExecContext(ctx, `
UPDATE chat_subscriptions
SET provider_checkout_id=?,updated_at=?
WHERE id=? AND user_id=? AND status=?`,
		checkoutID, formatSubscriptionTime(s.nowFunc()), subscriptionID, userID, string(SubscriptionPending))
	if err != nil {
		return fmt.Errorf("liki: attach subscription checkout: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

func (s *SubscriptionStore) GetByUser(ctx context.Context, userID, subscriptionID string) (Subscription, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT `+subscriptionColumns()+`
FROM chat_subscriptions WHERE id=? AND user_id=?`, subscriptionID, userID)
	item, err := scanSubscription(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return Subscription{}, ErrSubscriptionNotFound
	}
	return item, err
}

// RequireNoBlockingSubscription prevents a second checkout while a paid or
// recoverable subscription already owns access. Abandoned pending rows do not
// block a retry.
func (s *SubscriptionStore) RequireNoBlockingSubscription(ctx context.Context, userID string) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `
SELECT COUNT(1)
FROM chat_subscriptions
WHERE user_id=? AND status IN (?,?)`,
		userID, string(SubscriptionActive), string(SubscriptionOnHold),
	).Scan(&count); err != nil {
		return fmt.Errorf("liki: check blocking subscription: %w", err)
	}
	if count > 0 {
		return ErrSubscriptionConflict
	}
	return nil
}

func (s *SubscriptionStore) ListByUser(ctx context.Context, userID string, limit int) ([]Subscription, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT `+subscriptionColumns()+`
FROM chat_subscriptions WHERE user_id=?
ORDER BY created_at DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]Subscription, 0)
	for rows.Next() {
		item, scanErr := scanSubscription(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SubscriptionStore) ListPendingWithCheckoutByUser(ctx context.Context, userID string, limit int) ([]Subscription, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT `+subscriptionColumns()+`
FROM chat_subscriptions
WHERE user_id=? AND status=? AND provider_checkout_id<>''
ORDER BY created_at DESC LIMIT ?`, userID, string(SubscriptionPending), limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]Subscription, 0)
	for rows.Next() {
		item, scanErr := scanSubscription(rows.Scan)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SubscriptionStore) ApplyWebhook(ctx context.Context, entitlements *EntitlementStore, event payment.WebhookEvent) error {
	if !payment.IsSubscriptionLifecycle(event.Type) {
		return fmt.Errorf("liki: %q is not a subscription lifecycle event", event.Type)
	}
	data := event.Data
	if data.SubscriptionID == "" {
		return fmt.Errorf("liki: subscription webhook has no provider subscription id")
	}
	localKey := data.Metadata["liki_subscription_key"]
	existing, err := s.findByProviderOrLocalKey(ctx, data.SubscriptionID, localKey)
	if err != nil {
		return err
	}

	userID := data.Metadata["liki_user_id"]
	if userID == "" {
		userID = existing.UserID
	}
	email := strings.ToLower(strings.TrimSpace(data.Email))
	if email == "" {
		email = existing.Email
	}
	productID := existing.ProductID
	syncedAt := s.nowFunc()
	if data.ProviderEventTime != nil && data.ProviderEventTime.Before(existing.providerSyncedAtOr(syncedAt)) &&
		data.ProviderPeriodCount < existing.ProviderPeriodCount {
		return nil
	}

	status := subscriptionStatusFor(event.Type, data.Status)
	cancelledAt := existing.CancelledAt
	if status == SubscriptionCancelled && cancelledAt == nil {
		cancelledAt = &syncedAt
	}
	if existing.ID == "" {
		if localKey == "" || userID == "" {
			return fmt.Errorf("liki: cannot attach webhook to unknown subscription %s", data.SubscriptionID)
		}
		existing.ID = localKey
		existing.UserID = userID
		existing.Email = email
		existing.ProductID = Product(orDefault(data.Metadata["liki_product"], string(ProductChat)))
		if _, err := ProductDefinitionFor(existing.ProductID); err != nil {
			return err
		}
		existing.BillingMode = BillingModeSubscription
		existing.Provider = event.Provider
		existing.Status = SubscriptionPending
		existing.CurrencyCode = CurrencyCode(orDefault(data.Currency, "USD"))
		existing.CreatedAt = syncedAt
	}

	if existing.ID != "" {
		_, err = s.db.ExecContext(ctx, `
UPDATE chat_subscriptions SET
    provider_subscription_id=?,provider_customer_id=?,status=?,email=?,
    current_period_start=?,current_period_end=?,next_billing_at=?,
    cancel_at_period_end=?,cancelled_at=?,provider_synced_at=?,
    provider_period_count=?,updated_at=?
WHERE id=?`,
			data.SubscriptionID, data.CustomerID, string(status), email,
			formatNullableSubscriptionTime(data.PeriodStart),
			formatNullableSubscriptionTime(data.PeriodEnd),
			formatNullableSubscriptionTime(data.NextBillingAt),
			boolInt(data.CancelAtPeriodEnd),
			formatNullableSubscriptionTime(cancelledAt),
			formatSubscriptionTime(syncedAt), data.ProviderPeriodCount,
			formatSubscriptionTime(syncedAt), existing.ID)
		if err != nil {
			return fmt.Errorf("liki: update subscription: %w", err)
		}
	}

	if status == SubscriptionActive && data.PeriodStart != nil && data.PeriodEnd != nil {
		_, err = entitlements.GrantSubscriptionPeriod(
			ctx, userID, email, productID, data.SubscriptionID,
			*data.PeriodStart, *data.PeriodEnd,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SubscriptionStore) RecordPayment(
	ctx context.Context, event payment.WebhookEvent,
	observer func(context.Context, payment.WebhookEvent) error,
) error {
	data := event.Data
	if data.SubscriptionID == "" || data.PaymentID == "" || data.Amount <= 0 {
		return fmt.Errorf("liki: subscription payment payload is incomplete")
	}
	subscription, err := s.findByProviderOrLocalKey(ctx, data.SubscriptionID, data.Metadata["liki_subscription_key"])
	if err != nil {
		return err
	}
	userID := data.Metadata["liki_user_id"]
	if userID == "" {
		userID = subscription.UserID
	}
	now := s.nowFunc()
	id := s.idFunc()
	result, err := s.db.ExecContext(ctx, `
INSERT INTO chat_subscription_payments
(id,subscription_id,user_id,product_id,provider,provider_payment_id,
 amount_minor_units,currency_code,period_start,period_end,status,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,'paid',?,?)
ON CONFLICT(provider,provider_payment_id) DO NOTHING`,
		id, subscription.ID, userID, string(subscription.ProductID), string(event.Provider),
		data.PaymentID, data.Amount, string(subscription.CurrencyCode),
		formatNullableSubscriptionTime(data.PeriodStart),
		formatNullableSubscriptionTime(data.PeriodEnd),
		formatSubscriptionTime(now), formatSubscriptionTime(now))
	if err != nil {
		return fmt.Errorf("liki: record subscription payment: %w", err)
	}
	inserted, _ := result.RowsAffected()
	notified, err := s.notifyPaymentObserver(ctx, event, inserted > 0, observer)
	if err != nil {
		return err
	}
	_ = notified
	return nil
}

func (s *SubscriptionStore) notifyPaymentObserver(
	ctx context.Context,
	event payment.WebhookEvent,
	justInserted bool,
	observer func(context.Context, payment.WebhookEvent) error,
) (bool, error) {
	if observer == nil {
		return false, nil
	}
	needsObserver := justInserted
	if !needsObserver {
		var notified sql.NullString
		if err := s.db.QueryRowContext(ctx, `
SELECT observer_enqueued_at
FROM chat_subscription_payments
WHERE provider=? AND provider_payment_id=?`,
			string(event.Provider), event.Data.PaymentID,
		).Scan(&notified); err != nil {
			return false, fmt.Errorf("liki: load subscription payment observer state: %w", err)
		}
		needsObserver = !notified.Valid
	}
	if !needsObserver {
		return false, nil
	}
	if err := observer(context.WithoutCancel(ctx), event); err != nil {
		return false, err
	}
	now := formatSubscriptionTime(s.nowFunc())
	if _, err := s.db.ExecContext(ctx, `
UPDATE chat_subscription_payments
SET observer_enqueued_at=?,updated_at=?
WHERE provider=? AND provider_payment_id=? AND observer_enqueued_at IS NULL`,
		now, now, string(event.Provider), event.Data.PaymentID); err != nil {
		return true, fmt.Errorf("liki: mark subscription payment observer: %w", err)
	}
	return true, nil
}

func (s *SubscriptionStore) RecordPaymentRefund(
	ctx context.Context, event payment.WebhookEvent,
	refundObserver func(context.Context, payment.WebhookEvent, payment.PaymentIntent) error,
) error {
	data := event.Data
	refundID := data.RefundID
	if refundID == "" {
		refundID = data.PaymentID
	}
	if event.Type != payment.EventPaymentRefunded || refundID == "" || data.RefundedPaymentID == "" || data.Amount <= 0 {
		return fmt.Errorf("liki: subscription refund payload is incomplete")
	}
	var paymentID, subscriptionID, userID, productID, providerPaymentID, currency string
	var amount, refunded int64
	var providerSubscriptionID sql.NullString
	if err := s.db.QueryRowContext(ctx, `
SELECT p.id,p.subscription_id,p.user_id,p.product_id,p.provider_payment_id,
       p.amount_minor_units,p.refunded_minor_units,p.currency_code,
       COALESCE(sub.provider_subscription_id,'')
FROM chat_subscription_payments p
       JOIN chat_subscriptions sub ON sub.id=p.subscription_id
WHERE p.provider=? AND p.provider_payment_id=?`,
		string(event.Provider), data.RefundedPaymentID,
	).Scan(
		&paymentID, &subscriptionID, &userID, &productID, &providerPaymentID,
		&amount, &refunded, &currency, &providerSubscriptionID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSubscriptionNotFound
		}
		return fmt.Errorf("liki: load subscription payment for refund: %w", err)
	}
	var refundSeen int
	if err := s.db.QueryRowContext(ctx, `
SELECT 1
FROM chat_subscription_payment_refunds
WHERE provider=? AND provider_refund_id=?`,
		string(event.Provider), refundID,
	).Scan(&refundSeen); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("liki: load subscription refund fact: %w", err)
	}
	if refundSeen == 1 {
		return nil
	}
	if refunded+data.Amount > amount {
		return fmt.Errorf("liki: subscription refund exceeds payment amount: refunded %d requested %d total %d",
			refunded, data.Amount, amount)
	}
	providerSubscriptionIDOrLocal := providerSubscriptionID.String
	if providerSubscriptionIDOrLocal == "" {
		providerSubscriptionIDOrLocal = subscriptionID
	}
	// Referral intake is idempotent, so calling before the local fact write
	// makes a crash between the two writes safe to retry.
	if refundObserver != nil {
		intent := payment.PaymentIntent{
			ID:              providerPaymentID,
			BusinessOrderID: providerSubscriptionIDOrLocal,
			UserID:          userID,
			Product:         productID,
			Provider:        event.Provider,
			Status:          payment.IntentPaid,
			AmountMinor:     amount,
			Currency:        currency,
		}
		if err := refundObserver(context.WithoutCancel(ctx), event, intent); err != nil {
			return err
		}
	}
	now := s.nowFunc()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("liki: begin subscription refund: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	refundResult, err := tx.ExecContext(ctx, `
INSERT INTO chat_subscription_payment_refunds
(id,payment_id,provider,provider_refund_id,amount_minor_units,currency_code,created_at)
VALUES (?,?,?,?,?,?,?)
ON CONFLICT(provider,provider_refund_id) DO NOTHING`,
		s.idFunc(), paymentID, string(event.Provider), refundID,
		data.Amount, currency, formatSubscriptionTime(now),
	)
	if err != nil {
		return fmt.Errorf("liki: record subscription refund fact: %w", err)
	}
	if inserted, _ := refundResult.RowsAffected(); inserted == 0 {
		return nil
	}
	result, err := tx.ExecContext(ctx, `
UPDATE chat_subscription_payments
SET refunded_minor_units=refunded_minor_units+?,
    status=CASE WHEN refunded_minor_units+? >= amount_minor_units THEN 'refunded' ELSE status END,
    updated_at=?
WHERE id=? AND refunded_minor_units+?<=amount_minor_units`,
		data.Amount, data.Amount, formatSubscriptionTime(now), paymentID, data.Amount,
	)
	if err != nil {
		return fmt.Errorf("liki: update subscription refund: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("liki: subscription refund was not applied")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("liki: commit subscription refund: %w", err)
	}
	return nil
}

func (s *SubscriptionStore) ListPaymentsByUser(ctx context.Context, userID string, limit int) ([]SubscriptionPayment, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id,subscription_id,user_id,product_id,provider,provider_payment_id,
       amount_minor_units,refunded_minor_units,currency_code,period_start,period_end,
       status,created_at,updated_at
FROM chat_subscription_payments WHERE user_id=?
ORDER BY created_at DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]SubscriptionPayment, 0)
	for rows.Next() {
		var item SubscriptionPayment
		var periodStart, periodEnd, createdAt, updatedAt sql.NullString
		var productID, provider, currency, status string
		if err := rows.Scan(
			&item.ID, &item.SubscriptionID, &item.UserID, &productID, &provider,
			&item.ProviderPaymentID, &item.AmountMinorUnits, &item.RefundedMinorUnits,
			&currency, &periodStart, &periodEnd, &status, &createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		item.ProductID = Product(productID)
		item.Provider = payment.Provider(provider)
		item.CurrencyCode = CurrencyCode(currency)
		item.Status = status
		item.PeriodStart = parseNullableSubscriptionTime(periodStart.String)
		item.PeriodEnd = parseNullableSubscriptionTime(periodEnd.String)
		item.CreatedAt = parseSubscriptionTime(createdAt.String)
		item.UpdatedAt = parseSubscriptionTime(updatedAt.String)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SubscriptionStore) SetCancellation(
	ctx context.Context, userID, subscriptionID string, cancel bool,
	gateway payment.SubscriptionCancellationGateway,
) error {
	item, err := s.GetByUser(ctx, userID, subscriptionID)
	if err != nil {
		return err
	}
	if item.ProviderSubscriptionID == "" {
		return fmt.Errorf("liki: subscription has no provider id")
	}
	if err := gateway.SetSubscriptionCancellation(ctx, item.ProviderSubscriptionID, cancel); err != nil {
		return fmt.Errorf("liki: update provider cancellation: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
UPDATE chat_subscriptions SET cancel_at_period_end=?,updated_at=? WHERE id=?`,
		boolInt(cancel), formatSubscriptionTime(s.nowFunc()), item.ID)
	return err
}

// AnonymizeUser retains provider billing IDs and amounts for audit while
// removing the local contact email from subscription records.
func (s *SubscriptionStore) AnonymizeUser(ctx context.Context, userID, oldEmail, tombstone string) (int64, error) {
	if userID == "" || tombstone == "" {
		return 0, fmt.Errorf("liki: anonymize subscriptions requires user and tombstone")
	}
	res, err := s.db.ExecContext(ctx, `
UPDATE chat_subscriptions SET email=?
WHERE user_id=? OR (? != '' AND email=?)`, tombstone, userID, oldEmail, oldEmail)
	if err != nil {
		return 0, fmt.Errorf("liki: anonymize subscriptions: %w", err)
	}
	return res.RowsAffected()
}

func (s *SubscriptionStore) findByProviderOrLocalKey(ctx context.Context, providerID, localKey string) (Subscription, error) {
	if localKey != "" {
		row := s.db.QueryRowContext(ctx, `SELECT `+subscriptionColumns()+` FROM chat_subscriptions WHERE id=?`, localKey)
		item, err := scanSubscription(row.Scan)
		if err == nil {
			return item, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return Subscription{}, err
		}
	}
	query := `SELECT ` + subscriptionColumns() + ` FROM chat_subscriptions WHERE provider_subscription_id=?`
	arg := any(providerID)
	row := s.db.QueryRowContext(ctx, query, arg)
	item, err := scanSubscription(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return Subscription{}, nil
	}
	return item, err
}

func subscriptionColumns() string {
	return `id,user_id,email,product_id,billing_mode,provider,provider_subscription_id,
	        provider_customer_id,provider_checkout_id,status,currency_code,
	        current_period_start,current_period_end,next_billing_at,
	        cancel_at_period_end,cancelled_at,provider_synced_at,provider_period_count,
	        created_at,updated_at`
}

func scanSubscription(scan func(dest ...any) error) (Subscription, error) {
	var item Subscription
	var providerSubscriptionID, providerCustomerID, providerCheckoutID sql.NullString
	var productID, billingMode, provider, status, currency string
	var periodStart, periodEnd, nextBillingAt, cancelledAt, syncedAt sql.NullString
	var cancelAtPeriodEnd int
	var createdAt, updatedAt string
	if err := scan(
		&item.ID, &item.UserID, &item.Email, &productID, &billingMode, &provider,
		&providerSubscriptionID, &providerCustomerID, &providerCheckoutID, &status,
		&currency, &periodStart, &periodEnd, &nextBillingAt, &cancelAtPeriodEnd,
		&cancelledAt, &syncedAt, &item.ProviderPeriodCount, &createdAt, &updatedAt,
	); err != nil {
		return Subscription{}, err
	}
	item.ProductID = Product(productID)
	item.BillingMode = BillingMode(billingMode)
	item.Provider = payment.Provider(provider)
	item.ProviderSubscriptionID = providerSubscriptionID.String
	item.ProviderCustomerID = providerCustomerID.String
	item.ProviderCheckoutID = providerCheckoutID.String
	item.Status = SubscriptionStatus(status)
	item.CurrencyCode = CurrencyCode(currency)
	item.CurrentPeriodStart = parseNullableSubscriptionTime(periodStart.String)
	item.CurrentPeriodEnd = parseNullableSubscriptionTime(periodEnd.String)
	item.NextBillingAt = parseNullableSubscriptionTime(nextBillingAt.String)
	item.CancelAtPeriodEnd = cancelAtPeriodEnd == 1
	item.CancelledAt = parseNullableSubscriptionTime(cancelledAt.String)
	item.ProviderSyncedAt = parseNullableSubscriptionTime(syncedAt.String)
	item.CreatedAt = parseSubscriptionTime(createdAt)
	item.UpdatedAt = parseSubscriptionTime(updatedAt)
	return item, nil
}

func (s Subscription) providerSyncedAtOr(fallback time.Time) time.Time {
	if s.ProviderSyncedAt == nil {
		return fallback
	}
	return *s.ProviderSyncedAt
}

func subscriptionStatusFor(eventType, providerStatus string) SubscriptionStatus {
	switch eventType {
	case payment.EventSubscriptionActive, payment.EventSubscriptionRenewed:
		return SubscriptionActive
	case payment.EventSubscriptionUpdated:
		switch providerStatus {
		case "active", "on_hold", "cancelled", "expired", "failed":
			return SubscriptionStatus(providerStatus)
		default:
			return SubscriptionOnHold
		}
	case payment.EventSubscriptionCancellationScheduled:
		return SubscriptionActive
	case payment.EventSubscriptionOnHold:
		return SubscriptionOnHold
	case payment.EventSubscriptionCancelled:
		return SubscriptionCancelled
	case payment.EventSubscriptionExpired:
		return SubscriptionExpired
	case payment.EventSubscriptionFailed:
		return SubscriptionFailed
	default:
		return SubscriptionFailed
	}
}

func formatSubscriptionTime(value time.Time) string { return value.UTC().Format(time.RFC3339) }

func formatNullableSubscriptionTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatSubscriptionTime(*value)
}

func parseSubscriptionTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339, value)
	return parsed
}

func parseNullableSubscriptionTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed := parseSubscriptionTime(value)
	return &parsed
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
