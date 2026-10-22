package sales

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
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

type SubscriptionPaymentStatus string

const (
	SubscriptionPaymentPaid     SubscriptionPaymentStatus = "paid"
	SubscriptionPaymentRefunded SubscriptionPaymentStatus = "refunded"
)

var (
	ErrSubscriptionNotFound = errors.New("sales: subscription not found")
)

type Subscription struct {
	ID                        string              `json:"id"`
	UserID                    string              `json:"user_id"`
	Email                     string              `json:"email"`
	ProductID                 product.Product     `json:"product_id"`
	BillingMode               product.BillingMode `json:"billing_mode"`
	Provider                  payment.Provider    `json:"provider"`
	ProviderSubscriptionID    string              `json:"provider_subscription_id"`
	ProviderCustomerID        string              `json:"provider_customer_id"`
	ProviderCheckoutSessionID string              `json:"provider_checkout_id"`
	Status                    SubscriptionStatus  `json:"status"`
	CurrencyCode              product.Currency    `json:"currency_code"`
	CurrentPeriodStart        *time.Time          `json:"current_period_start"`
	CurrentPeriodEnd          *time.Time          `json:"current_period_end"`
	NextBillingAt             *time.Time          `json:"next_billing_at"`
	CancelAtPeriodEnd         bool                `json:"cancel_at_period_end"`
	CancelledAt               *time.Time          `json:"cancelled_at"`
	ProviderSyncedAt          *time.Time          `json:"provider_synced_at"`
	ProviderPeriodCount       int64               `json:"provider_period_count"`
	CreatedAt                 time.Time           `json:"created_at"`
	UpdatedAt                 time.Time           `json:"updated_at"`
}

type SubscriptionPayment struct {
	ID                 string                    `json:"id"`
	SubscriptionID     string                    `json:"subscription_id"`
	UserID             string                    `json:"user_id"`
	ProductID          product.Product           `json:"product_id"`
	Provider           payment.Provider          `json:"provider"`
	ProviderPaymentID  string                    `json:"provider_payment_id"`
	AmountMinorUnits   int64                     `json:"amount_minor_units"`
	RefundedMinorUnits int64                     `json:"refunded_minor_units"`
	CurrencyCode       product.Currency          `json:"currency_code"`
	Status             SubscriptionPaymentStatus `json:"status"`
	PeriodStart        *time.Time                `json:"period_start"`
	PeriodEnd          *time.Time                `json:"period_end"`
	CreatedAt          time.Time                 `json:"created_at"`
	UpdatedAt          time.Time                 `json:"updated_at"`
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

func (s *SubscriptionStore) CreatePending(
	ctx context.Context, userID, email string, productID product.Product, currency product.Currency,
) (Subscription, error) {
	userID = strings.TrimSpace(userID)
	email = strings.ToLower(strings.TrimSpace(email))
	if userID == "" || email == "" {
		return Subscription{}, fmt.Errorf("sales: subscription requires user and email")
	}
	if _, err := product.DefinitionFor(productID); err != nil {
		return Subscription{}, err
	}
	if currency != product.USD {
		return Subscription{}, fmt.Errorf("sales: subscription does not support currency %s", currency)
	}
	now := s.nowFunc()
	item := Subscription{
		ID: s.idFunc(), UserID: userID, Email: email, ProductID: productID,
		BillingMode: product.BillingModeSubscription, Provider: payment.ProviderDodo,
		Status: SubscriptionPending, CurrencyCode: currency,
		CreatedAt: now, UpdatedAt: now,
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO sales_subscriptions
(id,user_id,email,product_id,billing_mode,provider,status,currency_code,
 cancel_at_period_end,created_at,updated_at)
VALUES (?,?,?,?,?,?,?, ?,0,?,?)`,
		item.ID, item.UserID, item.Email, string(item.ProductID), string(item.BillingMode),
		string(item.Provider), string(item.Status), string(item.CurrencyCode),
		formatSubscriptionTime(now), formatSubscriptionTime(now))
	if err != nil {
		return Subscription{}, fmt.Errorf("sales: create subscription: %w", err)
	}
	return item, nil
}

func (s *SubscriptionStore) AttachCheckout(ctx context.Context, userID, subscriptionID, checkoutID string) error {
	res, err := s.db.ExecContext(ctx, `
UPDATE sales_subscriptions
SET provider_checkout_id=?,updated_at=?
WHERE id=? AND user_id=? AND status=?`,
		checkoutID, formatSubscriptionTime(s.nowFunc()), subscriptionID, userID, string(SubscriptionPending))
	if err != nil {
		return fmt.Errorf("sales: attach subscription checkout: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

func (s *SubscriptionStore) GetByUser(ctx context.Context, userID, subscriptionID string) (Subscription, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT `+subscriptionColumns()+`
FROM sales_subscriptions WHERE id=? AND user_id=?`, subscriptionID, userID)
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
FROM sales_subscriptions
WHERE user_id=? AND status IN (?,?)`,
		userID, string(SubscriptionActive), string(SubscriptionOnHold),
	).Scan(&count); err != nil {
		return fmt.Errorf("sales: check blocking subscription: %w", err)
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
FROM sales_subscriptions WHERE user_id=?
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
FROM sales_subscriptions
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

func (s *SubscriptionStore) ListPaymentsByUser(ctx context.Context, userID string, limit int) ([]SubscriptionPayment, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id,subscription_id,user_id,product_id,provider,provider_payment_id,
       amount_minor_units,refunded_minor_units,currency_code,period_start,period_end,
       status,created_at,updated_at
FROM sales_subscription_payments WHERE user_id=?
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
		item.ProductID = product.Product(productID)
		item.Provider = payment.Provider(provider)
		item.CurrencyCode = product.Currency(currency)
		item.Status = SubscriptionPaymentStatus(status)
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
		return fmt.Errorf("sales: subscription has no provider id")
	}
	if err := gateway.SetSubscriptionCancellation(ctx, item.ProviderSubscriptionID, cancel); err != nil {
		return fmt.Errorf("sales: update provider cancellation: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
UPDATE sales_subscriptions SET cancel_at_period_end=?,updated_at=? WHERE id=?`,
		boolInt(cancel), formatSubscriptionTime(s.nowFunc()), item.ID)
	return err
}

// AnonymizeUser retains provider billing IDs and amounts for audit while
// removing the local contact email from subscription records.
func (s *SubscriptionStore) AnonymizeUser(ctx context.Context, userID, oldEmail, tombstone string) (int64, error) {
	if userID == "" || tombstone == "" {
		return 0, fmt.Errorf("sales: anonymize subscriptions requires user and tombstone")
	}
	res, err := s.db.ExecContext(ctx, `
UPDATE sales_subscriptions SET email=?
WHERE user_id=? OR (? != '' AND email=?)`, tombstone, userID, oldEmail, oldEmail)
	if err != nil {
		return 0, fmt.Errorf("sales: anonymize subscriptions: %w", err)
	}
	return res.RowsAffected()
}

func (s *SubscriptionStore) findByProviderOrLocalKey(ctx context.Context, providerID, localKey string) (Subscription, error) {
	if localKey != "" {
		row := s.db.QueryRowContext(ctx, `SELECT `+subscriptionColumns()+` FROM sales_subscriptions WHERE id=?`, localKey)
		item, err := scanSubscription(row.Scan)
		if err == nil {
			return item, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return Subscription{}, err
		}
	}
	query := `SELECT ` + subscriptionColumns() + ` FROM sales_subscriptions WHERE provider_subscription_id=?`
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
	var providerSubscriptionID, providerCustomerID, providerCheckoutSessionID sql.NullString
	var productID, billingMode, provider, status, currency string
	var periodStart, periodEnd, nextBillingAt, cancelledAt, syncedAt sql.NullString
	var cancelAtPeriodEnd int
	var createdAt, updatedAt string
	if err := scan(
		&item.ID, &item.UserID, &item.Email, &productID, &billingMode, &provider,
		&providerSubscriptionID, &providerCustomerID, &providerCheckoutSessionID, &status,
		&currency, &periodStart, &periodEnd, &nextBillingAt, &cancelAtPeriodEnd,
		&cancelledAt, &syncedAt, &item.ProviderPeriodCount, &createdAt, &updatedAt,
	); err != nil {
		return Subscription{}, err
	}
	item.ProductID = product.Product(productID)
	item.BillingMode = product.BillingMode(billingMode)
	item.Provider = payment.Provider(provider)
	item.ProviderSubscriptionID = providerSubscriptionID.String
	item.ProviderCustomerID = providerCustomerID.String
	item.ProviderCheckoutSessionID = providerCheckoutSessionID.String
	item.Status = SubscriptionStatus(status)
	item.CurrencyCode = product.Currency(currency)
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

// subscriptionStatusForEvent is the single projection from a provider
// lifecycle event to the local subscription status.
func subscriptionStatusForEvent(event payment.WebhookEvent) SubscriptionStatus {
	switch event.Type {
	case payment.EventSubscriptionActive, payment.EventSubscriptionRenewed:
		return SubscriptionActive
	case payment.EventSubscriptionUpdated:
		switch event.Data.Status {
		case "active", "on_hold", "cancelled", "expired", "failed":
			return SubscriptionStatus(event.Data.Status)
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

var ErrSubscriptionConflict = errors.New("sales: subscription conflict")
