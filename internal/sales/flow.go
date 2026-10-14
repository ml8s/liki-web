package sales

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
)

// Fulfiller adapts a Liki Chat order to the generic payment runtime and
// grants its entitlement only after a captured payment is durably fulfilled.
type Flow struct {
	orders       *OrderStore
	entitlements EntitlementProvisioner
	productID    product.Product
	duration     time.Duration
}

func NewFlow(orders *OrderStore, entitlements EntitlementProvisioner, productID product.Product, duration time.Duration) *Flow {
	return &Flow{orders: orders, entitlements: entitlements, productID: productID, duration: duration}
}

func (f *Flow) Product() string { return string(f.productID) }

func (f *Flow) DisplayName(locale string) string {
	_ = locale
	definition, err := product.DefinitionFor(f.productID)
	if err != nil {
		return string(f.productID)
	}
	return definition.Name
}

func (f *Flow) PrepareCheckout(ctx context.Context, businessOrderID string) (payment.BusinessOrder, error) {
	order, err := f.orders.getOrder(ctx, businessOrderID)
	if err != nil {
		return payment.BusinessOrder{}, err
	}
	if order.Status != OrderPending {
		return payment.BusinessOrder{}, fmt.Errorf("sales: order %s is %s, not pending", order.ID, order.Status)
	}
	return payment.BusinessOrder{
		ID: order.ID, UserID: order.UserID, Product: string(order.ProductID),
		AmountMinor: order.AmountMinor, Currency: order.Currency,
		Email: order.Email, Status: string(order.Status),
	}, nil
}

func (f *Flow) OnPaid(ctx context.Context, intent payment.PaymentIntent) error {
	order, changed, err := f.orders.MarkPaidByIntent(ctx, intent.ID)
	if err != nil {
		return fmt.Errorf("sales: mark order paid: %w", err)
	}
	if !changed {
		// A repeated webhook must remain idempotent for a paid order. A canceled
		// order must never grant access, even if a stale intent is marked paid.
		order, err = f.orders.GetOrderForUser(ctx, intent.UserID, intent.BusinessOrderID)
		if err != nil {
			return fmt.Errorf("sales: load paid order: %w", err)
		}
		if order.Status != OrderPaid {
			return fmt.Errorf("sales: order %s is %s, not paid", order.ID, order.Status)
		}
	}
	if err := f.entitlements.ProvisionProduct(
		ctx, order.UserID, order.Email, order.ProductID,
		"commerce_order", order.ID, f.duration,
	); err != nil {
		return fmt.Errorf("sales: grant entitlement: %w", err)
	}
	return nil
}

func (f *Flow) SuccessRedirect(intent payment.PaymentIntent) string {
	return appRedirect("/app/chat", intent.BusinessOrderID, "paid")
}

func (f *Flow) FailureRedirect(intent payment.PaymentIntent) string {
	return appRedirect("/app/orders", intent.BusinessOrderID, "unpaid")
}

func appRedirect(path, orderID, status string) string {
	if !strings.HasPrefix(path, "/app/") {
		return "/"
	}
	return path + "?order_id=" + url.QueryEscape(orderID) + "&status=" + url.QueryEscape(status)
}
