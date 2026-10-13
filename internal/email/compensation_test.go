package email

import (
	"strings"
	"testing"
	"time"
)

func TestBuildCompensationEmailTemplates(t *testing.T) {
	starts := time.Date(2026, 10, 14, 8, 30, 0, 0, time.UTC)
	expires := starts.Add(30 * 24 * time.Hour)
	tests := []struct {
		locale        string
		subject       string
		opening       string
		validityLabel string
		action        string
	}{
		{locale: "zh-CN", subject: "Liki 补偿权益已发放", opening: "我们已为你的账号发放 30 天 Liki Chat 使用权益。", validityLabel: "有效期：", action: "登录 Liki Chat 即可继续使用。"},
		{locale: "zh-Hant", subject: "Liki 補償權益已發放", opening: "我們已為你的帳號發放 30 天 Liki Chat 使用權益。", validityLabel: "有效期：", action: "登入 Liki Chat 即可繼續使用。"},
		{locale: "en", subject: "Your Liki Chat compensation has been added", opening: "We have added 30 days of Liki Chat access to your account.", validityLabel: "Valid: ", action: "Sign in to Liki Chat to continue."},
	}

	for _, test := range tests {
		footer := "使用 Liki"
		if test.locale == "en" {
			footer = "to use Liki"
		}
		message, err := BuildCompensationEmail(test.locale, "support_30d", starts, expires, "https://example.com/")
		if err != nil {
			t.Fatalf("%s: %v", test.locale, err)
		}
		for _, want := range []string{
			test.opening, test.validityLabel,
			"2026-10-14 08:30 UTC", "2026-11-13 08:30 UTC", test.action,
			`href="https://example.com"`, footer,
		} {
			if !strings.Contains(message.Body, want) {
				t.Fatalf("%s body %q missing %q", test.locale, message.Body, want)
			}
		}
		if message.Subject != test.subject {
			t.Fatalf("%s subject=%q", test.locale, message.Subject)
		}
	}
}

func TestBuildCompensationEmailSevenDaysAndValidation(t *testing.T) {
	starts := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	message, err := BuildCompensationEmail("en", "support_7d", starts, starts.Add(7*24*time.Hour), "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(message.Body, "7 days") {
		t.Fatalf("body=%q", message.Body)
	}
	if HomepageURL("") != DefaultHomepageURL || !strings.Contains(message.Body, DefaultHomepageURL) {
		t.Fatalf("body=%q", message.Body)
	}
	if _, err := BuildCompensationEmail("en", "support_999d", starts, starts, ""); err == nil {
		t.Fatal("invalid template accepted")
	}
	if _, err := BuildCompensationEmail("en", "support_7d", time.Time{}, starts, ""); err == nil {
		t.Fatal("missing validity accepted")
	}
}
