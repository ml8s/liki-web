// Package money is the single converter between fixed 2-decimal amount
// strings (provider wire format) and integer minor units (canonical
// storage). Parsing is strict: whitespace is trimmed, but empty, negative,
// zero, or >2-decimal-place values are rejected — on a money path a
// protocol violation must fail loudly, never silently lose precision.
package money

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Amount is the canonical monetary value. Amounts are always integer minor
// units so provider formatting never becomes domain state.
type Amount struct {
	Currency   string
	MinorUnits int64
}

// ParseDecimalToMinor converts a fixed 2-decimal amount string ("1",
// "1.5", "1.00") into integer minor units without float rounding.
func ParseDecimalToMinor(value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, errors.New("money: empty amount")
	}
	parts := strings.SplitN(trimmed, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole < 0 {
		return 0, fmt.Errorf("money: invalid amount %q", value)
	}
	minor := whole * 100
	if len(parts) == 2 {
		frac := parts[1]
		if len(frac) > 2 {
			return 0, fmt.Errorf("money: amount %q has more than 2 decimal places", value)
		}
		for len(frac) < 2 {
			frac += "0"
		}
		fraction, parseErr := strconv.ParseInt(frac, 10, 64)
		if parseErr != nil {
			return 0, fmt.Errorf("money: invalid amount %q", value)
		}
		minor += fraction
	}
	if minor <= 0 {
		return 0, fmt.Errorf("money: amount %q must be positive", value)
	}
	return minor, nil
}

// FormatDecimalFromMinor renders integer minor units as a fixed 2-decimal
// string (the provider wire format), handling negative values correctly.
func FormatDecimalFromMinor(minor int64) string {
	sign := ""
	if minor < 0 {
		sign = "-"
		minor = -minor
	}
	return fmt.Sprintf("%s%d.%02d", sign, minor/100, minor%100)
}

// FormatDisplay renders minor units for human-facing surfaces that do not
// do their own locale formatting (emails, notifications): CNY/USD get
// their conventional symbols, other currencies the ISO code.
func FormatDisplay(minor int64, currency string) string {
	units := minor / 100
	cents := minor % 100
	switch currency {
	case "CNY":
		return fmt.Sprintf("¥%d.%02d", units, cents)
	case "USD":
		return fmt.Sprintf("$%d.%02d", units, cents)
	default:
		return fmt.Sprintf("%s %d.%02d", currency, units, cents)
	}
}
