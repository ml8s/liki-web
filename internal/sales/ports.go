package sales

import (
	"context"
	"time"

	"github.com/ml8s/liki-web/internal/product"
)

// EntitlementProvisioner receives sales facts that justify granting product
// access. Sales owns the fact; entitlement owns the meaning of access.
type EntitlementProvisioner interface {
	ProvisionProduct(
		ctx context.Context,
		userID, email string,
		productID product.Product,
		source, sourceID string,
		duration time.Duration,
	) error

	ProvisionSubscriptionPeriod(
		ctx context.Context,
		userID, email string,
		productID product.Product,
		sourceID string,
		startsAt, expiresAt time.Time,
	) error
}
