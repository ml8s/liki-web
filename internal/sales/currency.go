package sales

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ml8s/liki-web/internal/product"
)

// CurrencyResolver selects a server-owned billing currency. Explicit resolvers
// ignore edge geo hints; trusted-edge resolvers may use them only when the
// deployment has declared its ingress trustworthy.
type CurrencyResolver interface {
	CurrencyFor(r *http.Request) product.Currency
}

type StaticCurrencyResolver struct{ Currency product.Currency }

func (r StaticCurrencyResolver) CurrencyFor(*http.Request) product.Currency {
	return r.Currency
}

type HeaderCurrencyResolver struct{}

func (HeaderCurrencyResolver) CurrencyFor(r *http.Request) product.Currency {
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
			return StaticCurrencyResolver{Currency: product.CNY}, nil
		case "us":
			return StaticCurrencyResolver{Currency: product.USD}, nil
		default:
			return nil, errors.New("sales: LIKI_BILLING_REGION must be cn or us when LIKI_CURRENCY_SOURCE=explicit")
		}
	default:
		return nil, errors.New("sales: LIKI_CURRENCY_SOURCE must be explicit or trusted-edge")
	}
}

func normalizeBillingRegion(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// DetectCurrency derives billing currency from trusted edge/geo headers.
// Cloudflare's country is authoritative; the fallback is set by the trusted
// reverse proxy. Absence of geo information defaults to CNY.
func DetectCurrency(cfCountry, geoCountry string) product.Currency {
	switch {
	case cfCountry != "":
		if cfCountry == "CN" {
			return product.CNY
		}
		return product.USD
	case geoCountry != "":
		if geoCountry == "CN" {
			return product.CNY
		}
		return product.USD
	default:
		return product.CNY
	}
}
