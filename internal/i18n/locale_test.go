package i18n

import "testing"

func TestNegotiateMarketingLocale(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   string
	}{
		{name: "simplified Chinese", accept: "zh-CN,zh;q=0.9", want: "zh-Hans"},
		{name: "Singapore Chinese", accept: "zh-SG,zh;q=0.9", want: "zh-Hans"},
		{name: "traditional Chinese", accept: "zh-TW,zh;q=0.9", want: "zh-Hant"},
		{name: "English", accept: "en-US,en;q=0.9", want: "en"},
		{name: "unknown defaults to traditional Chinese", accept: "ja-JP,ja;q=0.9", want: "zh-Hant"},
		{name: "empty defaults to traditional Chinese", accept: "", want: "zh-Hant"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NegotiateMarketingLocale(test.accept); got != test.want {
				t.Fatalf("NegotiateMarketingLocale(%q) = %q, want %q", test.accept, got, test.want)
			}
		})
	}
}
