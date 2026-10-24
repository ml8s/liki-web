package sales

import (
	"net/http/httptest"
	"testing"

	"github.com/ml8s/liki-web/internal/product"
)

func TestExplicitCurrencyResolverIgnoresSpoofedHeaders(t *testing.T) {
	resolver, err := NewCurrencyResolver(CurrencySourceExplicit, "US")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/", nil)
	request.Header.Set("CF-IPCountry", "CN")
	request.Header.Set("X-Geo-Country", "CN")
	if got := resolver.CurrencyFor(request); got != product.USD {
		t.Fatalf("currency = %q, want USD", got)
	}
}

func TestTrustedEdgeCurrencyResolverUsesHeaders(t *testing.T) {
	resolver, err := NewCurrencyResolver(CurrencySourceTrustedEdge, "")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/", nil)
	request.Header.Set("CF-IPCountry", "CN")
	if got := resolver.CurrencyFor(request); got != product.CNY {
		t.Fatalf("currency = %q, want CNY", got)
	}
}

func TestExplicitCurrencyResolverRequiresValidRegion(t *testing.T) {
	if _, err := NewCurrencyResolver(CurrencySourceExplicit, "mars"); err == nil {
		t.Fatal("error = nil, want invalid region")
	}
}

func TestTrustedEdgeCurrencyResolverDefaultsToUSDWithoutGeo(t *testing.T) {
	resolver, err := NewCurrencyResolver(CurrencySourceTrustedEdge, "")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/", nil)
	if got := resolver.CurrencyFor(request); got != product.USD {
		t.Fatalf("currency = %q, want USD", got)
	}
}

func TestDetectCurrencyUsesGeoFallbackAndDefaultsToUSD(t *testing.T) {
	if got := DetectCurrency("", "CN"); got != product.CNY {
		t.Fatalf("geo CN = %q, want CNY", got)
	}
	if got := DetectCurrency("", "US"); got != product.USD {
		t.Fatalf("geo US = %q, want USD", got)
	}
	if got := DetectCurrency("", ""); got != product.USD {
		t.Fatalf("no geo = %q, want USD", got)
	}
}
