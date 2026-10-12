package i18n

import (
	"testing"
)

func TestLangConstants(t *testing.T) {
	t.Parallel()
	if string(LangHans) != "zh-Hans" {
		t.Errorf("LangHans = %q", LangHans)
	}
	if string(LangHant) != "zh-Hant" {
		t.Errorf("LangHant = %q", LangHant)
	}
	if string(EN) != "en" {
		t.Errorf("EN = %q", EN)
	}
}
