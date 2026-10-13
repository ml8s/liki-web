package referral

import (
	"context"
	"time"
)

// AdminMoney is an aggregated amount for finance review. It contains no member
// or payout account data.
type AdminMoney struct {
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor"`
}

// AdminFinanceSummary is the referral-domain half of the finance report.
type AdminFinanceSummary struct {
	PeriodStart         time.Time    `json:"period_start"`
	PeriodEnd           time.Time    `json:"period_end"`
	CommissionNet       []AdminMoney `json:"commission_net"`
	PayoutPaid          []AdminMoney `json:"payout_paid"`
	PendingPayoutAmount []AdminMoney `json:"pending_payout_amount"`
}

// AdminFinanceSummary aggregates commission created and payouts settled in the
// period. Commission is net of refunds so support does not mistake gross
// commission for payable commission.
func (s *Store) AdminFinanceSummary(ctx context.Context, start, end time.Time) (AdminFinanceSummary, error) {
	summary := AdminFinanceSummary{PeriodStart: start.UTC(), PeriodEnd: end.UTC()}
	commission, err := sumMoney(ctx, s.db, `
SELECT currency,COALESCE(SUM(amount_minor-refunded_minor),0)
FROM referral_commission_entries
WHERE created_at>=? AND created_at<? AND status!='voided'
GROUP BY currency ORDER BY currency`, start, end)
	if err != nil {
		return summary, err
	}
	summary.CommissionNet = commission

	paid, err := sumMoney(ctx, s.db, `
SELECT currency,COALESCE(SUM(amount_minor),0)
FROM referral_payouts
WHERE status='paid' AND paid_at>=? AND paid_at<?
GROUP BY currency ORDER BY currency`, start, end)
	if err != nil {
		return summary, err
	}
	summary.PayoutPaid = paid

	pending, err := sumMoneyAll(ctx, s.db, `
SELECT currency,COALESCE(SUM(amount_minor),0)
FROM referral_payouts
WHERE status IN ('requested','pending','approved')
GROUP BY currency ORDER BY currency`)
	if err != nil {
		return summary, err
	}
	summary.PendingPayoutAmount = pending
	return summary, nil
}
