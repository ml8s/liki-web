package liki

import "testing"

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
	want := map[BillingMode]MoneyMinor{
		BillingModeOneTime:      {CurrencyCode: CurrencyCNY, MinorUnits: 7900},
		BillingModeSubscription: {CurrencyCode: CurrencyUSD, MinorUnits: 1990},
	}
	for mode, expected := range want {
		got, priceErr := catalog.Price(ProductChat, mode, expected.CurrencyCode)
		if priceErr != nil {
			t.Fatalf("Price(%s/%s): %v", mode, expected.CurrencyCode, priceErr)
		}
		if got != expected {
			t.Fatalf("price[%s/%s] = %#v, want %#v", mode, expected.CurrencyCode, got, expected)
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
