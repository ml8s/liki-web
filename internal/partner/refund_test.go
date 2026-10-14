package partner

import (
	"context"
	"errors"
	"testing"

	"github.com/ml8s/liki-web/internal/payment"
)

func newPendingCommissionForRefund(t *testing.T) (*Store, Commission) {
	t.Helper()
	store := newStore(t)
	kol, err := store.createKOLPartner(context.Background(), "kol")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Attribute(context.Background(), "buyer", kol.Code); err != nil {
		t.Fatal(err)
	}
	sale := paidCommission(payment.ProviderXunhu, "buyer", 10000)
	if _, _, err := store.EnqueueCommissionPayment(context.Background(), sale); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionPayments(context.Background(), 10); err != nil || n != 1 {
		t.Fatalf("processed=%d err=%v", n, err)
	}
	commission, err := store.commission(context.Background(), string(payment.ProviderXunhu), sale.ID)
	if err != nil {
		t.Fatal(err)
	}
	return store, commission
}

func TestPartialRefundAdjustsCommissionLedger(t *testing.T) {
	ctx := context.Background()
	store, commission := newPendingCommissionForRefund(t)
	sale := paidCommission(payment.ProviderXunhu, "buyer", 10000)
	event := payment.WebhookEvent{
		Provider: payment.ProviderXunhu, Type: payment.EventPaymentRefunded,
		Data: payment.WebhookEventData{OrderID: sale.ID, PaymentID: "refund-1", Amount: 4000, Currency: "CNY"},
	}
	if err := store.EnqueueCommissionRefund(ctx, event, sale); err != nil {
		t.Fatal(err)
	}
	// A duplicate webhook fact must be ignored by the durable intake.
	if err := store.EnqueueCommissionRefund(ctx, event, sale); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionRefunds(ctx, 10); err != nil || n != 1 {
		t.Fatalf("processed=%d err=%v", n, err)
	}
	var refunded int64
	var status string
	if err := store.db.QueryRowContext(ctx, `
SELECT refunded_minor,status FROM referral_commission_entries WHERE id=?`, commission.ID+":kol").Scan(&refunded, &status); err != nil {
		t.Fatal(err)
	}
	if refunded != 2000 || status != string(CommissionPending) {
		t.Fatalf("entry refunded/status = %d/%q", refunded, status)
	}
	if commission, err := store.commission(ctx, string(payment.ProviderXunhu), sale.ID); err != nil || commission.RefundedMinor != 4000 {
		t.Fatalf("sale refunded=%d err=%v", commission.RefundedMinor, err)
	}
}

func TestFullRefundVoidsCommissionEntry(t *testing.T) {
	ctx := context.Background()
	store, commission := newPendingCommissionForRefund(t)
	sale := paidCommission(payment.ProviderXunhu, "buyer", 10000)
	event := payment.WebhookEvent{
		Provider: payment.ProviderXunhu, Type: payment.EventPaymentRefunded,
		Data: payment.WebhookEventData{OrderID: sale.ID, PaymentID: "refund-full", Amount: 10000, Currency: "CNY"},
	}
	if err := store.EnqueueCommissionRefund(ctx, event, sale); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionRefunds(ctx, 10); err != nil || n != 1 {
		t.Fatalf("processed=%d err=%v", n, err)
	}
	var status string
	if err := store.db.QueryRowContext(ctx, `
SELECT status FROM referral_commission_entries WHERE id=?`, commission.ID+":kol").Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != string(CommissionVoided) {
		t.Fatalf("status=%q want voided", status)
	}
	if _, err := store.CreatePayout(ctx, commission.ReferrerID, PayoutMethodAlipay, "me@example.com"); !errors.Is(err, ErrNoWithdrawableCommission) {
		t.Fatalf("payout err=%v want no withdrawable commission", err)
	}
}

func TestRefundDoesNotChangePaidLedgerAutomatically(t *testing.T) {
	ctx := context.Background()
	store, commission := newPendingCommissionForRefund(t)
	payout, err := store.CreatePayout(ctx, commission.ReferrerID, PayoutMethodAlipay, "me@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ApprovePayout(ctx, payout.ID, "admin"); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkPayoutPaid(ctx, payout.ID, "admin", "transfer-paid"); err != nil {
		t.Fatal(err)
	}
	sale := paidCommission(payment.ProviderXunhu, "buyer", 10000)
	event := payment.WebhookEvent{
		Provider: payment.ProviderXunhu, Type: payment.EventPaymentRefunded,
		Data: payment.WebhookEventData{OrderID: sale.ID, PaymentID: "refund-after-paid", Amount: 10000, Currency: "CNY"},
	}
	if err := store.EnqueueCommissionRefund(ctx, event, sale); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionRefunds(ctx, 10); err != nil || n != 1 {
		t.Fatalf("processed=%d err=%v", n, err)
	}
	var status string
	var refunded int64
	if err := store.db.QueryRowContext(ctx, `
SELECT status,refunded_minor FROM referral_commission_entries WHERE id=?`,
		commission.ID+":kol").Scan(&status, &refunded); err != nil {
		t.Fatal(err)
	}
	if status != string(CommissionPaid) || refunded != 5000 {
		t.Fatalf("paid entry status/refunded=%q/%d", status, refunded)
	}
	tasks, err := store.ListFinanceTasks(ctx, FinanceTaskPending, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].EntryID != commission.ID+":kol" ||
		tasks[0].AmountMinor != 5000 || tasks[0].ProviderRefundID != "refund-after-paid" {
		t.Fatalf("finance tasks=%+v", tasks)
	}
	if err := store.ResolveFinanceTask(ctx, tasks[0].ID, "admin", string(FinanceTaskResolved), "recovered by operations"); err != nil {
		t.Fatal(err)
	}
	if err := store.ResolveFinanceTask(ctx, tasks[0].ID, "admin", string(FinanceTaskResolved), "duplicate"); err != ErrFinanceTaskInvalidTransition {
		t.Fatalf("duplicate resolve err=%v", err)
	}
}

func TestRefundRejectsLockedPayoutButPreservesRequestedAmount(t *testing.T) {
	ctx := context.Background()
	store, commission := newPendingCommissionForRefund(t)
	payout, err := store.CreatePayout(ctx, commission.ReferrerID, PayoutMethodAlipay, "me@example.com")
	if err != nil {
		t.Fatal(err)
	}
	sale := paidCommission(payment.ProviderXunhu, "buyer", 10000)
	event := payment.WebhookEvent{
		Provider: payment.ProviderXunhu, Type: payment.EventPaymentRefunded,
		Data: payment.WebhookEventData{OrderID: sale.ID, PaymentID: "refund-locked", Amount: 10000, Currency: "CNY"},
	}
	if err := store.EnqueueCommissionRefund(ctx, event, sale); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionRefunds(ctx, 10); err != nil || n != 1 {
		t.Fatalf("processed=%d err=%v", n, err)
	}
	got, err := store.PayoutByID(ctx, payout.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != PayoutRejected || got.RequestedAmountMinor != 5000 || got.AmountMinor != 5000 {
		t.Fatalf("payout=%+v", got)
	}
	var before, after int64
	if err := store.db.QueryRowContext(ctx, `
SELECT before_minor,after_minor FROM referral_payout_adjustments WHERE payout_id=?`, payout.ID).Scan(&before, &after); err != nil {
		t.Fatal(err)
	}
	if before != 5000 || after != 0 {
		t.Fatalf("adjustment=%d/%d", before, after)
	}
	var allocations int64
	if err := store.db.QueryRowContext(ctx, `
SELECT COUNT(1) FROM commission_entry_refunds WHERE provider_refund_id='refund-locked'`).Scan(&allocations); err != nil {
		t.Fatal(err)
	}
	if allocations != 1 {
		t.Fatalf("refund allocations=%d", allocations)
	}
}

func TestRecoverCommissionRefundsResumesRunning(t *testing.T) {
	ctx := context.Background()
	store, commission := newPendingCommissionForRefund(t)
	sale := paidCommission(payment.ProviderXunhu, "buyer", 10000)
	event := payment.WebhookEvent{
		Provider: payment.ProviderXunhu, Type: payment.EventPaymentRefunded,
		Data: payment.WebhookEventData{OrderID: sale.ID, PaymentID: "refund-1", Amount: 4000, Currency: "CNY"},
	}
	if err := store.EnqueueCommissionRefund(ctx, event, sale); err != nil {
		t.Fatal(err)
	}
	// 模拟进程在 claim 后、apply 前崩溃：intake 卡在 running
	if _, err := store.db.ExecContext(ctx, `UPDATE commission_refund_intents SET status='running'`); err != nil {
		t.Fatal(err)
	}
	if err := store.recoverCommissionRefunds(ctx); err != nil {
		t.Fatal(err)
	}
	// 恢复为 queued 后，worker 必须能继续处理（不再永久卡死）
	if n, err := store.ProcessCommissionRefunds(ctx, 10); err != nil || n != 1 {
		t.Fatalf("processed=%d err=%v", n, err)
	}
	_ = commission
}
