package liki

import (
	"fmt"
)

type CurrencyCode = Currency

const (
	CurrencyCNY CurrencyCode = "CNY"
	CurrencyUSD CurrencyCode = "USD"
)

type BillingMode string

const (
	BillingModeOneTime      BillingMode = "one_time"
	BillingModeSubscription BillingMode = "subscription"
)

// MoneyMinor is ISO-4217 minor units: cents for USD and fen for CNY.
type MoneyMinor struct {
	CurrencyCode CurrencyCode
	MinorUnits   int64
}

type PurchaseOption struct {
	BillingMode BillingMode
	Price       MoneyMinor
}

// Definition describes the commerce-facing product card (name, description,
// pricing). It shares the Product identifier with the entitlement definition.
type Definition struct {
	ID              Product
	Name            string
	Description     string
	PurchaseOptions []PurchaseOption
}

type CommerceCatalogConfig struct {
	OneTimePriceMinorCNY      int64
	OneTimePriceMinorUSD      int64
	SubscriptionPriceMinorUSD int64
}

// Catalog is immutable runtime pricing. It is injected at startup so deployment
// configuration—not a compiled test value—is the source of truth.
type Catalog struct {
	definitions map[Product]Definition
}

func NewCommerceCatalog(config CommerceCatalogConfig) *Catalog {
	chat := Definition{
		ID:          ProductChat,
		Name:        "Liki Chat",
		Description: "Naming, BaZi, divination, and Feng Shui in one Chat.",
	}
	if config.OneTimePriceMinorCNY > 0 {
		chat.PurchaseOptions = append(chat.PurchaseOptions, PurchaseOption{
			BillingMode: BillingModeOneTime,
			Price:       MoneyMinor{CurrencyCode: CurrencyCNY, MinorUnits: config.OneTimePriceMinorCNY},
		})
	}
	if config.OneTimePriceMinorUSD > 0 {
		chat.PurchaseOptions = append(chat.PurchaseOptions, PurchaseOption{
			BillingMode: BillingModeOneTime,
			Price:       MoneyMinor{CurrencyCode: CurrencyUSD, MinorUnits: config.OneTimePriceMinorUSD},
		})
	}
	if config.SubscriptionPriceMinorUSD > 0 {
		chat.PurchaseOptions = append(chat.PurchaseOptions, PurchaseOption{
			BillingMode: BillingModeSubscription,
			Price:       MoneyMinor{CurrencyCode: CurrencyUSD, MinorUnits: config.SubscriptionPriceMinorUSD},
		})
	}
	return &Catalog{definitions: map[Product]Definition{ProductChat: chat}}
}

// commerceCatalog returns the commerce-facing product cards: presentation and
// price per currency. Entitlement semantics (which skills a purchase grants)
// live in ProductCatalog — see the boundary note there.
func (c *Catalog) List() []Definition {
	if c == nil || c.definitions == nil {
		return nil
	}
	return []Definition{c.definitions[ProductChat]}
}

func (c *Catalog) DefinitionFor(id Product) (Definition, error) {
	if c == nil || c.definitions == nil {
		return Definition{}, fmt.Errorf("liki: commerce catalog is not configured")
	}
	item, ok := c.definitions[id]
	if !ok {
		return Definition{}, fmt.Errorf("liki: unknown product %q", id)
	}
	return item, nil
}

func (c *Catalog) Price(id Product, mode BillingMode, currency CurrencyCode) (MoneyMinor, error) {
	if currency == "" {
		return MoneyMinor{}, fmt.Errorf("liki: currency is required")
	}
	item, err := c.DefinitionFor(id)
	if err != nil {
		return MoneyMinor{}, err
	}
	for _, option := range item.PurchaseOptions {
		if option.BillingMode == mode && option.Price.CurrencyCode == currency {
			if option.Price.MinorUnits <= 0 {
				break
			}
			return option.Price, nil
		}
	}
	return MoneyMinor{}, fmt.Errorf("liki: product %s does not support %s pricing in %s", id, mode, currency)
}
