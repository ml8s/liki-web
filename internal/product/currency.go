package product

import (
	"errors"
	"net/http"
	"strings"
)

// Currency identifies a supported payment currency.
type Currency string

const (
	CNY Currency = "CNY"
	USD Currency = "USD"
)

// CurrencyResolver selects a server-owned billing currency. Explicit resolvers
// ignore edge geo hints; trusted-edge resolvers may use them only when the
// deployment has declared its ingress trustworthy.
type CurrencyResolver interface {
	CurrencyFor(r *http.Request) Currency
}

type StaticCurrencyResolver struct{ Currency Currency }

func (r StaticCurrencyResolver) CurrencyFor(*http.Request) Currency {
	return r.Currency
}

type HeaderCurrencyResolver struct{}

func (HeaderCurrencyResolver) CurrencyFor(r *http.Request) Currency {
	return DetectCurrency(r.Header.Get("CF-IPCountry"), r.Header.Get("X-Geo-Country"))
}

const (
	CurrencySourceExplicit    = "explicit"
	CurrencySourceTrustedEdge = "trusted-edge"
)

func NewCurrencyResolver(source, region string) (CurrencyResolver, error) {
	switch source {
	case "", CurrencySourceTrustedEdge:
		return HeaderCurrencyResolver{}, nil
	case CurrencySourceExplicit:
		switch normalizeBillingRegion(region) {
		case "cn":
			return StaticCurrencyResolver{Currency: CNY}, nil
		case "us":
			return StaticCurrencyResolver{Currency: USD}, nil
		default:
			return nil, errors.New("LIKI_BILLING_REGION must be cn or us when LIKI_CURRENCY_SOURCE=explicit")
		}
	default:
		return nil, errors.New("LIKI_CURRENCY_SOURCE must be explicit or trusted-edge")
	}
}

func normalizeBillingRegion(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

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
