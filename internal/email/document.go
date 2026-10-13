package email

import (
	"net/url"
	"strings"
)

// DefaultHomepageURL is the fallback brand destination for transactional
// email. Production configuration normally supplies LIKI_PUBLIC_BASE_URL.
const DefaultHomepageURL = "https://liki.hk"

// NormalizeLocale maps synonyms used by auth/payment events onto the three
// supported transactional locales.
func NormalizeLocale(locale string) string {
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case "en", "en-us", "en-gb":
		return "en"
	case "zh-hans", "zh-cn", "zh-sg":
		return "zh-Hans"
	default:
		return "zh-Hant"
	}
}

// HomepageURL resolves the brand home for an email. An absolute configured
// public URL wins; the fallback keeps emails useful in misconfigured test
// environments without ever emitting a relative href.
func HomepageURL(publicBaseURL string) string {
	value := strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	if value == "" {
		return DefaultHomepageURL
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" && parsed.Scheme != "http" || parsed.Host == "" {
		return DefaultHomepageURL
	}
	return parsed.String()
}

// Document appends the shared brand footer to transactional email content.
// Transactional bodies intentionally remain small HTML fragments; adding a full
// HTML shell here would change the contract of every existing sender.
func Document(locale, publicBaseURL, content string) string {
	homepage := HomepageURL(publicBaseURL)
	switch NormalizeLocale(locale) {
	case "en":
		return content + `<p class="liki-email-footer">Visit <a href="` + homepage + `" target="_blank" rel="noopener noreferrer">` + homepage + `</a> to use Liki.</p>`
	case "zh-Hans":
		return content + `<p class="liki-email-footer">前往 <a href="` + homepage + `" target="_blank" rel="noopener noreferrer">` + homepage + `</a> 使用 Liki。</p>`
	default:
		return content + `<p class="liki-email-footer">前往 <a href="` + homepage + `" target="_blank" rel="noopener noreferrer">` + homepage + `</a> 使用 Liki。</p>`
	}
}
