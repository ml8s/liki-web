package referral

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAdminFinanceSummaryAggregatesRealReferralTables(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "referral.db"), []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 7)
	outside := start.AddDate(0, 0, -1).Format(time.RFC3339)
	inside := start.Add(24 * time.Hour).Format(time.RFC3339)
	endBoundary := end.Format(time.RFC3339)
	now := inside

	// USD commission in period is net of refund. USD voided and out-of-period
	// commission must not be counted. CNY proves currencies stay separate.
	inserts := []struct {
		sql    string
		args   []any
		mustBe []string
	}{
		{sql: `
INSERT INTO referral_commission_entries
(id,commission_id,beneficiary_id,beneficiary_role,amount_minor,refunded_minor,currency,status,created_at,updated_at)
VALUES ('entry-in','commission-in','partner-1','kol',500,100,'USD','pending',?,?)`,
			args: []any{inside, now}},
		{sql: `
INSERT INTO referral_commission_entries
(id,commission_id,beneficiary_id,beneficiary_role,amount_minor,refunded_minor,currency,status,created_at,updated_at)
VALUES ('entry-void','commission-void','partner-1','kol',999,0,'USD','voided',?,?)`,
			args: []any{inside, now}},
		{sql: `
INSERT INTO referral_commission_entries
(id,commission_id,beneficiary_id,beneficiary_role,amount_minor,refunded_minor,currency,status,created_at,updated_at)
VALUES ('entry-out','commission-out','partner-1','kol',1000,0,'USD','pending',?,?)`,
			args: []any{outside, outside}},
		{sql: `
INSERT INTO referral_commission_entries
(id,commission_id,beneficiary_id,beneficiary_role,amount_minor,refunded_minor,currency,status,created_at,updated_at)
VALUES ('entry-end','commission-end','partner-1','kol',700,0,'CNY','pending',?,?)`,
			args: []any{endBoundary, endBoundary}},
		{sql: `
INSERT INTO referral_payouts
(id,referrer_id,beneficiary_role,amount_minor,requested_amount_minor,currency,method,account,status,created_at,updated_at,paid_at)
VALUES ('payout-paid','partner-1','kol',300,300,'USD','alipay','sealed','paid',?, ?, ?)`,
			args: []any{inside, now, inside}},
		{sql: `
INSERT INTO referral_payouts
(id,referrer_id,beneficiary_role,amount_minor,requested_amount_minor,currency,method,account,status,created_at,updated_at,paid_at)
VALUES ('payout-out','partner-1','kol',1000,1000,'USD','alipay','sealed','paid',?, ?, ?)`,
			args: []any{outside, outside, outside}},
		{sql: `
INSERT INTO referral_payouts
(id,referrer_id,beneficiary_role,amount_minor,requested_amount_minor,currency,method,account,status,created_at,updated_at)
VALUES ('payout-pending','partner-1','kol',700,700,'USD','paypal','sealed','requested',?,?)`,
			args: []any{outside, outside}},
	}
	for _, item := range inserts {
		if _, err := store.db.ExecContext(ctx, item.sql, item.args...); err != nil {
			t.Fatalf("%s: %v", strings.Join(strings.Fields(item.sql), " "), err)
		}
	}

	summary, err := store.AdminFinanceSummary(ctx, start, end)
	if err != nil {
		t.Fatal(err)
	}
	assertAdminMoney(t, summary.CommissionNet, "USD", 400)
	for _, item := range summary.CommissionNet {
		if item.Currency == "CNY" {
			t.Fatalf("end boundary is not exclusive: %+v", summary.CommissionNet)
		}
	}
	assertAdminMoney(t, summary.PayoutPaid, "USD", 300)
	assertAdminMoney(t, summary.PendingPayoutAmount, "USD", 700)
}

func assertAdminMoney(t *testing.T, items []AdminMoney, currency string, amount int64) {
	t.Helper()
	for _, item := range items {
		if item.Currency == currency && item.AmountMinor == amount {
			return
		}
	}
	t.Fatalf("missing %s %d in %+v", currency, amount, items)
}
