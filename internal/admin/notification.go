package admin

import (
	"context"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/entitlement"
)

// CompensationNotifier queues the customer-facing notification after the
// entitlement transaction has committed. Production uses the same durable
// transactional-email queue as order receipts; implementations return an error
// when the notification could not be durably queued.
type CompensationNotifier func(ctx context.Context, user auth.User, item entitlement.Entitlement, template string) error
