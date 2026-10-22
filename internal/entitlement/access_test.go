package entitlement

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

// 越权拒绝矩阵：用户 A 的权益不得出现在用户 B 的读取结果中。
func TestCrossUserEntitlementsIsolated(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store := NewStore(db)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GrantForProduct(ctx, "user-a", "a@example.com", product.Chat, "test", "src-1", time.Hour); err != nil {
		t.Fatal(err)
	}

	t.Run("allowed skills", func(t *testing.T) {
		skills, err := store.AllowedSkills(ctx, "user-b")
		if err != nil {
			t.Fatal(err)
		}
		for _, skill := range skills {
			if skill == product.SkillConsultation {
				t.Fatalf("foreign entitlement leaked to user-b: %v", skills)
			}
		}
	})

	t.Run("active list", func(t *testing.T) {
		list, err := store.ListActiveByUser(ctx, "user-b")
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 0 {
			t.Fatalf("foreign entitlements leaked: %d", len(list))
		}
	})

	t.Run("owner still works", func(t *testing.T) {
		skills, err := store.AllowedSkills(ctx, "user-a")
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, skill := range skills {
			if skill == product.SkillConsultation {
				found = true
			}
		}
		if !found {
			t.Fatalf("owner entitlement missing: %v", skills)
		}
	})
}
