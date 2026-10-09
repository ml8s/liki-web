package money

import "testing"

func TestParseDecimalToMinor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want int64
		err  bool
	}{
		{"1", 100, false},
		{"1.5", 150, false},
		{"1.00", 100, false},
		{"12.34", 1234, false},
		{" 1.00 ", 100, false},
		{"0.01", 1, false},
		{"1.005", 0, true}, // >2 位小数：拒绝而非截断
		{"1.000", 0, true}, // 同上（剥零形式）
		{"0", 0, true},     // 非正
		{"0.00", 0, true},  // 非正
		{"-1.00", 0, true}, // 负数
		{"", 0, true},      // 空
		{"abc", 0, true},   // 非法
	}
	for _, tc := range cases {
		got, err := ParseDecimalToMinor(tc.in)
		if tc.err {
			if err == nil {
				t.Errorf("ParseDecimalToMinor(%q) want error, got %d", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseDecimalToMinor(%q) unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseDecimalToMinor(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestFormatDecimalFromMinor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   int64
		want string
	}{
		{100, "1.00"},
		{150, "1.50"},
		{1, "0.01"},
		{1234, "12.34"},
		{-1, "-0.01"},
		{-150, "-1.50"},
	}
	for _, tc := range cases {
		if got := FormatDecimalFromMinor(tc.in); got != tc.want {
			t.Errorf("FormatDecimalFromMinor(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
