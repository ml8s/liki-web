package referral

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/payment"
)

func TestCommissionIntakeSkipsIneligiblePayments(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	sale := paidCommission("stripe", "buyer", 8800)
	sale.ID = "pi-stripe"
	status, reason, err := store.EnqueueCommissionPayment(ctx, sale)
	if err != nil || status != IntakeSkipped || reason != "unsupported_provider" {
		t.Fatalf("status=%q reason=%q err=%v", status, reason, err)
	}
}

func TestCommissionWorkerSettlesDodoAndXunhuExactlyOnce(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	recruiter, err := store.CreateScoutPartner(ctx, "recruiter", "admin", "standalone scout")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Attribute(ctx, "kol", recruiter.Code); err != nil {
		t.Fatal(err)
	}
	kol, err := store.createKOLPartner(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	if kol.RecruiterID != recruiter.UserID {
		t.Fatalf("recruiter = %q, want %q", kol.RecruiterID, recruiter.UserID)
	}
	if _, err := store.Attribute(ctx, "buyer", kol.Code); err != nil {
		t.Fatal(err)
	}

	xunhu := paidCommission(payment.ProviderXunhu, "buyer", 8800)
	xunhu.ID = "pi-xunhu"
	if status, reason, err := store.EnqueueCommissionPayment(ctx, xunhu); err != nil || status != IntakeQueued || reason != "" {
		t.Fatalf("xunhu intake=%q/%q err=%v", status, reason, err)
	}
	dodo := paidCommission(payment.ProviderDodo, "buyer", 1000)
	dodo.ID = "pi-dodo"
	dodo.Currency = "USD"
	if _, _, err := store.EnqueueCommissionPayment(ctx, dodo); err != nil {
		t.Fatal(err)
	}

	if n, err := store.ProcessCommissionPayments(ctx, 10); err != nil || n != 2 {
		t.Fatalf("processed=%d err=%v", n, err)
	}
	xcom, err := store.commission(ctx, string(payment.ProviderXunhu), xunhu.ID)
	if err != nil {
		t.Fatal(err)
	}
	if xcom.ReferrerAmount != 4400 || xcom.RecruiterAmount != 1760 {
		t.Fatalf("xunhu commission=%+v", xcom)
	}
	if n, err := store.ProcessCommissionPayments(ctx, 10); err != nil || n != 0 {
		t.Fatalf("second processed=%d err=%v", n, err)
	}
	stats, err := store.PartnerStats(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Balances) != 2 || len(stats.Commissions) != 2 {
		t.Fatalf("stats=%+v", stats)
	}
	cny, usd := stats.Balances[0], stats.Balances[1]
	if cny.Currency != "CNY" || cny.TotalMinor != 4400 || usd.Currency != "USD" || usd.TotalMinor != 500 {
		t.Fatalf("currency balances=%+v", stats.Balances)
	}
	if !strings.Contains(xcom.PaymentProvider, "xunhu") {
		t.Fatalf("provider=%q", xcom.PaymentProvider)
	}
}

func TestReconcileSuccessfulPaymentsCompensatesAttributionRace(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	kol, err := store.createKOLPartner(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}

	platformDB, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "platform.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = platformDB.Close() })
	if _, err := platformDB.Exec(`
CREATE TABLE payment_intents (
 id TEXT PRIMARY KEY,business_order_id TEXT,user_id TEXT,product TEXT,
 provider TEXT,status TEXT,amount_minor INTEGER,currency TEXT,updated_at TEXT
);
CREATE TABLE fulfillment_tasks (payment_intent_id TEXT,status TEXT);`); err != nil {
		t.Fatal(err)
	}
	if _, err := platformDB.Exec(`
INSERT INTO payment_intents VALUES ('pi-reconcile','order-r','buyer','liki-chat','xunhu','paid',8800,'CNY','2026-01-01T00:00:00Z');
INSERT INTO fulfillment_tasks VALUES ('pi-reconcile','succeeded');`); err != nil {
		t.Fatal(err)
	}

	n, err := store.ReconcileSuccessfulPayments(ctx, platformDB, 10)
	if err != nil || n != 1 {
		t.Fatalf("reconcile=%d err=%v", n, err)
	}
	var status, reason string
	if err := store.db.QueryRow(`SELECT status,skip_reason FROM commission_intents WHERE payment_intent_id='pi-reconcile'`).Scan(&status, &reason); err != nil {
		t.Fatal(err)
	}
	if status != string(IntakeSkipped) || reason != "no_attribution" {
		t.Fatalf("status=%q reason=%q", status, reason)
	}

	if _, err := store.Attribute(ctx, "buyer", kol.Code); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReconcileSuccessfulPayments(ctx, platformDB, 10); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionPayments(ctx, 10); err != nil || n != 1 {
		t.Fatalf("processed=%d err=%v", n, err)
	}
	commission, err := store.commission(ctx, "xunhu", "pi-reconcile")
	if err != nil {
		t.Fatal(err)
	}
	if commission.ReferrerAmount != 4400 {
		t.Fatalf("commission=%+v", commission)
	}
}

func TestVoidCommissionForPayment(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	kol, err := store.createKOLPartner(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Attribute(ctx, "buyer", kol.Code); err != nil {
		t.Fatal(err)
	}
	sale := paidCommission(payment.ProviderXunhu, "buyer", 8800)
	sale.ID = "pi-refund"
	if _, _, err := store.EnqueueCommissionPayment(ctx, sale); err != nil {
		t.Fatal(err)
	}
	if n, err := store.ProcessCommissionPayments(ctx, 10); err != nil || n != 1 {
		t.Fatalf("processed=%d err=%v", n, err)
	}
	if err := store.VoidCommissionForPayment(ctx, payment.ProviderXunhu, sale.ID, "payment.refunded"); err != nil {
		t.Fatal(err)
	}
	got, err := store.commission(ctx, string(payment.ProviderXunhu), sale.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != CommissionVoided {
		t.Fatalf("status = %q, want voided", got.Status)
	}
	if err := store.VoidCommissionForPayment(ctx, payment.ProviderXunhu, sale.ID, "payment.refunded"); err != nil {
		t.Fatalf("repeat void should be idempotent: %v", err)
	}
}
