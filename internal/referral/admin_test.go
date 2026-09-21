package referral

import (
	"context"
	"path/filepath"
	"testing"
)

func newAdminStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "referral.db"), []byte("test-referral-payout-secret"), func(context.Context, string) (bool, error) {
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

func TestScoutApplicationLifecycle(t *testing.T) {
	ctx := context.Background()
	store := newAdminStore(t)

	if _, err := store.SubmitScoutApplication(ctx, "missing", "test"); err == nil {
		t.Fatal("expected error for non-referrer")
	}

	kol, err := store.RegisterReferrer(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	app, err := store.SubmitScoutApplication(ctx, kol.UserID, "creator network")
	if err != nil {
		t.Fatal(err)
	}
	if app.Status != ApplicationPending {
		t.Fatalf("status = %q, want pending", app.Status)
	}
	if _, err := store.SubmitScoutApplication(ctx, kol.UserID, "duplicate"); err != ErrApplicationPending {
		t.Fatalf("duplicate err = %v, want ErrApplicationPending", err)
	}

	apps, err := store.ListApplications(ctx, ApplicationPending, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].ID != app.ID {
		t.Fatalf("applications = %+v", apps)
	}

	if err := store.DecideApplication(ctx, app.ID, "admin", true, "approved"); err != nil {
		t.Fatal(err)
	}
	partner, err := store.ReferrerByUser(ctx, kol.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if partner.Role != RoleScout {
		t.Fatalf("role = %q, want scout", partner.Role)
	}
	if err := store.DecideApplication(ctx, app.ID, "admin", false, "again"); err != ErrApplicationNotPending {
		t.Fatalf("second decision err = %v", err)
	}
}

func TestPartnerStatusAndList(t *testing.T) {
	ctx := context.Background()
	store := newAdminStore(t)
	kol, err := store.RegisterReferrer(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetPartnerStatus(ctx, kol.UserID, PartnerFrozen, "admin", "manual review"); err != nil {
		t.Fatal(err)
	}
	partners, err := store.ListPartners(ctx, "", PartnerFrozen, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(partners) != 1 || partners[0].Status != PartnerFrozen {
		t.Fatalf("partners = %+v", partners)
	}
	if err := store.SetPartnerStatus(ctx, kol.UserID, PartnerActive, "admin", "cleared"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ListPartners(ctx, "", PartnerFrozen, "", 20); err != nil {
		t.Fatal(err)
	}
}

func TestScoutIsNotTreatedAsKOLStatus(t *testing.T) {
	ctx := context.Background()
	store := newAdminStore(t)
	kol, err := store.RegisterReferrer(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PromoteToScout(ctx, kol.UserID, "admin", "approved"); err != nil {
		t.Fatal(err)
	}

	status, err := store.KOLStatus(ctx, kol.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Registered || status.Eligible {
		t.Fatalf("scout unexpectedly treated as KOL: %+v", status)
	}
}

func TestScoutDoesNotReceiveKOLStatus(t *testing.T) {
	ctx := context.Background()
	store := newAdminStore(t)
	kol, err := store.RegisterReferrer(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PromoteToScout(ctx, kol.UserID, "admin", "approved"); err != nil {
		t.Fatal(err)
	}

	status, err := store.KOLStatus(ctx, kol.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Registered || status.Eligible || status.Status != "" {
		t.Fatalf("scout received KOL status: %+v", status)
	}
}
