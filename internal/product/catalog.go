package product

import (
	"fmt"

	"github.com/ml8s/liki-web/internal/money"
)

type BillingMode string

const (
	BillingModeOneTime      BillingMode = "one_time"
	BillingModeSubscription BillingMode = "subscription"
)

type Offer struct {
	BillingMode BillingMode
	Price       money.Amount
}

// Definition describes the sales-facing product card (name, description,
// pricing). It shares the Product identifier with the entitlement definition.
type ProductOffering struct {
	ID          Product
	Name        string
	Description string
	Offers      []Offer
}

type CatalogConfig struct {
	OneTimePriceMinorCNY      int64
	OneTimePriceMinorUSD      int64
	SubscriptionPriceMinorUSD int64
}

// Catalog is immutable runtime pricing. It is injected at startup so deployment
// configuration—not a compiled test value—is the source of truth.
type Catalog struct {
	definitions map[Product]ProductOffering
}

func NewCatalog(config CatalogConfig) *Catalog {
	chat := ProductOffering{
		ID:          Chat,
		Name:        "Liki Chat",
		Description: "BaZi, ZiWei, divination, Feng Shui, and naming — one consultant in one Chat.",
	}
	if config.OneTimePriceMinorCNY > 0 {
		chat.Offers = append(chat.Offers, Offer{
			BillingMode: BillingModeOneTime,
			Price:       money.Amount{Currency: string(CNY), MinorUnits: config.OneTimePriceMinorCNY},
		})
	}
	if config.OneTimePriceMinorUSD > 0 {
		chat.Offers = append(chat.Offers, Offer{
			BillingMode: BillingModeOneTime,
			Price:       money.Amount{Currency: string(USD), MinorUnits: config.OneTimePriceMinorUSD},
		})
	}
	if config.SubscriptionPriceMinorUSD > 0 {
		chat.Offers = append(chat.Offers, Offer{
			BillingMode: BillingModeSubscription,
			Price:       money.Amount{Currency: string(USD), MinorUnits: config.SubscriptionPriceMinorUSD},
		})
	}
	return &Catalog{definitions: map[Product]ProductOffering{Chat: chat}}
}

// commerceCatalog returns the sales-facing product cards: presentation and
// price per currency. Entitlement semantics (which skills a purchase grants)
// live in ProductCatalog — see the boundary note there.
func (c *Catalog) List() []ProductOffering {
	if c == nil || c.definitions == nil {
		return nil
	}
	return []ProductOffering{c.definitions[Chat]}
}

func (c *Catalog) OfferingFor(id Product) (ProductOffering, error) {
	if c == nil || c.definitions == nil {
		return ProductOffering{}, fmt.Errorf("product: sales catalog is not configured")
	}
	item, ok := c.definitions[id]
	if !ok {
		return ProductOffering{}, fmt.Errorf("product: unknown product %q", id)
	}
	return item, nil
}

func (c *Catalog) Price(id Product, mode BillingMode, currency Currency) (money.Amount, error) {
	if currency == "" {
		return money.Amount{}, fmt.Errorf("product: currency is required")
	}
	item, err := c.OfferingFor(id)
	if err != nil {
		return money.Amount{}, err
	}
	for _, option := range item.Offers {
		if option.BillingMode == mode && option.Price.Currency == string(currency) {
			if option.Price.MinorUnits <= 0 {
				break
			}
			return option.Price, nil
		}
	}
	return money.Amount{}, fmt.Errorf("product: product %s does not support %s pricing in %s", id, mode, currency)
}
