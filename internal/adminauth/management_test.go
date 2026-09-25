package adminauth

import (
	"context"
	"errors"
	"testing"

	"github.com/ml8s/liki-web/internal/auth"
)

func TestAdminManagementLifecycle(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	if err := store.BootstrapAdminByEmail(ctx, "owner@example.com"); err != nil {
		t.Fatal(err)
	}

	if err := store.EnsureAdmin(ctx, "owner@example.com", "ops@example.com", "new operations reviewer"); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureAdmin(ctx, "owner@example.com", "ops@example.com", "duplicate"); !errors.Is(err, ErrAdminAlreadyExists) {
		t.Fatalf("duplicate err=%v", err)
	}

	if err := store.GrantCapability(ctx, "owner@example.com", "ops@example.com", CapabilityPayoutRead, "finance review"); err != nil {
		t.Fatal(err)
	}
	ok, err := store.Authorize(ctx, adminUser("ops@example.com"), CapabilityPayoutRead)
	if err != nil || !ok {
		t.Fatalf("authorized=%v err=%v", ok, err)
	}

	admins, err := store.ListAdmins(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(admins) != 2 {
		t.Fatalf("admins=%+v", admins)
	}
	var ops AdminAccount
	for _, item := range admins {
		if item.Email == "ops@example.com" {
			ops = item
		}
	}
	if len(ops.Capabilities) != 1 || ops.Capabilities[0].Capability != CapabilityPayoutRead {
		t.Fatalf("ops=%+v", ops)
	}

	if err := store.RevokeCapability(ctx, "owner@example.com", "ops@example.com", CapabilityPayoutRead, "rotation"); err != nil {
		t.Fatal(err)
	}
	ok, err = store.Authorize(ctx, adminUser("ops@example.com"), CapabilityPayoutRead)
	if err != nil || ok {
		t.Fatalf("after revoke authorized=%v err=%v", ok, err)
	}
}

func TestAdminManagementSafetyRules(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	if err := store.BootstrapAdminByEmail(ctx, "owner@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureAdmin(ctx, "owner@example.com", "ops@example.com", "second reviewer"); err != nil {
		t.Fatal(err)
	}

	if err := store.RevokeCapability(ctx, "owner@example.com", "owner@example.com", CapabilityAdminSuper, "self service"); !errors.Is(err, ErrSelfManagementForbidden) {
		t.Fatalf("self revoke err=%v", err)
	}
	if err := store.RevokeCapability(ctx, "owner@example.com", "ops@example.com", Capability("made.up"), "reason"); !errors.Is(err, ErrInvalidCapability) {
		t.Fatalf("invalid capability err=%v", err)
	}
	if err := store.SetAdminStatus(ctx, "owner@example.com", "owner@example.com", AdminDisabled, "self service"); !errors.Is(err, ErrSelfManagementForbidden) {
		t.Fatalf("self disable err=%v", err)
	}
	if err := store.SetAdminStatus(ctx, "owner@example.com", "owner@example.com", AdminDisabled, ""); !errors.Is(err, ErrSelfManagementForbidden) {
		t.Fatalf("validation order err=%v", err)
	}

	// Removing the only active super capability would lock out authorization.
	if err := store.EnsureAdmin(ctx, "owner@example.com", "second@example.com", "no capabilities"); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeCapability(ctx, "second@example.com", "owner@example.com", CapabilityAdminSuper, "rotation without successor"); !errors.Is(err, ErrLastSuperAdmin) {
		t.Fatalf("last super revoke err=%v", err)
	}
	if err := store.SetAdminStatus(ctx, "second@example.com", "owner@example.com", AdminDisabled, "disable without successor"); !errors.Is(err, ErrLastSuperAdmin) {
		t.Fatalf("last super disable err=%v", err)
	}

	// A second active super makes rotation possible.
	if err := store.GrantCapability(ctx, "owner@example.com", "second@example.com", CapabilityAdminSuper, "successor owner"); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeCapability(ctx, "second@example.com", "owner@example.com", CapabilityAdminSuper, "planned rotation"); err != nil {
		t.Fatalf("rotation revoke err=%v", err)
	}
	ok, err := store.Authorize(ctx, auth.User{ID: "user-owner@example.com", Email: "owner@example.com", Status: auth.UserActive}, CapabilityAdminSuper)
	if err != nil || ok {
		t.Fatalf("revoked super authorized=%v err=%v", ok, err)
	}
}

func TestAdminManagementWritesAudit(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	if err := store.BootstrapAdminByEmail(ctx, "owner@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureAdmin(ctx, "owner@example.com", "ops@example.com", "operations"); err != nil {
		t.Fatal(err)
	}
	if err := store.GrantCapability(ctx, "owner@example.com", "ops@example.com", CapabilityPartnerRead, "read partners"); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeCapability(ctx, "owner@example.com", "ops@example.com", CapabilityPartnerRead, "leave project"); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListAudit(ctx, 20)
	if err != nil {
		t.Fatal(err)
	}
	actions := make([]string, 0, len(items))
	for _, item := range items {
		actions = append(actions, item.Action)
	}
	for _, want := range []string{"admin.created", "admin.capability.granted", "admin.capability.revoked"} {
		found := false
		for _, action := range actions {
			if action == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("audit missing %q in %v", want, actions)
		}
	}
}
