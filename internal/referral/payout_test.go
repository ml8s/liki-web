package referral

import (
	"context"
	"errors"
	"testing"

	"liki-web/internal/payment"
)

func newPendingCommissionPayout(t *testing.T) (*Store, Commission) {
	t.Helper()
	ctx := context.Background()
	store := newStore(t)
	kol, err := store.RegisterReferrer(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Attribute(ctx, "buyer", kol.Code); err != nil {
		t.Fatal(err)
	}
	intent := paidIntent(payment.ProviderXunhu, "buyer", 10000)
	intent.ID = "pi-payout"
	if _, _, err := store.EnqueueCommissionIntent(ctx, intent); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionIntents(ctx, 10); err != nil || n != 1 {
		t.Fatalf("processed=%d err=%v", n, err)
	}
	commission, err := store.commission(ctx, string(payment.ProviderXunhu), intent.ID)
	if err != nil {
		t.Fatal(err)
	}
	return store, commission
}

func TestCreatePayoutLocksPendingCommissions(t *testing.T) {
	ctx := context.Background()
	store, commission := newPendingCommissionPayout(t)
	payout, err := store.CreatePayout(ctx, commission.ReferrerID, PayoutMethodAlipay, "me@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if payout.Status != PayoutRequested || payout.AmountMinor != 5000 {
		t.Fatalf("payout = %+v", payout)
	}
	if _, err := store.CreatePayout(ctx, commission.ReferrerID, PayoutMethodAlipay, "again@example.com"); !errors.Is(err, ErrNoWithdrawableCommission) {
		t.Fatalf("second payout err = %v", err)
	}
}

func TestRejectPayoutReleasesCommissions(t *testing.T) {
	ctx := context.Background()
	store, commission := newPendingCommissionPayout(t)
	payout, err := store.CreatePayout(ctx, commission.ReferrerID, PayoutMethodAlipay, "me@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DecidePayout(ctx, payout.ID, "admin", false, "invalid account"); err != nil {
		t.Fatal(err)
	}
	got, err := store.PayoutByID(ctx, payout.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != PayoutRejected {
		t.Fatalf("payout = %+v", got)
	}
	if _, err := store.CreatePayout(ctx, commission.ReferrerID, PayoutMethodAlipay, "fixed@example.com"); err != nil {
		t.Fatalf("commission was not released: %v", err)
	}
}

func TestMarkPayoutPaidMarksCommissions(t *testing.T) {
	ctx := context.Background()
	store, commission := newPendingCommissionPayout(t)
	payout, err := store.CreatePayout(ctx, commission.ReferrerID, PayoutMethodAlipay, "me@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ApprovePayout(ctx, payout.ID, "admin"); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkPayoutPaid(ctx, payout.ID, "admin", "transfer-1"); err != nil {
		t.Fatal(err)
	}
	got, err := store.PayoutByID(ctx, payout.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != PayoutPaid || got.PaidAt == nil || got.PaymentReference != "transfer-1" {
		t.Fatalf("payout = %+v", got)
	}
	var entryStatus string
	err = store.db.QueryRowContext(ctx, `
SELECT status FROM referral_commission_entries WHERE id=?`, commission.ID+":kol").Scan(&entryStatus)
	if err != nil {
		t.Fatal(err)
	}
	if entryStatus != string(CommissionPaid) {
		t.Fatalf("entry status = %q, want paid", entryStatus)
	}
}

func TestCreatePayoutRejectsMixedCurrencies(t *testing.T) {
	ctx := context.Background()
	store, commission := newPendingCommissionPayout(t)
	kol, err := store.ReferrerByUser(ctx, commission.ReferrerID)
	if err != nil {
		t.Fatal(err)
	}
	intent := paidIntent(payment.ProviderDodo, "buyer-usd", 1000)
	intent.ID = "pi-payout-usd"
	intent.Currency = "USD"
	if _, err := store.Attribute(ctx, "buyer-usd", kol.Code); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.EnqueueCommissionIntent(ctx, intent); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionIntents(ctx, 10); err != nil || n != 1 {
		t.Fatalf("processed=%d err=%v", n, err)
	}
	if _, err := store.CreatePayout(ctx, kol.UserID, PayoutMethodAlipay, "me@example.com"); !errors.Is(err, ErrMixedPayoutCurrency) {
		t.Fatalf("err = %v, want ErrMixedPayoutCurrency", err)
	}
}

func TestMarkPayoutPaidRequiresApproval(t *testing.T) {
	ctx := context.Background()
	store, commission := newPendingCommissionPayout(t)
	payout, err := store.CreatePayout(ctx, commission.ReferrerID, PayoutMethodAlipay, "me@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MarkPayoutPaid(ctx, payout.ID, "admin", "transfer-1"); !errors.Is(err, ErrPayoutInvalidTransition) {
		t.Fatalf("err = %v, want ErrPayoutInvalidTransition", err)
	}
}

func TestKOLAndScoutPayoutsAreIndependent(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	scout, err := store.RegisterReferrer(ctx, "scout")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Attribute(ctx, "kol", scout.Code); err != nil {
		t.Fatal(err)
	}
	kol, err := store.RegisterReferrer(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	if kol.RecruiterID != "scout" {
		t.Fatalf("recruiter = %q, want scout", kol.RecruiterID)
	}
	if err := store.PromoteToScout(ctx, "scout", "admin", "approved partner"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Attribute(ctx, "buyer", kol.Code); err != nil {
		t.Fatal(err)
	}
	intent := paidIntent(payment.ProviderXunhu, "buyer", 50000)
	if _, _, err := store.EnqueueCommissionIntent(ctx, intent); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionIntents(ctx, 10); err != nil || n != 1 {
		t.Fatalf("processed=%d err=%v", n, err)
	}

	scoutPayout, err := store.CreatePayoutForRole(ctx, "scout", RoleScout, PayoutMethodAlipay, "scout@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if scoutPayout.AmountMinor != 10000 {
		t.Fatalf("scout payout = %+v", scoutPayout)
	}
	kolPayout, err := store.CreatePayoutForRole(ctx, kol.UserID, RoleKOL, PayoutMethodAlipay, "kol@example.com")
	if err != nil {
		t.Fatalf("kol payout was blocked by scout payout: %v", err)
	}
	if kolPayout.AmountMinor != 25000 {
		t.Fatalf("kol payout = %+v", kolPayout)
	}
	if err := store.ApprovePayout(ctx, scoutPayout.ID, "admin"); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkPayoutPaid(ctx, scoutPayout.ID, "admin", "scout-transfer"); err != nil {
		t.Fatal(err)
	}
	var kolStatus, scoutStatus string
	if err := store.db.QueryRowContext(ctx, `SELECT status FROM referral_commission_entries WHERE beneficiary_role='kol'`).Scan(&kolStatus); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT status FROM referral_commission_entries WHERE beneficiary_role='scout'`).Scan(&scoutStatus); err != nil {
		t.Fatal(err)
	}
	if kolStatus != string(CommissionPending) || scoutStatus != string(CommissionPaid) {
		t.Fatalf("statuses kol/scout = %q/%q", kolStatus, scoutStatus)
	}
}
