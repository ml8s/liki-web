package product

import (
	"net/http/httptest"
	"testing"
)

func TestExplicitCurrencyResolverIgnoresSpoofedHeaders(t *testing.T) {
	resolver, err := NewCurrencyResolver(CurrencySourceExplicit, "US")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/", nil)
	request.Header.Set("CF-IPCountry", "CN")
	request.Header.Set("X-Geo-Country", "CN")

	if got := resolver.CurrencyFor(request); got != USD {
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

	if got := resolver.CurrencyFor(request); got != CNY {
		t.Fatalf("currency = %q, want CNY", got)
	}
}

func TestExplicitCurrencyResolverRequiresValidRegion(t *testing.T) {
	if _, err := NewCurrencyResolver(CurrencySourceExplicit, "mars"); err == nil {
		t.Fatal("error = nil, want invalid region")
	}
}
