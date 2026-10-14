package partner

import (
	"context"
	"errors"
	"testing"
)

func TestEraseUserHoldsActivePayout(t *testing.T) {
	ctx := context.Background()
	store, commission := newPendingCommissionPayout(t)
	if _, err := store.CreatePayout(ctx, commission.ReferrerID, PayoutMethodAlipay, "pay@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.EraseUser(ctx, commission.ReferrerID); !errors.Is(err, ErrReferralFinancialHold) {
		t.Fatalf("err=%v, want financial hold", err)
	}
	var account, status string
	if err := store.db.QueryRowContext(ctx, `
SELECT account,status FROM referral_payouts WHERE referrer_id=?`, commission.ReferrerID).Scan(&account, &status); err != nil {
		t.Fatal(err)
	}
	if status != string(PayoutRequested) || account == "erased" {
		t.Fatalf("active payout changed: status=%q account=%q", status, account)
	}
}

func TestEraseUserFreezesPartnerAndPreservesPendingLedger(t *testing.T) {
	ctx := context.Background()
	store, commission := newPendingCommissionPayout(t)
	payout, err := store.CreatePayout(ctx, commission.ReferrerID, PayoutMethodAlipay, "pay@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DecidePayout(ctx, payout.ID, "admin", false, "erasure requested"); err != nil {
		t.Fatal(err)
	}

	res, err := store.EraseUser(ctx, commission.ReferrerID)
	if err != nil {
		t.Fatal(err)
	}
	if res.PayoutsErased != 1 || res.FinanceTasks != 1 {
		t.Fatalf("erasure=%+v", res)
	}

	var code, partnerStatus string
	if err := store.db.QueryRowContext(ctx, `
SELECT code,status FROM referrers WHERE user_id=?`, commission.ReferrerID).Scan(&code, &partnerStatus); err != nil {
		t.Fatal(err)
	}
	if partnerStatus != string(PartnerFrozen) || code == commission.ReferrerID {
		t.Fatalf("partner status=%q code=%q", partnerStatus, code)
	}

	var account, payoutStatus string
	if err := store.db.QueryRowContext(ctx, `
SELECT account,status FROM referral_payouts WHERE id=?`, payout.ID).Scan(&account, &payoutStatus); err != nil {
		t.Fatal(err)
	}
	if account != "erased" || payoutStatus != string(PayoutRejected) {
		t.Fatalf("payout account/status=%q/%q", account, payoutStatus)
	}

	tasks, err := store.ListFinanceTasks(ctx, FinanceTaskPending, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Type != FinanceTaskUserErasure ||
		tasks[0].EntryID != commission.ID+":kol" || tasks[0].AmountMinor != 5000 {
		t.Fatalf("tasks=%+v", tasks)
	}

	if _, err := store.EraseUser(ctx, commission.ReferrerID); err != nil {
		t.Fatalf("repeat erasure err=%v", err)
	}
	tasks, err = store.ListFinanceTasks(ctx, FinanceTaskPending, 10)
	if err != nil || len(tasks) != 1 {
		t.Fatalf("repeat tasks=%+v err=%v", tasks, err)
	}
}
