package liki

import "testing"

func TestChatCatalogUsesOneUnitTestPrice(t *testing.T) {
	currencies := map[string]int64{"CNY": 100, "USD": 100}
	for currency, want := range currencies {
		got, err := AmountMinor(ProductChat, currency)
		if err != nil {
			t.Fatalf("AmountMinor(%s): %v", currency, err)
		}
		if got != want {
			t.Fatalf("AmountMinor(%s) = %d, want %d", currency, got, want)
		}
	}
}
