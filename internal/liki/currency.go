package liki

// Currency identifies a supported payment currency.
type Currency string

const (
	CNY Currency = "CNY"
	USD Currency = "USD"
)

// DetectCurrency derives billing currency from trusted edge/geo headers.
// Cloudflare's country is authoritative; the fallback is set by the trusted
// reverse proxy. Absence of geo information defaults to CNY.
func DetectCurrency(cfCountry, geoCountry string) Currency {
	switch {
	case cfCountry != "":
		if cfCountry == "CN" {
			return CNY
		}
		return USD
	case geoCountry != "":
		if geoCountry == "CN" {
			return CNY
		}
		return USD
	default:
		return CNY
	}
}
