package naming

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func nmSeedOrder(t *testing.T, s *Store) string {
	t.Helper()
	oid := uuid.NewString()
	if err := s.CreateOrder(context.Background(), oid, "naming", 990, "CNY", "test@t.com", "zh-Hans"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return oid
}
