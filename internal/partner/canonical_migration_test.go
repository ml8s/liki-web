package partner

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ml8s/liki-web/internal/migrate"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestCanonicalPartnerMigrationRenamesLegacyTables(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "referral.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	legacyTables := []string{
		"referrers",
		"referral_attributions",
		"referral_applications",
		"referral_admin_actions",
		"referral_commissions",
		"referral_commission_entries",
		"commission_intents",
		"commission_refund_intents",
		"commission_entry_refunds",
		"referral_payouts",
		"referral_payout_adjustments",
		"finance_tasks",
	}
	for _, table := range legacyTables {
		column := ""
		if table == "referral_payouts" {
			column = ",referrer_id TEXT NOT NULL DEFAULT ''"
		}
		_, err := db.ExecContext(ctx, "CREATE TABLE "+table+" (id TEXT PRIMARY KEY"+column+")")
		if err != nil {
			t.Fatal(err)
		}
	}

	store, err := Open(filepath.Join(t.TempDir(), "runner.db"), []byte("test-referral-payout-secret"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	migrations := store.migrations()
	if err := migrate.Run(ctx, db, migrations[12:]); err != nil {
		t.Fatal(err)
	}

	for _, oldTable := range legacyTables {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, oldTable).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("legacy table %s still exists", oldTable)
		}
	}
	for _, table := range legacyTables {
		expectedNames := map[string]string{
			"referrers":                   "partners",
			"referral_attributions":       "partner_attributions",
			"referral_applications":       "partner_applications",
			"referral_admin_actions":      "partner_admin_actions",
			"referral_commissions":        "partner_commissions",
			"referral_commission_entries": "partner_commission_entries",
			"commission_intents":          "partner_commission_intake",
			"commission_refund_intents":   "partner_commission_refund_intake",
			"commission_entry_refunds":    "partner_commission_entry_refunds",
			"referral_payouts":            "partner_payouts",
			"referral_payout_adjustments": "partner_payout_adjustments",
			"finance_tasks":               "partner_finance_tasks",
		}
		table = expectedNames[table]
		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("canonical table %s missing", table)
		}
	}
}
