package payment

import "fmt"

// ProviderRoutePolicy chooses the payment provider after the business order's
// authoritative amount and currency have been loaded. Client-supplied provider
// values are hints only and never bypass this policy.
type ProviderRoutePolicy interface {
	Select(currency string, available []Provider) (Provider, error)
}

// CurrencyRoutePolicy implements the current product routing rule: CNY uses
// Xunhu and all non-CNY orders use Dodo. It fails closed when the required
// gateway is not configured.
type CurrencyRoutePolicy struct{}

func (CurrencyRoutePolicy) Select(currency string, available []Provider) (Provider, error) {
	want := ProviderDodo
	if currency == "CNY" {
		want = ProviderXunhu
	}
	for _, provider := range available {
		if provider == want {
			return want, nil
		}
	}
	return "", fmt.Errorf("payment: no gateway configured for %s orders", currency)
}
