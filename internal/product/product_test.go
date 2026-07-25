package product

import "testing"

func TestProductNaming_EmailSubject(t *testing.T) {
	t.Parallel()
	if got := ProductNaming.EmailSubject(); got != "服务已开通" {
		t.Errorf("EmailSubject() = %q, want 服务已开通", got)
	}
}

func TestProduct_UnknownEmailSubject(t *testing.T) {
	t.Parallel()
	p := Product("unknown")
	if got := p.EmailSubject(); got != "服务已开通" {
		t.Errorf("EmailSubject() = %q, want 服务已开通", got)
	}
}

func TestNamingAmountCents(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		c    Currency
		want int
	}{
		{"CNY", CNY, 19900},
		{"USD", USD, 2990},
		{"unknown", Currency("EUR"), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NamingAmountCents(tt.c); got != tt.want {
				t.Errorf("NamingAmountCents(%s) = %d, want %d", tt.c, got, tt.want)
			}
		})
	}
}

func TestNewOrderID(t *testing.T) {
	t.Parallel()
	id := NewOrderID()
	if id == "" {
		t.Error("NewOrderID() returned empty string")
	}
}

func TestNewOrderID_Unique(t *testing.T) {
	t.Parallel()
	ids := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id := NewOrderID()
		if ids[id] {
			t.Fatalf("duplicate UUID: %s", id)
		}
		ids[id] = true
	}
}

