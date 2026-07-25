package main

import (
	"context"
	"fmt"
	"log/slog"

	"liki-web/internal/email"
	"liki-web/internal/hepan"
	"liki-web/internal/mingshu"
	"liki-web/internal/naming"
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

func mingshuOnReportReady(store *mingshu.Store, sender email.Sender, url string) product.OnReportReady {
	return func(ctx context.Context, orderID string) {
		order, err := store.GetOrder(ctx, orderID)
		if err != nil || order == nil || order.Email == "" {
			return
		}
		link := fmt.Sprintf("%s/mingshu?order_id=%s", url, orderID)
		body := fmt.Sprintf(`<p>你的生命之书报告已生成。<a href="%s">点击查看</a></p>`, link)
		if err := sender.SendReport(ctx, order.Email, "报告已生成 — 生命之书 | liki.hk", body); err != nil {
			slog.Error("mail: mingshu report", "err", err)
		}
	}
}

func hepanOnReportReady(store *hepan.Store, sender email.Sender, url string) product.OnReportReady {
	return func(ctx context.Context, orderID string) {
		order, err := store.GetOrder(ctx, orderID)
		if err != nil || order == nil || order.Email == "" {
			return
		}
		link := fmt.Sprintf("%s/hepan?order_id=%s", url, orderID)
		body := fmt.Sprintf(`<p>你的合盘报告已生成。<a href="%s">点击查看</a></p>`, link)
		if err := sender.SendReport(ctx, order.Email, "报告已生成 — 合盘 | liki.hk", body); err != nil {
			slog.Error("mail: hepan report", "err", err)
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
