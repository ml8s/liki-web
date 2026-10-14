package partner

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func newAdminStore(t *testing.T) *Store {
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

func TestScoutApplicationLifecycle(t *testing.T) {
	ctx := context.Background()
	store := newAdminStore(t)

	app, err := store.SubmitApplication(ctx, RoleScout, "startup-blogger", "creator network")
	if err != nil {
		t.Fatal(err)
	}
	if app.Status != ApplicationPending {
		t.Fatalf("status = %q, want pending", app.Status)
	}
	if _, err := store.SubmitApplication(ctx, RoleScout, "startup-blogger", "duplicate"); err != ErrApplicationPending {
		t.Fatalf("duplicate err = %v, want ErrApplicationPending", err)
	}

	apps, err := store.ListApplications(ctx, ApplicationPending, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].ID != app.ID {
		t.Fatalf("applications = %+v", apps)
	}

	if err := store.DecideApplication(ctx, app.ID, "admin", true, "approved"); err != nil {
		t.Fatal(err)
	}
	partner, err := store.ReferrerByUser(ctx, "startup-blogger")
	if err != nil {
		t.Fatal(err)
	}
	if partner.Role != RoleScout {
		t.Fatalf("role = %q, want scout", partner.Role)
	}
	overview, err := store.ScoutOverview(ctx, partner.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if overview.Partner.Code != partner.Code || overview.RecruitCount != 0 {
		t.Fatalf("overview = %+v, want partner code %q and no recruits", overview, partner.Code)
	}
	if err := store.DecideApplication(ctx, app.ID, "admin", false, "again"); err != ErrApplicationNotPending {
		t.Fatalf("second decision err = %v", err)
	}
}

func TestScoutApplicationDoesNotUpgradeKOL(t *testing.T) {
	ctx := context.Background()
	store := newAdminStore(t)
	kol, err := store.createKOLPartner(ctx, "kol")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.SubmitApplication(ctx, RoleScout, kol.UserID, "invalid"); !errors.Is(err, ErrRoleConflict) {
		t.Fatalf("submit err = %v, want ErrRoleConflict", err)
	}
	if err := store.DecideApplication(ctx, "missing-application", "admin", true, "approved"); err != ErrApplicationNotFound {
		t.Fatal(err)
	}
	if partner, err := store.ReferrerByUser(ctx, kol.UserID); err != nil {
		t.Fatal(err)
	} else if partner.Role != RoleKOL {
		t.Fatalf("role = %q, want kol", partner.Role)
	}
}

func TestPartnerStatusAndList(t *testing.T) {
	ctx := context.Background()
	store := newAdminStore(t)
	kol, err := store.createKOLPartner(ctx, "kol")
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

func TestPartnerStatesKeepRolesSeparate(t *testing.T) {
	ctx := context.Background()
	store := newAdminStore(t)
	scout, err := store.CreateScoutPartner(ctx, "scout", "admin", "standalone scout")
	if err != nil {
		t.Fatal(err)
	}

	states, err := store.PartnerStates(ctx, scout.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if states.Scout.Status != RoleActive || states.KOL.Status != "" || states.KOL.CanSubmit {
		t.Fatalf("unexpected partner states: %+v", states)
	}
}

func TestKOLApplicationApprovalPreservesScoutAttribution(t *testing.T) {
	ctx := context.Background()
	store := newAdminStore(t)
	scout, err := store.CreateScoutPartner(ctx, "scout", "admin", "seed scout")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Attribute(ctx, "applicant", scout.Code); err != nil {
		t.Fatal(err)
	}

	// There is deliberately no paid-access precondition: review is the gate.
	app, err := store.SubmitApplication(ctx, RoleKOL, "applicant", "content creator")
	if err != nil {
		t.Fatal(err)
	}
	if app.Type != RoleKOL {
		t.Fatalf("application type = %q, want kol", app.Type)
	}
	if err := store.DecideApplication(ctx, app.ID, "admin", true, "approved"); err != nil {
		t.Fatal(err)
	}
	partner, err := store.ReferrerByUser(ctx, "applicant")
	if err != nil {
		t.Fatal(err)
	}
	if partner.Role != RoleKOL || partner.RecruiterID != scout.UserID {
		t.Fatalf("partner = %+v, want scout-attributed KOL", partner)
	}
}
