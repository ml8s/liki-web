package partner

import (
	"context"
	"errors"
	"testing"

	"github.com/ml8s/liki-web/internal/payment"
)

func newPendingCommissionPayout(t *testing.T) (*Store, Commission) {
	t.Helper()
	ctx := context.Background()
	store := newStore(t)
	kol, err := store.createKOLPartner(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Attribute(ctx, "buyer", kol.Code); err != nil {
		t.Fatal(err)
	}
	sale := paidCommission(payment.ProviderXunhu, "buyer", 10000)
	sale.ID = "pi-payout"
	if _, _, err := store.EnqueueCommissionPayment(ctx, sale); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionPayments(ctx, 10); err != nil || n != 1 {
		t.Fatalf("processed=%d err=%v", n, err)
	}
	commission, err := store.commission(ctx, string(payment.ProviderXunhu), sale.ID)
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

func TestListPayoutsIncludesPartnerCode(t *testing.T) {
	ctx := context.Background()
	store, commission := newPendingCommissionPayout(t)
	kol, err := store.PartnerByUser(ctx, commission.ReferrerID)
	if err != nil {
		t.Fatal(err)
	}
	payout, err := store.CreatePayout(ctx, commission.ReferrerID, PayoutMethodAlipay, "me@example.com")
	if err != nil {
		t.Fatal(err)
	}

	itemsPage, err := store.ListPayouts(ctx, PayoutRequested, AdminPageRequest{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(itemsPage.Items) != 1 {
		t.Fatalf("payouts = %+v, want one", itemsPage)
	}
	if itemsPage.Items[0].ID != payout.ID || itemsPage.Items[0].BeneficiaryID != kol.UserID {
		t.Fatalf("payout = %+v, want id=%s referrer=%s", itemsPage.Items[0], payout.ID, kol.UserID)
	}
	if itemsPage.Items[0].BeneficiaryCode != kol.Code {
		t.Fatalf("beneficiary_code = %q, want %q", itemsPage.Items[0].BeneficiaryCode, kol.Code)
	}
}

func TestRetiredRoleCanSettleOnlyItsOwnHistory(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	kol, err := store.createKOLPartner(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Attribute(ctx, "buyer", kol.Code); err != nil {
		t.Fatal(err)
	}
	sale := paidCommission(payment.ProviderXunhu, "buyer", 50000)
	if _, _, err := store.EnqueueCommissionPayment(ctx, sale); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionPayments(ctx, 10); err != nil || n != 1 {
		t.Fatalf("processed=%d err=%v", n, err)
	}
	if _, err := store.db.ExecContext(ctx,
		`UPDATE partner_roles SET status='retired' WHERE user_id=? AND role='kol'`, kol.UserID); err != nil {
		t.Fatal(err)
	}

	states, err := store.PartnerStates(ctx, kol.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if states.KOL.Status != RoleRetired || states.KOL.CanSubmit {
		t.Fatalf("states = %+v", states)
	}
	settlement, err := store.SettlementForRole(ctx, kol.UserID, RoleKOL)
	if err != nil {
		t.Fatal(err)
	}
	if settlement.Status != RoleRetired || len(settlement.Balances) != 1 || settlement.Balances[0].TotalMinor != 25000 {
		t.Fatalf("settlement = %+v", settlement)
	}

	payout, err := store.CreatePayoutForRole(ctx, kol.UserID, RoleKOL, PayoutMethodAlipay, "me@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if payout.AmountMinor != 25000 {
		t.Fatalf("payout = %+v", payout)
	}
	payouts, err := store.ListPayoutsForRole(ctx, kol.UserID, RoleKOL, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(payouts) != 1 || payouts[0].Role != RoleKOL {
		t.Fatalf("payouts = %+v", payouts)
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
SELECT status FROM partner_commission_entries WHERE id=?`, commission.ID+":kol").Scan(&entryStatus)
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
	kol, err := store.PartnerByUser(ctx, commission.ReferrerID)
	if err != nil {
		t.Fatal(err)
	}
	sale := paidCommission(payment.ProviderDodo, "buyer-usd", 1000)
	sale.ID = "pi-payout-usd"
	sale.Currency = "USD"
	if _, err := store.Attribute(ctx, "buyer-usd", kol.Code); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.EnqueueCommissionPayment(ctx, sale); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionPayments(ctx, 10); err != nil || n != 1 {
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
	scout, err := store.CreateScoutPartner(ctx, "scout", "admin", "standalone scout")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Attribute(ctx, "kol", scout.Code); err != nil {
		t.Fatal(err)
	}
	kol, err := store.createKOLPartner(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	if kol.RecruiterID != "scout" {
		t.Fatalf("recruiter = %q, want scout", kol.RecruiterID)
	}
	if _, err := store.Attribute(ctx, "buyer", kol.Code); err != nil {
		t.Fatal(err)
	}
	sale := paidCommission(payment.ProviderXunhu, "buyer", 50000)
	if _, _, err := store.EnqueueCommissionPayment(ctx, sale); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionPayments(ctx, 10); err != nil || n != 1 {
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
	if err := store.db.QueryRowContext(ctx, `SELECT status FROM partner_commission_entries WHERE beneficiary_role='kol'`).Scan(&kolStatus); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT status FROM partner_commission_entries WHERE beneficiary_role='scout'`).Scan(&scoutStatus); err != nil {
		t.Fatal(err)
	}
	if kolStatus != string(CommissionPending) || scoutStatus != string(CommissionPaid) {
		t.Fatalf("statuses kol/scout = %q/%q", kolStatus, scoutStatus)
	}
}

func TestPayoutAccessAllowsRetiredButNotFrozenRole(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	kol, err := store.createKOLPartner(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Attribute(ctx, "buyer", kol.Code); err != nil {
		t.Fatal(err)
	}
	sale := paidCommission(payment.ProviderDodo, "buyer", 50000)
	if _, _, err := store.EnqueueCommissionPayment(ctx, sale); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionPayments(ctx, 10); err != nil || n != 1 {
		t.Fatalf("processed=%d err=%v", n, err)
	}

	if _, err := store.db.ExecContext(ctx, `UPDATE partner_roles SET status='retired' WHERE user_id=? AND role='kol'`, kol.UserID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.KOLByID(ctx, kol.UserID); err == nil {
		t.Fatal("retired role unexpectedly grants console access")
	}
	if _, err := store.PartnerForPayout(ctx, kol.UserID, RoleKOL); err != nil {
		t.Fatalf("retired role cannot request payout: %v", err)
	}
	payout, err := store.CreatePayoutForRole(ctx, kol.UserID, RoleKOL, PayoutMethodAlipay, "payout@example.com")
	if err != nil {
		t.Fatalf("retired role payout = %v", err)
	}
	if payout.AmountMinor != 25000 {
		t.Fatalf("payout = %+v", payout)
	}

	if err := store.SetPartnerStatus(ctx, kol.UserID, PartnerFrozen, "admin", "freeze"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PartnerForPayout(ctx, kol.UserID, RoleKOL); err == nil {
		t.Fatal("frozen partner can request payout")
	}
}
