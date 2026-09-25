package http

import "github.com/ml8s/liki-web/internal/i18n"

// langToLocale maps frontend language code to BCP 47 locale.
// Short "zh" is a legacy alias for zh-Hant (Traditional Chinese).
func langToLocale(lang string) string {
	switch lang {
	case "zh", string(i18n.LangHant):
		return string(i18n.LangHant)
	case "zh-CN", string(i18n.LangHans):
		return string(i18n.LangHans)
	case string(i18n.EN):
		return string(i18n.EN)
	default:
		return string(i18n.LangHant)
	}
}
