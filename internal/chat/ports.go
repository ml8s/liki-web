package chat

import (
	"context"

	"github.com/ml8s/liki-web/internal/product"
)

// AccessAuthorizer decouples chat from the concrete entitlement store.
type AccessAuthorizer interface {
	HasSkill(ctx context.Context, userID string, skill product.Skill) (bool, error)
}
