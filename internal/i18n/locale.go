package i18n

import "strings"

// NegotiateMarketingLocale maps a browser Accept-Language value to one of the
// three public marketing locales. Unknown and absent locales intentionally
// fall back to zh-Hant everywhere, including referral landing pages.
func NegotiateMarketingLocale(acceptLanguage string) string {
	accept := strings.ToLower(strings.TrimSpace(acceptLanguage))
	switch {
	case strings.HasPrefix(accept, "zh-cn"),
		strings.HasPrefix(accept, "zh-sg"),
		strings.HasPrefix(accept, "zh-hans"):
		return "zh-Hans"
	case strings.HasPrefix(accept, "zh-tw"),
		strings.HasPrefix(accept, "zh-hk"),
		strings.HasPrefix(accept, "zh-mo"),
		strings.HasPrefix(accept, "zh-hant"):
		return "zh-Hant"
	case strings.HasPrefix(accept, "en"):
		return "en"
	default:
		return "zh-Hant"
	}
}
