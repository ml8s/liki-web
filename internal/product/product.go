// Package product defines the canonical catalog shared by product handlers and
// payment. Product IDs, currencies, and prices must not be invented by callers.
package product

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// OnPaid is a small callback for legacy admin-created orders.
type OnPaid func(ctx context.Context, orderID string)

// Product identifies a paid product.
type Product string

const (
	ProductNaming  Product = "naming"
	ProductMingShu Product = "mingshu"
	ProductHePan   Product = "hepan"
)

// Currency identifies a supported payment currency.
type Currency string

const (
	CNY Currency = "CNY"
	USD Currency = "USD"
)

type Definition struct {
	ID          Product
	AmountMinor map[Currency]int64
}

var catalog = map[Product]Definition{
	ProductNaming: {
		ID: ProductNaming,
		AmountMinor: map[Currency]int64{
			CNY: 990,
			USD: 990,
		},
	},
	ProductMingShu: {
		ID: ProductMingShu,
		AmountMinor: map[Currency]int64{
			CNY: 990,
			USD: 990,
		},
	},
	ProductHePan: {
		ID: ProductHePan,
		AmountMinor: map[Currency]int64{
			CNY: 990,
			USD: 990,
		},
	},
}

// NormalizeProduct maps the historical lifebook ID to its canonical domain ID.
func NormalizeProduct(id Product) Product {
	if id == "lifebook" {
		return ProductMingShu
	}
	return id
}

// IsValidProduct reports whether a canonical product exists in the catalog.
func IsValidProduct(id Product) bool {
	_, ok := catalog[NormalizeProduct(id)]
	return ok
}

// AmountMinor returns the catalog price for a product and currency. Unknown
// combinations fail closed; handlers must never invent a price.
func AmountMinor(id Product, currency Currency) (int64, error) {
	definition, ok := catalog[NormalizeProduct(id)]
	if !ok {
		return 0, fmt.Errorf("product: unknown product %q", id)
	}
	amount, ok := definition.AmountMinor[currency]
	if !ok || amount <= 0 {
		return 0, fmt.Errorf("product: %s does not support currency %s", id, currency)
	}
	return amount, nil
}

// NewOrderID generates a new UUID-based order ID.
func NewOrderID() string {
	return uuid.NewString()
}

// DetectCurrency derives billing currency from trusted edge/geo headers.
// Cloudflare's country is authoritative; the fallback is set by the trusted
// reverse proxy. Absence of geo information defaults to CNY.
func DetectCurrency(cfCountry, geoCountry string) Currency {
	switch {
	case cfCountry != "":
		if cfCountry == "CN" {
			return CNY
		}
		return USD
	case geoCountry != "":
		if geoCountry == "CN" {
			return CNY
		}
		return USD
	default:
		return CNY
	}
}
