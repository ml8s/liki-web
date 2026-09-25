package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ml8s/liki-web/internal/email"
	"github.com/ml8s/liki-web/internal/naming"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
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

type paymentEmailSender struct {
	client *email.Client
}

func (s paymentEmailSender) Send(ctx context.Context, msg payment.EmailMessage) error {
	return s.client.SendReport(ctx, msg.To, msg.Subject, msg.Body)
}
