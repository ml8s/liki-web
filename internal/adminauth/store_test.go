package adminauth

import (
	"context"
	"path/filepath"
	"testing"

	"liki-web/internal/auth"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "platform.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return db
}

func adminUser(email string) auth.User {
	return auth.User{ID: "user-" + email, Email: email, Status: auth.UserActive}
}

func TestBootstrapAdminGrantsSuperCapability(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	if err := store.BootstrapAdminByEmail(ctx, "admin@example.com"); err != nil {
		t.Fatal(err)
	}
	ok, err := store.Authorize(ctx, adminUser("admin@example.com"), CapabilityPartnerApprove)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("super admin missing partner approve capability")
	}
	ok, err = store.Authorize(ctx, adminUser("admin@example.com"), CapabilityAdminSuper)
	if err != nil || !ok {
		t.Fatalf("super capability ok=%v err=%v", ok, err)
	}
}

func TestAuthorizeRejectsUnknownAndDisabledAdmin(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	if err := store.BootstrapAdminByEmail(ctx, "admin@example.com"); err != nil {
		t.Fatal(err)
	}

	if ok, err := store.Authorize(ctx, adminUser("unknown@example.com"), CapabilityPartnerApprove); err != nil || ok {
		t.Fatalf("unknown admin ok=%v err=%v", ok, err)
	}

	if err := store.EnsureAdmin(ctx, "admin@example.com", "successor@example.com", "keep one active owner"); err != nil {
		t.Fatal(err)
	}
	if err := store.GrantCapability(ctx, "admin@example.com", "successor@example.com", CapabilityAdminSuper, "successor owner"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetAdminStatus(ctx, "system", "admin@example.com", AdminDisabled, "test disable"); err != nil {
		t.Fatal(err)
	}
	if ok, err := store.Authorize(ctx, adminUser("admin@example.com"), CapabilityPartnerApprove); err != nil || ok {
		t.Fatalf("disabled admin ok=%v err=%v", ok, err)
	}
}

func TestAuthorizeRequiresExactCapability(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	if err := store.BootstrapAdminByEmail(ctx, "ops@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := store.GrantCapability(ctx, "system", "ops@example.com", CapabilityPayoutRead, "test grant"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE admin_capabilities SET status='revoked' WHERE admin_email=? AND capability=?`, "ops@example.com", CapabilityAdminSuper); err != nil {
		t.Fatal(err)
	}
	if ok, err := store.Authorize(ctx, adminUser("ops@example.com"), CapabilityPayoutRead); err != nil || !ok {
		t.Fatalf("payout read ok=%v err=%v", ok, err)
	}
	if ok, err := store.Authorize(ctx, adminUser("ops@example.com"), CapabilityPartnerApprove); err != nil || ok {
		t.Fatalf("missing capability ok=%v err=%v", ok, err)
	}
}

func TestAuditRecordsActorAndAction(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	entry := AuditEntry{
		ActorEmail: "admin@example.com",
		Action:     "partner.freeze",
		ObjectType: "partner",
		ObjectID:   "partner-1",
		Reason:     "fraud review",
	}
	if err := store.Audit(ctx, entry); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListAudit(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Action != entry.Action || items[0].ObjectID != entry.ObjectID {
		t.Fatalf("audit = %+v", items)
	}
}

func TestBootstrapDoesNotReactivateDisabledAdminOrRevokedCapability(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	if err := store.BootstrapAdminByEmail(ctx, "admin@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureAdmin(ctx, "admin@example.com", "successor@example.com", "keep one active owner"); err != nil {
		t.Fatal(err)
	}
	if err := store.GrantCapability(ctx, "admin@example.com", "successor@example.com", CapabilityAdminSuper, "successor owner"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetAdminStatus(ctx, "system", "admin@example.com", AdminDisabled, "test disable"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE admin_capabilities SET status='revoked' WHERE admin_email=? AND capability=?`, "admin@example.com", CapabilityAdminSuper); err != nil {
		t.Fatal(err)
	}

	if err := store.BootstrapAdminByEmail(ctx, "admin@example.com"); err != nil {
		t.Fatal(err)
	}

	var adminStatus, capabilityStatus string
	if err := store.db.QueryRow(`SELECT status FROM admin_users WHERE email=?`, "admin@example.com").Scan(&adminStatus); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT status FROM admin_capabilities WHERE admin_email=? AND capability=?`, "admin@example.com", CapabilityAdminSuper).Scan(&capabilityStatus); err != nil {
		t.Fatal(err)
	}
	if adminStatus != string(AdminDisabled) {
		t.Fatalf("disabled admin was reactivated: %s", adminStatus)
	}
	if capabilityStatus != "revoked" {
		t.Fatalf("revoked capability was restored: %s", capabilityStatus)
	}
}

func TestAuthorizeRejectsMismatchedUnifiedUserID(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	if err := store.BootstrapAdminByEmail(ctx, "admin@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE admin_users SET user_id='original-user' WHERE email=?`, "admin@example.com"); err != nil {
		t.Fatal(err)
	}
	ok, err := store.Authorize(ctx, adminUser("admin@example.com"), CapabilityAdminSuper)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("different unified user id should not authorize")
	}
}
