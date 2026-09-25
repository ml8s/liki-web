package referral

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ml8s/liki-web/internal/payment"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "referral.db"), []byte("test-referral-payout-secret"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store
}

func paidIntent(provider payment.Provider, buyer string, amount int64) payment.PaymentIntent {
	return payment.PaymentIntent{
		ID:              "pi-" + string(provider) + "-" + buyer,
		BusinessOrderID: "order-" + buyer, Product: "liki-chat", UserID: buyer,
		Provider: provider, Status: payment.IntentPaid, AmountMinor: amount,
		Currency: "CNY", ProviderPaymentID: "pay-" + buyer,
	}
}

func TestCreateKOLPartnerRecruitAndAttribute(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	scout, err := store.CreateScoutPartner(ctx, "scout", "admin", "seed scout")
	if err != nil {
		t.Fatal(err)
	}
	if len(scout.Code) != 10 || scout.Status != PartnerActive {
		t.Fatalf("scout = %+v", scout)
	}

	again, err := store.CreateScoutPartner(ctx, "scout", "admin", "seed scout again")
	if err != nil {
		t.Fatal(err)
	}
	if again.Code != scout.Code {
		t.Fatalf("registration was not idempotent: %+v != %+v", again, scout)
	}

	if _, err := store.Attribute(ctx, "kol", scout.Code); err != nil {
		t.Fatal(err)
	}
	partner, err := store.createKOLPartner(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	if partner.RecruiterID != scout.UserID {
		t.Fatalf("recruiter = %q, want %q", partner.RecruiterID, scout.UserID)
	}

	if _, err := store.Attribute(ctx, "scout", scout.Code); err != ErrSelfReferral {
		t.Fatalf("self attribution err = %v", err)
	}
	attribution, err := store.Attribute(ctx, "buyer", strings.ToLower(scout.Code))
	if err != nil {
		t.Fatal(err)
	}
	if attribution.ReferrerID != scout.UserID || attribution.Code != scout.Code {
		t.Fatalf("attribution = %+v", attribution)
	}
	repeated, err := store.Attribute(ctx, "buyer", scout.Code)
	if err != nil || repeated != attribution {
		t.Fatalf("attribute was not idempotent: %+v, %v", repeated, err)
	}
}
