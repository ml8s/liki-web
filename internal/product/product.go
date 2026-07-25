// Package product defines the product types and pricing shared between agent and payment layers.
package product

import (
	"context"

	"github.com/google/uuid"
)

// CheckoutOrder holds the minimal order fields needed by shared payment handlers.
type CheckoutOrder struct {
	Status  string
	Amount  int
	Product string // order product type (used by naming for dynamic routing)
}

// OrderStore is the store interface for shared payment lifecycle handlers.
// Each product domain (hepan, lifebook, naming) implements this on its Store.
type OrderStore interface {
	GetCheckoutOrder(ctx context.Context, orderID string) (*CheckoutOrder, error)
	UpdateEmail(ctx context.Context, orderID, email string) error
	UpdateProvider(ctx context.Context, orderID, provider string) error
	MarkPaidViaCallback(ctx context.Context, orderID string) (bool, string, error)
}

// OnPaid is called when an order is paid.
type OnPaid func(ctx context.Context, orderID string)

// OnReportReady is called when a report has been generated.
type OnReportReady func(ctx context.Context, orderID string)

// Product identifies a paid report product.
type Product string

const ProductNaming Product = "naming"
const ProductMingShu Product = "lifebook"

func (p Product) EmailSubject() string {
	switch p {
	case ProductNaming:
		return "服务已开通"
	case ProductMingShu:
		return "灵机命书报告已生成"
	default:
		return "服务已开通"
	}
}

// Currency identifies a payment currency.
type Currency string

const (
	CNY Currency = "CNY"
	USD Currency = "USD"
)

// NewOrderID generates a new UUID-based order ID.
func NewOrderID() string {
	return uuid.New().String()
}

// DetectCurrency returns the currency for the given country codes.
func DetectCurrency(cfCountry, geoCountry string) Currency {
	if cfCountry == "CN" || geoCountry == "CN" {
		return CNY
	}
	if cfCountry != "" || geoCountry != "" {
		return USD
	}
	return CNY
}

// NamingAmountCents returns the price in cents for the naming product.
func NamingAmountCents(c Currency) int {
	switch c {
	case USD:
		return 2990
	case CNY:
		return 19900
	default:
		return 0
	}
}
