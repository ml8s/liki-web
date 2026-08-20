package hepan

import (
	"strings"

	"liki-web/internal/product"
)

const ProductHepan product.Product = "hepan"

func hepanAmount(curr string) (int, string) {
	switch strings.ToUpper(curr) {
	case "USD":
		return 990, "USD"
	default:
		return 990, "CNY"
	}
}
