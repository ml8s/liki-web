package liki

import (
	"testing"

	"github.com/ml8s/liki-web/internal/money"
)

func TestCommerceCatalogUsesInjectedMinorUnitPrices(t *testing.T) {
	catalog := NewCommerceCatalog(CommerceCatalogConfig{
		OneTimePriceMinorCNY:      7900,
		OneTimePriceMinorUSD:      2990,
		SubscriptionPriceMinorUSD: 1990,
	})
	definition, err := catalog.DefinitionFor(ProductChat)
	if err != nil {
		t.Fatal(err)
	}
	if len(definition.PurchaseOptions) != 3 {
		t.Fatalf("purchase options = %#v, want one-time CNY/USD and subscription USD", definition.PurchaseOptions)
	}
	want := map[BillingMode]money.Amount{
		BillingModeOneTime:      {Currency: string(CurrencyCNY), MinorUnits: 7900},
		BillingModeSubscription: {Currency: string(CurrencyUSD), MinorUnits: 1990},
	}
	for mode, expected := range want {
		got, priceErr := catalog.Price(ProductChat, mode, CurrencyCode(expected.Currency))
		if priceErr != nil {
			t.Fatalf("Price(%s/%s): %v", mode, expected.Currency, priceErr)
		}
		if got != expected {
			t.Fatalf("price[%s/%s] = %#v, want %#v", mode, expected.Currency, got, expected)
		}
	}
}

func TestCommerceCatalogPriceFailsClosed(t *testing.T) {
	catalog := NewCommerceCatalog(CommerceCatalogConfig{
		OneTimePriceMinorCNY: 7900,
		OneTimePriceMinorUSD: 2990,
	})
	if _, err := catalog.Price(ProductChat, BillingModeSubscription, CurrencyUSD); err == nil {
		t.Fatal("Price(subscription) = nil, want unsupported-mode error")
	}
}
