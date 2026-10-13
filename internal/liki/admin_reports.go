package liki

import (
	"context"
	"database/sql"
	"time"
)

// AdminMoney is an aggregated accounting total. It intentionally avoids user
// identifiers so finance review can consume aggregates without widening PII
// access.
type AdminMoney struct {
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor"`
}

// CommerceFinanceSummary is the liki-domain half of the admin finance report.
type CommerceFinanceSummary struct {
	PeriodStart          time.Time    `json:"period_start"`
	PeriodEnd            time.Time    `json:"period_end"`
	OneTimePaid          []AdminMoney `json:"one_time_paid"`
	SubscriptionPaid     []AdminMoney `json:"subscription_paid"`
	SubscriptionRefunded []AdminMoney `json:"subscription_refunded"`
}

// CommerceStores groups the two stores that share liki.db and feed finance
// reporting. Keeping the group explicit prevents admin from querying either
// store through a generic database handle.
type CommerceStores struct {
	Orders        *OrderStore
	Subscriptions *SubscriptionStore
}

func NewCommerceStores(orders *OrderStore, subscriptions *SubscriptionStore) *CommerceStores {
	return &CommerceStores{Orders: orders, Subscriptions: subscriptions}
}

// AdminFinanceSummary aggregates realized revenue and refunds in [start,end).
// Canceled and pending orders are intentionally excluded: they are not revenue.
func (s *CommerceStores) AdminFinanceSummary(ctx context.Context, start, end time.Time) (CommerceFinanceSummary, error) {
	summary := CommerceFinanceSummary{PeriodStart: start.UTC(), PeriodEnd: end.UTC()}
	if s == nil || s.Orders == nil {
		return summary, nil
	}
	oneTime, err := sumMoney(ctx, s.Orders.db, `
SELECT currency,COALESCE(SUM(amount_minor),0)
FROM chat_orders
WHERE status='paid' AND paid_at>=? AND paid_at<?
GROUP BY currency ORDER BY currency`, start, end)
	if err != nil {
		return summary, err
	}
	summary.OneTimePaid = oneTime

	if s.Subscriptions == nil {
		return summary, nil
	}
	paid, err := sumMoney(ctx, s.Subscriptions.db, `
SELECT currency_code,COALESCE(SUM(amount_minor_units),0)
FROM chat_subscription_payments
WHERE status='paid' AND created_at>=? AND created_at<?
GROUP BY currency_code ORDER BY currency_code`, start, end)
	if err != nil {
		return summary, err
	}
	summary.SubscriptionPaid = paid

	refunded, err := sumMoney(ctx, s.Subscriptions.db, `
SELECT p.currency_code,COALESCE(SUM(r.amount_minor_units),0)
FROM chat_subscription_payment_refunds r
JOIN chat_subscription_payments p ON p.id=r.payment_id
WHERE r.created_at>=? AND r.created_at<?
GROUP BY p.currency_code ORDER BY p.currency_code`, start, end)
	if err != nil {
		return summary, err
	}
	summary.SubscriptionRefunded = refunded
	return summary, nil
}

func sumMoney(ctx context.Context, db *sql.DB, query string, start, end time.Time) ([]AdminMoney, error) {
	rows, err := db.QueryContext(ctx, query, start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]AdminMoney, 0)
	for rows.Next() {
		var item AdminMoney
		if err := rows.Scan(&item.Currency, &item.AmountMinor); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
