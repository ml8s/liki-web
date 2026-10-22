package sales

import (
	"context"
	"database/sql"

	"github.com/ml8s/liki-web/internal/migrate"
)

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
	}, {
		ID: "003_canonical_sales_tables",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
ALTER TABLE chat_subscriptions RENAME TO sales_subscriptions;
ALTER TABLE chat_subscription_payments RENAME TO sales_subscription_payments;
ALTER TABLE chat_subscription_payment_refunds RENAME TO sales_subscription_payment_refunds;
DROP INDEX IF EXISTS idx_chat_subscriptions_user;
DROP INDEX IF EXISTS idx_chat_subscriptions_status;
CREATE INDEX idx_sales_subscriptions_user
    ON sales_subscriptions(user_id, created_at DESC);
CREATE INDEX idx_sales_subscriptions_status
    ON sales_subscriptions(status, current_period_end);
DROP INDEX IF EXISTS idx_chat_subscription_payments_user;
CREATE INDEX idx_sales_subscription_payments_user
    ON sales_subscription_payments(user_id, created_at DESC);
DROP INDEX IF EXISTS idx_chat_subscription_payment_refunds_payment;
CREATE INDEX idx_sales_subscription_payment_refunds_payment
    ON sales_subscription_payment_refunds(payment_id);
`)
			return err
		},
	}})
}
