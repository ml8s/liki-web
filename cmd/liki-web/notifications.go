package main

import (
	"context"
	"html"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/email"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/partner"
	"github.com/ml8s/liki-web/internal/payment"
)

func newAuthMailer(client *email.Client, baseURL string) auth.MailerFunc {
	return func(ctx context.Context, recipient, code, locale string) error {
		subject, body := "Liki login code", `<p>Your Liki login code:</p><p style="font-size:28px;font-weight:bold;letter-spacing:6px">`+code+`</p><p>The code is valid for 10 minutes.</p>`
		switch locale {
		case "zh-Hant":
			subject, body = "Liki 登入驗證碼", `<p>你的 Liki 登入驗證碼：</p><p style="font-size:28px;font-weight:bold;letter-spacing:6px">`+code+`</p><p>驗證碼 10 分鐘內有效。</p>`
		case "zh-Hans", "":
			subject, body = "Liki 登录验证码", `<p>你的 Liki 登录验证码：</p><p style="font-size:28px;font-weight:bold;letter-spacing:6px">`+code+`</p><p>验证码 10 分钟内有效。</p>`
		}
		return client.SendReport(ctx, recipient, subject, email.Document(locale, baseURL, body))
	}
}

func newPartnerDecisionNotifier(
	client *email.Client,
	users *auth.Store,
	baseURL string,
) func(context.Context, partner.PartnerApplication, bool, string) error {
	return func(ctx context.Context, item partner.PartnerApplication, approved bool, reason string) error {
		applicant, err := users.GetUser(ctx, item.UserID)
		if err != nil {
			return err
		}
		role := "KOL"
		if item.Type == partner.RoleScout {
			role = "Scout"
		}
		var subject, opening, noteLabel, next string
		switch applicant.Locale {
		case "en":
			noteLabel, next = "Reviewer note: ", "You may submit a new application with additional details."
			if approved {
				subject, opening, next =
					"Your Liki partner application has been approved",
					"Your "+role+" application was approved.",
					"Sign in to Liki to open your partner workspace."
			} else {
				subject, opening = "Your Liki partner application decision", "Your "+role+" application was not approved."
			}
		case "zh-Hant":
			subject, noteLabel, next = "Liki 合作申請審核結果", "審核意見：", "你可以補充材料後重新提交申請。"
			if approved {
				opening, next = "你的 "+role+" 合作申請已通過。", "登入 Liki 後即可進入對應合作工作台。"
			} else {
				opening = "你的 " + role + " 合作申請未通過。"
			}
		}
		if applicant.Locale != "en" && applicant.Locale != "zh-Hant" {
			subject, noteLabel, next = "Liki 合作申请审核结果", "审核意见：", "你可以补充材料后重新提交申请。"
			if approved {
				opening, next = "你的 "+role+" 合作申请已通过。", "登录 Liki 后即可进入对应合作工作台。"
			} else {
				opening = "你的 " + role + " 合作申请未通過。"
			}
		}
		body := `<p>` + opening + `</p><p>` + noteLabel + html.EscapeString(reason) + `</p><p>` + next + `</p>`
		return client.SendReport(ctx, applicant.Email, subject, email.Document(applicant.Locale, baseURL, body))
	}
}

func newCompensationNotifier(
	notifications *payment.NotificationStore,
	baseURL string,
) func(context.Context, auth.User, entitlement.Entitlement, string) error {
	return func(ctx context.Context, user auth.User, grant entitlement.Entitlement, template string) error {
		message, err := email.BuildCompensationEmail(
			user.Locale, template, grant.StartsAt, grant.ExpiresAt, baseURL,
		)
		if err != nil {
			return err
		}
		return notifications.EnqueueOnce(ctx, "entitlement:"+grant.ID,
			payment.NotificationEntitlementCompensation, payment.EmailMessage{
				To:      user.Email,
				Subject: message.Subject,
				Body:    message.Body,
			})
	}
}
