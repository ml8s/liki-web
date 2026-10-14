package product

import (
	"testing"

	"github.com/ml8s/liki-web/internal/money"
)

func TestProductCatalogUsesInjectedMinorUnitPrices(t *testing.T) {
	catalog := NewCatalog(CatalogConfig{
		OneTimePriceMinorCNY:      7900,
		OneTimePriceMinorUSD:      2990,
		SubscriptionPriceMinorUSD: 1990,
	})
	definition, err := catalog.OfferingFor(Chat)
	if err != nil {
		t.Fatal(err)
	}
	if len(definition.Offers) != 3 {
		t.Fatalf("purchase options = %#v, want one-time CNY/USD and subscription USD", definition.Offers)
	}
	want := map[BillingMode]money.Amount{
		BillingModeOneTime:      {Currency: string(CNY), MinorUnits: 7900},
		BillingModeSubscription: {Currency: string(USD), MinorUnits: 1990},
	}
	for mode, expected := range want {
		got, priceErr := catalog.Price(Chat, mode, Currency(expected.Currency))
		if priceErr != nil {
			t.Fatalf("Price(%s/%s): %v", mode, expected.Currency, priceErr)
		}
		if got != expected {
			t.Fatalf("price[%s/%s] = %#v, want %#v", mode, expected.Currency, got, expected)
		}
	}
}

func TestProductCatalogPriceFailsClosed(t *testing.T) {
	catalog := NewCatalog(CatalogConfig{
		OneTimePriceMinorCNY: 7900,
		OneTimePriceMinorUSD: 2990,
	})
	if _, err := catalog.Price(Chat, BillingModeSubscription, USD); err == nil {
		t.Fatal("Price(subscription) = nil, want unsupported-mode error")
	}
}
