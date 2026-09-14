package commerce

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"liki-web/internal/entitlement"
	"liki-web/internal/payment"
)

// Fulfiller adapts a commercial Chat order to the generic payment runtime and
// grants its entitlement only after a captured payment is durably fulfilled.
type Fulfiller struct {
	orders       *Store
	entitlements *entitlement.Store
	productID    Product
	duration     time.Duration
}

func NewFulfiller(orders *Store, entitlements *entitlement.Store, productID Product, duration time.Duration) *Fulfiller {
	return &Fulfiller{orders: orders, entitlements: entitlements, productID: productID, duration: duration}
}

func (f *Fulfiller) Product() string { return string(f.productID) }

func (f *Fulfiller) PrepareCheckout(ctx context.Context, businessOrderID string) (payment.BusinessOrder, error) {
	order, err := f.orders.getOrder(ctx, businessOrderID)
	if err != nil {
		return payment.BusinessOrder{}, err
	}
	if order.Status != OrderPending {
		return payment.BusinessOrder{}, fmt.Errorf("commerce: order %s is %s, not pending", order.ID, order.Status)
	}
	return payment.BusinessOrder{
		ID: order.ID, UserID: order.UserID, Product: string(order.ProductID),
		AmountMinor: order.AmountMinor, Currency: order.Currency,
		Email: order.Email, Locale: "zh-Hant", Status: string(order.Status),
	}, nil
}

func (f *Fulfiller) OnPaid(ctx context.Context, intent payment.PaymentIntent) error {
	order, changed, err := f.orders.MarkPaidByIntent(ctx, intent.ID)
	if err != nil {
		return fmt.Errorf("commerce: mark order paid: %w", err)
	}
	if !changed {
		// A repeated webhook must remain idempotent for a paid order. A canceled
		// order must never grant access, even if a stale intent is marked paid.
		order, err = f.orders.GetOrderForUser(ctx, intent.UserID, intent.BusinessOrderID)
		if err != nil {
			return fmt.Errorf("commerce: load paid order: %w", err)
		}
		if order.Status != OrderPaid {
			return fmt.Errorf("commerce: order %s is %s, not paid", order.ID, order.Status)
		}
	}
	if _, err := f.entitlements.GrantForProduct(
		ctx, order.UserID, order.Email, order.ProductID,
		"commerce_order", order.ID, f.duration,
	); err != nil {
		return fmt.Errorf("commerce: grant entitlement: %w", err)
	}
	return nil
}

func (f *Fulfiller) SuccessRedirect(intent payment.PaymentIntent) string {
	return appRedirect("/app/chat", intent.BusinessOrderID, "paid")
}

func (f *Fulfiller) FailureRedirect(intent payment.PaymentIntent) string {
	return appRedirect("/app/orders", intent.BusinessOrderID, "unpaid")
}

func appRedirect(path, orderID, status string) string {
	if !strings.HasPrefix(path, "/app/") {
		return "/"
	}
	return path + "?order_id=" + url.QueryEscape(orderID) + "&status=" + url.QueryEscape(status)
}
