package httputil

import (
	"net/http"
	"strings"
)

func LocaleFromAccept(r *http.Request) string {
	lang := strings.SplitN(r.Header.Get("Accept-Language"), ",", 2)[0]
	if strings.HasPrefix(lang, "zh-CN") || strings.HasPrefix(lang, "zh-Hans") {
		return "zh-Hans"
	}
	if strings.HasPrefix(lang, "zh") {
		return "zh-Hant"
	}
	return "en"
}
