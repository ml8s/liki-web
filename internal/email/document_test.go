package email

import "testing"

func TestHomepageURLRejectsRelativeConfiguration(t *testing.T) {
	if got := HomepageURL("/liki.hk"); got != DefaultHomepageURL {
		t.Fatalf("HomepageURL() = %q", got)
	}
	if got := HomepageURL("https://example.com/"); got != "https://example.com" {
		t.Fatalf("HomepageURL() = %q", got)
	}
}

func TestDocumentLocalizesFooter(t *testing.T) {
	if got := Document("zh-CN", "https://example.com", "<p>x</p>"); got != "<p>x</p><p class=\"liki-email-footer\">前往 <a href=\"https://example.com\" target=\"_blank\" rel=\"noopener noreferrer\">https://example.com</a> 使用 Liki。</p>" {
		t.Fatalf("unexpected document: %q", got)
	}
}
