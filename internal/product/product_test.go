package product

import "testing"

func TestAmountMinor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		product  Product
		currency Currency
		want     int64
		wantErr  bool
	}{
		{name: "naming CNY", product: ProductNaming, currency: CNY, want: 990},
		{name: "naming USD", product: ProductNaming, currency: USD, want: 990},
		{name: "mingshu CNY", product: ProductMingShu, currency: CNY, want: 990},
		{name: "mingshu USD", product: ProductMingShu, currency: USD, want: 990},
		{name: "hepan CNY", product: ProductHePan, currency: CNY, want: 990},
		{name: "hepan USD", product: ProductHePan, currency: USD, want: 990},
		{name: "historical lifebook", product: "lifebook", currency: CNY, want: 990},
		{name: "unsupported currency", product: ProductNaming, currency: "EUR", wantErr: true},
		{name: "unknown product", product: "unknown", currency: CNY, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AmountMinor(tt.product, tt.currency)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AmountMinor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("AmountMinor() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNormalizeProduct(t *testing.T) {
	t.Parallel()
	if got := NormalizeProduct("lifebook"); got != ProductMingShu {
		t.Fatalf("NormalizeProduct(lifebook) = %q, want %q", got, ProductMingShu)
	}
	if !IsValidProduct(ProductHePan) || !IsValidProduct(ProductMingShu) {
		t.Fatal("canonical products should be valid")
	}
	if IsValidProduct("unknown") {
		t.Fatal("unknown product should be invalid")
	}
}

func TestDetectCurrencyPrefersCloudflareCountry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		cfCountry       string
		fallbackCountry string
		want            Currency
	}{
		{name: "Cloudflare CN", cfCountry: "CN", fallbackCountry: "US", want: CNY},
		{name: "Cloudflare US", cfCountry: "US", fallbackCountry: "CN", want: USD},
		{name: "fallback CN", cfCountry: "", fallbackCountry: "CN", want: CNY},
		{name: "fallback US", cfCountry: "", fallbackCountry: "US", want: USD},
		{name: "default CNY", cfCountry: "", fallbackCountry: "", want: CNY},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectCurrency(tt.cfCountry, tt.fallbackCountry); got != tt.want {
				t.Fatalf("DetectCurrency(%q, %q) = %q, want %q",
					tt.cfCountry, tt.fallbackCountry, got, tt.want)
			}
		})
	}
}

func TestNewOrderID(t *testing.T) {
	t.Parallel()
	ids := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id := NewOrderID()
		if id == "" {
			t.Fatal("NewOrderID() returned empty string")
		}
		if ids[id] {
			t.Fatalf("duplicate UUID: %s", id)
		}
		ids[id] = true
	}
}
