package sales

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
)

func (s *SubscriptionStore) ApplyWebhook(ctx context.Context, entitlements EntitlementProvisioner, event payment.WebhookEvent) error {
	if !payment.IsSubscriptionLifecycle(event.Type) {
		return fmt.Errorf("sales: %q is not a subscription lifecycle event", event.Type)
	}
	data := event.Data
	if data.SubscriptionID == "" {
		return fmt.Errorf("sales: subscription webhook has no provider subscription id")
	}
	localKey := data.Metadata[payment.MetadataLikiSubscriptionKey]
	existing, err := s.findByProviderOrLocalKey(ctx, data.SubscriptionID, localKey)
	if err != nil {
		return err
	}

	userID := data.Metadata[payment.MetadataLikiUserID]
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
			return fmt.Errorf("sales: cannot attach webhook to unknown subscription %s", data.SubscriptionID)
		}
		existing.ID = localKey
		existing.UserID = userID
		existing.Email = email
		existing.ProductID = product.Product(orDefault(data.Metadata[payment.MetadataLikiProduct], string(product.Chat)))
		if _, err := product.DefinitionFor(existing.ProductID); err != nil {
			return err
		}
		existing.BillingMode = product.BillingModeSubscription
		existing.Provider = event.Provider
		existing.Status = SubscriptionPending
		existing.CurrencyCode = product.Currency(orDefault(data.Currency, "USD"))
		existing.CreatedAt = syncedAt
	}

	if existing.ID != "" {
		_, err = s.db.ExecContext(ctx, `
UPDATE sales_subscriptions SET
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
			return fmt.Errorf("sales: update subscription: %w", err)
		}
	}

	if status == SubscriptionActive && data.PeriodStart != nil && data.PeriodEnd != nil {
		err = entitlements.ProvisionSubscriptionPeriod(
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
		return fmt.Errorf("sales: subscription payment payload is incomplete")
	}
	subscription, err := s.findByProviderOrLocalKey(ctx, data.SubscriptionID, data.Metadata[payment.MetadataLikiSubscriptionKey])
	if err != nil {
		return err
	}
	userID := data.Metadata[payment.MetadataLikiUserID]
	if userID == "" {
		userID = subscription.UserID
	}
	now := s.nowFunc()
	id := s.idFunc()
	result, err := s.db.ExecContext(ctx, `
INSERT INTO sales_subscription_payments
(id,subscription_id,user_id,product_id,provider,provider_payment_id,
 amount_minor_units,currency_code,period_start,period_end,status,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(provider,provider_payment_id) DO NOTHING`,
		id, subscription.ID, userID, string(subscription.ProductID), string(event.Provider),
		data.PaymentID, data.Amount, string(subscription.CurrencyCode),
		formatNullableSubscriptionTime(data.PeriodStart),
		formatNullableSubscriptionTime(data.PeriodEnd),
		string(SubscriptionPaymentPaid), formatSubscriptionTime(now), formatSubscriptionTime(now))
	if err != nil {
		return fmt.Errorf("sales: record subscription payment: %w", err)
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
FROM sales_subscription_payments
WHERE provider=? AND provider_payment_id=?`,
			string(event.Provider), event.Data.PaymentID,
		).Scan(&notified); err != nil {
			return false, fmt.Errorf("sales: load subscription payment observer state: %w", err)
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
UPDATE sales_subscription_payments
SET observer_enqueued_at=?,updated_at=?
WHERE provider=? AND provider_payment_id=? AND observer_enqueued_at IS NULL`,
		now, now, string(event.Provider), event.Data.PaymentID); err != nil {
		return true, fmt.Errorf("sales: mark subscription payment observer: %w", err)
	}
	return true, nil
}

func (s *SubscriptionStore) RecordPaymentRefund(
	ctx context.Context, event payment.WebhookEvent,
	refundObserver func(context.Context, payment.WebhookEvent, payment.CommissionablePayment) error,
) error {
	data := event.Data
	refundID := data.RefundID
	if refundID == "" {
		refundID = data.PaymentID
	}
	if event.Type != payment.EventPaymentRefunded || refundID == "" || data.RefundedPaymentID == "" || data.Amount <= 0 {
		return fmt.Errorf("sales: subscription refund payload is incomplete")
	}
	var paymentID, subscriptionID, userID, productID, providerPaymentID, currency string
	var amount, refunded int64
	var providerSubscriptionID sql.NullString
	if err := s.db.QueryRowContext(ctx, `
SELECT p.id,p.subscription_id,p.user_id,p.product_id,p.provider_payment_id,
       p.amount_minor_units,p.refunded_minor_units,p.currency_code,
       COALESCE(sub.provider_subscription_id,'')
FROM sales_subscription_payments p
       JOIN sales_subscriptions sub ON sub.id=p.subscription_id
WHERE p.provider=? AND p.provider_payment_id=?`,
		string(event.Provider), data.RefundedPaymentID,
	).Scan(
		&paymentID, &subscriptionID, &userID, &productID, &providerPaymentID,
		&amount, &refunded, &currency, &providerSubscriptionID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSubscriptionNotFound
		}
		return fmt.Errorf("sales: load subscription payment for refund: %w", err)
	}
	var refundSeen int
	if err := s.db.QueryRowContext(ctx, `
SELECT 1
FROM sales_subscription_payment_refunds
WHERE provider=? AND provider_refund_id=?`,
		string(event.Provider), refundID,
	).Scan(&refundSeen); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("sales: load subscription refund fact: %w", err)
	}
	if refundSeen == 1 {
		return nil
	}
	if refunded+data.Amount > amount {
		return fmt.Errorf("sales: subscription refund exceeds payment amount: refunded %d requested %d total %d",
			refunded, data.Amount, amount)
	}
	providerSubscriptionIDOrLocal := providerSubscriptionID.String
	if providerSubscriptionIDOrLocal == "" {
		providerSubscriptionIDOrLocal = subscriptionID
	}
	// Referral intake is idempotent, so calling before the local fact write
	// makes a crash between the two writes safe to retry.
	if refundObserver != nil {
		sale := payment.CommissionablePayment{
			ID:              providerPaymentID,
			BusinessOrderID: providerSubscriptionIDOrLocal,
			UserID:          userID,
			Product:         productID,
			Provider:        event.Provider,
			AmountMinor:     amount,
			Currency:        currency,
		}
		if err := refundObserver(context.WithoutCancel(ctx), event, sale); err != nil {
			return err
		}
	}
	now := s.nowFunc()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sales: begin subscription refund: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	refundResult, err := tx.ExecContext(ctx, `
INSERT INTO sales_subscription_payment_refunds
(id,payment_id,provider,provider_refund_id,amount_minor_units,currency_code,created_at)
VALUES (?,?,?,?,?,?,?)
ON CONFLICT(provider,provider_refund_id) DO NOTHING`,
		s.idFunc(), paymentID, string(event.Provider), refundID,
		data.Amount, currency, formatSubscriptionTime(now),
	)
	if err != nil {
		return fmt.Errorf("sales: record subscription refund fact: %w", err)
	}
	if inserted, _ := refundResult.RowsAffected(); inserted == 0 {
		return nil
	}
	result, err := tx.ExecContext(ctx, `
UPDATE sales_subscription_payments
SET refunded_minor_units=refunded_minor_units+?,
    status=CASE WHEN refunded_minor_units+? >= amount_minor_units THEN ? ELSE status END,
    updated_at=?
WHERE id=? AND refunded_minor_units+?<=amount_minor_units`,
		data.Amount, data.Amount, string(SubscriptionPaymentRefunded), formatSubscriptionTime(now), paymentID, data.Amount,
	)
	if err != nil {
		return fmt.Errorf("sales: update subscription refund: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("sales: subscription refund was not applied")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sales: commit subscription refund: %w", err)
	}
	return nil
}
