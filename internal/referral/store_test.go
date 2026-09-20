package referral

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"liki-web/internal/payment"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "referral.db"), func(context.Context, string) (bool, error) {
		return true, nil
	})
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

func TestRegisterRequiresPaidAccess(t *testing.T) {
	store := newStore(t)
	store.paidCheck = func(context.Context, string) (bool, error) { return false, nil }
	if _, err := store.RegisterReferrer(context.Background(), "u1", ""); err != ErrNotEligible {
		t.Fatalf("err = %v, want ErrNotEligible", err)
	}
}

func TestRegisterRecruitAndAttribute(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	kol, err := store.RegisterReferrer(ctx, "u1", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(kol.Code) != 10 || kol.Status != PartnerActive {
		t.Fatalf("kol = %+v", kol)
	}

	again, err := store.RegisterReferrer(ctx, "u1", "")
	if err != nil {
		t.Fatal(err)
	}
	if again.Code != kol.Code {
		t.Fatalf("registration was not idempotent: %+v != %+v", again, kol)
	}

	if _, err := store.RegisterReferrer(ctx, "u2", kol.Code); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Attribute(ctx, "u1", kol.Code); err != ErrSelfReferral {
		t.Fatalf("self attribution err = %v", err)
	}
	attribution, err := store.Attribute(ctx, "u3", strings.ToLower(kol.Code))
	if err != nil {
		t.Fatal(err)
	}
	if attribution.ReferrerID != kol.UserID || attribution.Code != kol.Code {
		t.Fatalf("attribution = %+v", attribution)
	}
	repeated, err := store.Attribute(ctx, "u3", kol.Code)
	if err != nil || repeated != attribution {
		t.Fatalf("attribute was not idempotent: %+v, %v", repeated, err)
	}
}
