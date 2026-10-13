package admin

import "github.com/ml8s/liki-web/internal/liki"

func moneyMap(items []liki.AdminMoney) map[string]int64 {
	values := make(map[string]int64, len(items))
	for _, item := range items {
		values[item.Currency] = item.AmountMinor
	}
	return values
}

func addMoneyMaps(groups ...map[string]int64) map[string]int64 {
	merged := make(map[string]int64)
	for _, group := range groups {
		for currency, amount := range group {
			merged[currency] += amount
		}
	}
	return merged
}

func mergeAdminMoney(values ...map[string]int64) []liki.AdminMoney {
	merged := make(map[string]int64)
	for _, group := range values {
		for currency, amount := range group {
			merged[currency] += amount
		}
	}
	currencies := make([]string, 0, len(merged))
	for currency := range merged {
		currencies = append(currencies, currency)
	}
	sortStrings(currencies)
	items := make([]liki.AdminMoney, 0, len(currencies))
	for _, currency := range currencies {
		items = append(items, liki.AdminMoney{Currency: currency, AmountMinor: merged[currency]})
	}
	return items
}

func subtractAdminMoney(minuend, subtrahend map[string]int64) []liki.AdminMoney {
	values := make(map[string]int64, len(minuend))
	for currency, amount := range minuend {
		values[currency] = amount
	}
	for currency, amount := range subtrahend {
		values[currency] -= amount
	}
	return mergeAdminMoney(values)
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
