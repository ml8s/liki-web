package birthprofile

import (
	"context"
	"errors"
	"testing"
)

// 越权拒绝矩阵：用户 B 不能读/改/删用户 A 的出生档案。
func TestCrossUserAccessDenied(t *testing.T) {
	ctx := context.Background()
	store, _ := newStoreFixture(t)

	profile, err := store.Create(ctx, "user_1", validInput())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("get", func(t *testing.T) {
		if _, err := store.Get(ctx, "user_2", profile.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})

	t.Run("update", func(t *testing.T) {
		input := validInput()
		input.Nickname = "被改"
		if _, err := store.Update(ctx, "user_2", profile.ID, profile.Version, input); !errors.Is(err, ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})

	t.Run("soft delete", func(t *testing.T) {
		if err := store.SoftDelete(ctx, "user_2", profile.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})

	t.Run("owner still works", func(t *testing.T) {
		kept, err := store.Get(ctx, "user_1", profile.ID)
		if err != nil {
			t.Fatalf("owner access broken: %v", err)
		}
		if kept.Nickname != "妈妈" {
			t.Fatalf("profile mutated by foreign attempts: %q", kept.Nickname)
		}
	})
}
