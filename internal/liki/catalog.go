package liki

import (
	"fmt"
)

// Definition describes the commerce-facing product card (name, description,
// pricing). It shares the Product identifier with the entitlement definition.
type Definition struct {
	ID          Product
	Name        string
	Description string
	AmountMinor map[string]int64
}

// commerceCatalog returns the commerce-facing product cards: presentation and
// price per currency. Entitlement semantics (which skills a purchase grants)
// live in ProductCatalog — see the boundary note there.
func commerceCatalog() map[Product]Definition {
	return map[Product]Definition{
		ProductChat: {
			ID:          ProductChat,
			Name:        "Liki Chat",
			Description: "Naming, BaZi, divination, and Feng Shui in one Chat.",
			// Temporary test price: CNY 1.00 / USD 1.00.
			AmountMinor: map[string]int64{"CNY": 100, "USD": 100},
		},
	}
}

// Catalog returns a stable list for API rendering.
func Catalog() []Definition {
	return []Definition{commerceCatalog()[ProductChat]}
}

// DefinitionFor fails closed for unknown products.
func DefinitionFor(id Product) (Definition, error) {
	item, ok := commerceCatalog()[id]
	if !ok {
		return Definition{}, fmt.Errorf("liki: unknown product %q", id)
	}
	return item, nil
}

// AmountMinor returns the catalog price for a product and currency. Unknown
// combinations fail closed; handlers must never invent a price.
func AmountMinor(id Product, currency string) (int64, error) {
	item, err := DefinitionFor(id)
	if err != nil {
		return 0, err
	}
	amount, ok := item.AmountMinor[currency]
	if !ok || amount <= 0 {
		return 0, fmt.Errorf("liki: product %s does not support currency %s", id, currency)
	}
	return amount, nil
}
