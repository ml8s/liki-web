package referral

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"liki-web/internal/payment"
)

func TestCommissionIntakeSkipsIneligiblePayments(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	intent := paidIntent("stripe", "buyer", 8800)
	intent.ID = "pi-stripe"
	status, reason, err := store.EnqueueCommissionIntent(ctx, intent)
	if err != nil || status != IntakeSkipped || reason != "unsupported_provider" {
		t.Fatalf("status=%q reason=%q err=%v", status, reason, err)
	}
}

func TestCommissionWorkerSettlesDodoAndXunhuExactlyOnce(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	kol, err := store.RegisterReferrer(ctx, "kol", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RegisterReferrer(ctx, "recruiter", ""); err != nil {
		t.Fatal(err)
	}
	recruiter, err := store.ReferrerByUser(ctx, "recruiter")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE referrers SET recruiter_id=? WHERE user_id=?`, recruiter.UserID, "kol"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Attribute(ctx, "buyer", kol.Code); err != nil {
		t.Fatal(err)
	}

	xunhu := paidIntent(payment.ProviderXunhu, "buyer", 8800)
	xunhu.ID = "pi-xunhu"
	if status, reason, err := store.EnqueueCommissionIntent(ctx, xunhu); err != nil || status != IntakeQueued || reason != "" {
		t.Fatalf("xunhu intake=%q/%q err=%v", status, reason, err)
	}
	dodo := paidIntent(payment.ProviderDodo, "buyer", 1000)
	dodo.ID = "pi-dodo"
	dodo.Currency = "USD"
	if _, _, err := store.EnqueueCommissionIntent(ctx, dodo); err != nil {
		t.Fatal(err)
	}

	if n, err := store.ProcessCommissionIntents(ctx, 10); err != nil || n != 2 {
		t.Fatalf("processed=%d err=%v", n, err)
	}
	xcom, err := store.commission(ctx, string(payment.ProviderXunhu), xunhu.ID)
	if err != nil {
		t.Fatal(err)
	}
	if xcom.ReferrerAmount != 4400 || xcom.RecruiterAmount != 1760 {
		t.Fatalf("xunhu commission=%+v", xcom)
	}
	if n, err := store.ProcessCommissionIntents(ctx, 10); err != nil || n != 0 {
		t.Fatalf("second processed=%d err=%v", n, err)
	}
	stats, err := store.PartnerStats(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalEarnedMinor != 4900 || len(stats.Commissions) != 2 {
		t.Fatalf("stats=%+v", stats)
	}
	if !strings.Contains(xcom.PaymentProvider, "xunhu") {
		t.Fatalf("provider=%q", xcom.PaymentProvider)
	}
}

func TestReconcileSuccessfulPaymentsCompensatesAttributionRace(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	kol, err := store.RegisterReferrer(ctx, "kol", "")
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
	if n, err := store.ProcessCommissionIntents(ctx, 10); err != nil || n != 1 {
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
