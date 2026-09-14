package commerce

import (
	"fmt"

	"liki-web/internal/entitlement"
)

// Product shares the stable entitlement product identifier. Commerce owns
// checkout and orders; entitlement owns the resulting capability grant.
type Product = entitlement.Product

const ProductChat = entitlement.ProductChat

type Definition struct {
	ID          Product
	Name        string
	Description string
	AmountMinor map[string]int64
}

func catalog() map[Product]Definition {
	return map[Product]Definition{
		ProductChat: {
			ID:          ProductChat,
			Name:        "Liki Chat",
			Description: "Naming, BaZi, divination, and Feng Shui in one Chat.",
			AmountMinor: map[string]int64{"CNY": 2990, "USD": 2990},
		},
	}
}

// Catalog returns a stable list for API rendering.
func Catalog() []Definition {
	return []Definition{catalog()[ProductChat]}
}

func DefinitionFor(id Product) (Definition, error) {
	item, ok := catalog()[id]
	if !ok {
		return Definition{}, fmt.Errorf("commerce: unknown product %q", id)
	}
	return item, nil
}

func AmountMinor(id Product, currency string) (int64, error) {
	item, err := DefinitionFor(id)
	if err != nil {
		return 0, err
	}
	amount, ok := item.AmountMinor[currency]
	if !ok || amount <= 0 {
		return 0, fmt.Errorf("commerce: product %s does not support currency %s", id, currency)
	}
	return amount, nil
}

func DetectCurrency(cfCountry, geoCountry string) string {
	switch {
	case cfCountry == "CN" || geoCountry == "CN":
		return "CNY"
	case cfCountry != "" || geoCountry != "":
		return "USD"
	default:
		return "CNY"
	}
}
