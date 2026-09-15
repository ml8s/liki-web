package entitlement

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"liki-web/internal/auth"

	_ "modernc.org/sqlite"
)

func openEntitlementStore(t *testing.T) (*Store, *sql.DB, string) {
	t.Helper()
	db := openMemoryDB(t)
	authStore := auth.NewStore(db, []byte("test-hmac-key"))
	if err := authStore.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	store := NewStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	userID := createPlatformUser(t, db, "user@example.com")
	return store, db, userID
}

func createPlatformUser(t *testing.T, db *sql.DB, email string) string {
	t.Helper()
	userID := "user_" + email
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.ExecContext(context.Background(), `
INSERT INTO users (id,email,status,locale,created_at,updated_at) VALUES (?,?, 'active','zh-Hant',?,?)
ON CONFLICT(email) DO NOTHING`, userID, email, now, now)
	if err != nil {
		t.Fatal(err)
	}
	return userID
}

func TestGrantForProductIsIdempotentAndRejectsLegacyProducts(t *testing.T) {
	ctx := context.Background()
	store, _, userID := openEntitlementStore(t)

	first, err := store.GrantForProduct(ctx, userID, "User@Example.com", ProductChat, "payment_intent", "pay_1", 30*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if first.ProductID != ProductChat || len(first.Skills) != 1 || first.Skills[0] != SkillLiki {
		t.Fatalf("first grant = %+v", first)
	}

	second, err := store.GrantForProduct(ctx, userID, "user@example.com", ProductChat, "payment_intent", "pay_1", 30*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("duplicate payment created %q; want %q", second.ID, first.ID)
	}

	_, err = store.GrantForProduct(ctx, userID, "user@example.com", "unknown-product", "payment_intent", "pay_2", 30*24*time.Hour)
	if err == nil {
		t.Fatal("expected unknown product to fail")
	}
	for _, legacy := range []Product{"naming", "mingshu", "hepan", "liki-naming", "liki-chat-all"} {
		if _, err := store.GrantForProduct(ctx, userID, "user@example.com", legacy, "payment_intent", string(legacy), 24*time.Hour); err == nil {
			t.Fatalf("legacy product %q was accepted", legacy)
		}
	}
}

func TestChatProductGrantsUnifiedSkillAndExpiry(t *testing.T) {
	ctx := context.Background()
	store, _, userID := openEntitlementStore(t)

	item, err := store.GrantForProduct(ctx, userID, "user@example.com", ProductChat, "payment_intent", "pay_chat", 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if item.ProductID != ProductChat || len(item.Skills) != 1 || item.Skills[0] != SkillLiki {
		t.Fatalf("chat grant = %+v", item)
	}

	allowed, err := store.AllowedSkills(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(allowed) != 1 || allowed[0] != SkillLiki {
		t.Fatalf("allowed = %+v, want [liki]", allowed)
	}

	store.nowFunc = func() time.Time { return time.Now().UTC().Add(25 * time.Hour) }
	if n, err := store.ExpireDue(ctx); err != nil || n != 1 {
		t.Fatalf("ExpireDue = %d, %v; want 1, nil", n, err)
	}
	items, err := store.ListActiveByUser(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("active after expiry = %+v", items)
	}
}

// 存量（合并前）权益存的是旧四个 skill，仍应授予统一 skill，旧会话也继续可用。
func TestLegacyEntitlementGrantsUnifiedSkill(t *testing.T) {
	ctx := context.Background()
	store, db, userID := openEntitlementStore(t)
	now := time.Now().UTC()
	_, err := db.ExecContext(ctx, `
INSERT INTO entitlements (id,user_id,email,product_id,skills,status,source,source_id,starts_at,expires_at,created_at,updated_at)
VALUES (?,?,?,?,?, 'active','payment_intent','legacy_pay',?,?,?,?)`,
		"ent_legacy", userID, "user@example.com", string(ProductChat),
		`["liki-naming","liki-bazi","liki-divination","liki-fengshui"]`,
		now.Format(time.RFC3339), now.Add(24*time.Hour).Format(time.RFC3339),
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	allowed, err := store.AllowedSkills(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(allowed) != 1 || allowed[0] != SkillLiki {
		t.Fatalf("legacy allowed = %+v, want [liki]", allowed)
	}
	for _, skill := range []Skill{SkillLiki, SkillBazi, SkillNaming} {
		ok, err := store.HasSkill(ctx, userID, skill)
		if err != nil || !ok {
			t.Fatalf("HasSkill(%s) = %v, %v; want true", skill, ok, err)
		}
	}
}
