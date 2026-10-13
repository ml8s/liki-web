package email

import (
	"fmt"
	"html"
	"strings"
	"time"
)

// Message is a provider-neutral transactional email. Queue adapters convert it
// to their storage type; presentation remains in this package so locale and
// brand rules cannot drift between send paths.
type Message struct {
	To      string
	Subject string
	Body    string
}

// BuildCompensationEmail renders the support-compensation customer notice.
// Templates are intentionally support-specific: they describe granted access,
// not payment or order fulfillment.
func BuildCompensationEmail(locale, template string, startsAt, expiresAt time.Time, publicBaseURL string) (Message, error) {
	locale = NormalizeLocale(locale)
	template = strings.TrimSpace(template)
	days := 7
	switch template {
	case "support_7d":
	case "support_30d":
		days = 30
	default:
		return Message{}, fmt.Errorf("email: invalid compensation template %q", template)
	}
	if startsAt.IsZero() || expiresAt.IsZero() {
		return Message{}, fmt.Errorf("email: compensation validity is required")
	}

	var subject, opening, validityLabel, action string
	switch locale {
	case "en":
		subject = "Your Liki Chat compensation has been added"
		opening = fmt.Sprintf("We have added %d days of Liki Chat access to your account.", days)
		validityLabel = "Valid: "
		action = "Sign in to Liki Chat to continue."
	case "zh-Hans":
		subject = "Liki 补偿权益已发放"
		opening = fmt.Sprintf("我们已为你的账号发放 %d 天 Liki Chat 使用权益。", days)
		validityLabel = "有效期："
		action = "登录 Liki Chat 即可继续使用。"
	default:
		subject = "Liki 補償權益已發放"
		opening = fmt.Sprintf("我們已為你的帳號發放 %d 天 Liki Chat 使用權益。", days)
		validityLabel = "有效期："
		action = "登入 Liki Chat 即可繼續使用。"
	}

	// These timestamps come from database rows, but escaping keeps the template
	// safe if the storage format later becomes operator-editable.
	validity := html.EscapeString(startsAt.UTC().Format("2006-01-02 15:04 MST") + " — " +
		expiresAt.UTC().Format("2006-01-02 15:04 MST"))
	content := `<p>` + opening + `</p><p>` + validityLabel + validity + `</p><p>` + action + `</p>`
	return Message{
		Subject: subject,
		Body:    Document(locale, publicBaseURL, content),
	}, nil
}
