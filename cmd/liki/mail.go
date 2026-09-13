package main

import (
	"context"
	"fmt"
	"log/slog"

	"liki-web/internal/email"
	"liki-web/internal/naming"
	"liki-web/internal/payment"
	"liki-web/internal/product"
)

func namingOnPaid(store *naming.Store, sender email.Sender, url string) product.OnPaid {
	return func(ctx context.Context, orderID string) {
		order, err := store.GetOrder(ctx, orderID)
		if err != nil || order == nil || order.Email == "" {
			return
		}
		link := fmt.Sprintf("%s/chat?order_id=%s", url, orderID)
		body := fmt.Sprintf(`<p>服务已开通！<a href="%s">点击开始起名</a></p><p>15天有效期，输入邮箱即可继续。</p>`, link)
		if err := sender.SendReport(ctx, order.Email, "订单确认 — 起名 | liki.hk", body); err != nil {
			slog.Error("mail: naming onPaid", "err", err)
		}
	}
}

func adminVerification(sender email.Sender) func(ctx context.Context, email, code string) {
	return func(ctx context.Context, email, code string) {
		body := fmt.Sprintf(`<p>管理员登录验证码：</p><p style="font-size:24px;font-weight:bold;letter-spacing:4px">%s</p><p>5分钟内有效。</p>`, code)
		if err := sender.SendReport(ctx, email, "验证码 — liki.hk 管理后台", body); err != nil {
			slog.Error("mail: admin verification", "err", err)
		}
	}
}

type paymentEmailSender struct {
	client *email.Client
}

func (s paymentEmailSender) Send(ctx context.Context, msg payment.EmailMessage) error {
	return s.client.SendReport(ctx, msg.To, msg.Subject, msg.Body)
}
