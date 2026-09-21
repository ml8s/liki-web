package referral

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"liki-web/internal/migrate"
	"liki-web/internal/sqliteutil"
)

func TestFinancialMigrationsPreserveLegacyPayoutsAndRestoreRole(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "referral.db")
	db, err := sqliteutil.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := Open(dbPath, []byte("test-referral-payout-secret"), func(context.Context, string) (bool, error) { return true, nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	migrations := store.migrations()
	if err := migrate.Run(ctx, db, migrations[:5]); err != nil {
		t.Fatalf("migrate legacy schema: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	legacy := []string{
		`INSERT INTO referrers (user_id,code,role,status,created_at,updated_at) VALUES ('kol','KOLCODE','kol','active','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`,
		`INSERT INTO referrers (user_id,code,role,status,created_at,updated_at) VALUES ('scout','SCOUTCODE','scout','active','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`,
		`INSERT INTO referral_commissions
		 (id,payment_provider,payment_intent_id,business_order_id,product,buyer_user_id,referrer_id,recruiter_id,
		  amount_minor,currency,referrer_rate_bps,referrer_commission_minor,recruiter_rate_bps,recruiter_commission_minor,
		  status,payout_id,created_at,updated_at)
		 VALUES ('c1','xunhu','pi-1','o1','liki-chat','buyer','scout',NULL,10000,'CNY',5000,5000,2000,2000,
		  'pending','p-kol','2026-01-02T00:00:00Z','2026-01-02T00:00:00Z')`,
		`INSERT INTO referral_payouts
		 (id,referrer_id,amount_minor,currency,method,account,status,created_at,updated_at)
		 VALUES ('p-kol','scout',5000,'CNY','alipay','payout@example.com','requested','2026-01-03T00:00:00Z','2026-01-03T00:00:00Z')`,
		`INSERT INTO referral_payouts
		 (id,referrer_id,amount_minor,currency,method,account,status,created_at,updated_at)
		 VALUES ('p-scout','scout',10000,'CNY','paypal','payer@example.com','approved','2026-01-03T00:00:00Z','2026-01-03T00:00:00Z')`,
	}
	for _, query := range legacy {
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Fatalf("seed legacy data: %v", err)
		}
	}

	// Run 006 against the legacy rows, then attach the scout entry to its legacy
	// payout so role inference is exercised.
	if err := migrate.Run(ctx, db, migrations[5:6]); err != nil {
		t.Fatalf("migrate 006: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE referral_commission_entries SET payout_id='p-scout' WHERE beneficiary_role='scout'`); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Run(ctx, db, migrations[6:7]); err != nil {
		t.Fatalf("migrate 007: %v", err)
	}
	if err := migrate.Run(ctx, db, migrations[7:8]); err != nil {
		t.Fatalf("migrate 008: %v", err)
	}
	if err := migrate.Run(ctx, db, migrations[8:]); err != nil {
		t.Fatalf("migrate financial migrations: %v", err)
	}

	var role string
	if err := db.QueryRowContext(ctx, `SELECT status FROM partner_roles WHERE user_id='scout' AND role='kol'`).Scan(&role); err != nil {
		t.Fatalf("historical KOL role was not restored: %v", err)
	}
	if role != "retired" {
		t.Fatalf("historical KOL role = %q", role)
	}
	if _, err := store.KOLByID(ctx, "scout"); err == nil {
		t.Fatal("retired historical KOL capability must not access KOL console")
	}

	var requested int64
	var encryptedAccount string
	if err := db.QueryRowContext(ctx, `SELECT requested_amount_minor,account FROM referral_payouts WHERE id='p-kol'`).Scan(&requested, &encryptedAccount); err != nil {
		t.Fatal(err)
	}
	if requested != 5000 || !strings.HasPrefix(encryptedAccount, "v1:") {
		t.Fatalf("requested=%d account_encrypted=%t", requested, strings.HasPrefix(encryptedAccount, "v1:"))
	}
	payout, err := store.PayoutByID(ctx, "p-kol")
	if err != nil {
		t.Fatal(err)
	}
	if payout.Role != RoleKOL || payout.RequestedAmountMinor != 5000 || payout.Account != "payout@example.com" {
		t.Fatalf("payout=%+v", payout)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO commission_refund_intents
(id,payment_provider,payment_intent_id,provider_refund_id,amount_minor,currency,status,attempts,run_at,created_at,updated_at)
VALUES ('failed-test','xunhu','pi-1','refund-failed',1000,'CNY','failed',1,?, ?, ?)`, now, now, now); err != nil {
		t.Fatalf("failed status rejected: %v", err)
	}
}
