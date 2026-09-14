package paymentflow

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"liki-web/internal/hepan"
	"liki-web/internal/httputil"
	"liki-web/internal/mingshu"
	"liki-web/internal/naming"
	"liki-web/internal/payment"
	"liki-web/internal/product"
)

type NamingFlow struct {
	store     *naming.Store
	validDays int
}

func NewNamingFlow(store *naming.Store, validDays int) *NamingFlow {
	return &NamingFlow{store: store, validDays: validDays}
}

func defaultLocale(locale string) string {
	if locale == "" {
		return "zh-Hant"
	}
	return locale
}

func (f *NamingFlow) Product() string { return string(product.ProductNaming) }

func (f *NamingFlow) PrepareCheckout(ctx context.Context, businessOrderID string) (payment.BusinessOrder, error) {
	order, err := f.store.GetOrder(ctx, businessOrderID)
	if err != nil {
		return payment.BusinessOrder{}, err
	}
	return payment.BusinessOrder{
		ID: order.OrderID, Product: string(product.ProductNaming),
		AmountMinor: int64(order.Amount), Currency: order.Currency,
		Email: order.Email, Locale: defaultLocale(order.Locale), Status: order.Status,
	}, nil
}

func (f *NamingFlow) OnPaid(ctx context.Context, intent payment.PaymentIntent) error {
	changed, err := f.store.MarkPaidIdempotent(ctx, intent.BusinessOrderID, intent.ProviderPaymentID)
	if err != nil {
		return fmt.Errorf("naming flow: mark paid: %w", err)
	}
	if !changed {
		return nil
	}
	expiresAt := time.Now().UTC().AddDate(0, 0, f.validDays).Format(time.RFC3339)
	if err := f.store.OverrideChatExpiresAt(ctx, intent.BusinessOrderID, expiresAt); err != nil {
		return fmt.Errorf("naming flow: set expiry: %w", err)
	}
	return nil
}

func (f *NamingFlow) SuccessRedirect(intent payment.PaymentIntent) string {
	return "/" + defaultLocale(intent.Locale) + "/naming/chat?order_id=" + intent.BusinessOrderID
}

func (f *NamingFlow) FailureRedirect(intent payment.PaymentIntent) string {
	return "/" + defaultLocale(intent.Locale) + "/naming?order_id=" + intent.BusinessOrderID
}

type MingShuFlow struct {
	store         *mingshu.Store
	generator     *mingshu.Generator
	notifications *payment.NotificationStore
	baseURL       string
}

func NewMingShuFlow(store *mingshu.Store, generator *mingshu.Generator, notifications *payment.NotificationStore, baseURL string) *MingShuFlow {
	return &MingShuFlow{store: store, generator: generator, notifications: notifications, baseURL: baseURL}
}

func (f *MingShuFlow) Product() string { return string(product.ProductMingShu) }

func (f *MingShuFlow) PrepareCheckout(ctx context.Context, businessOrderID string) (payment.BusinessOrder, error) {
	order, err := f.store.GetOrder(ctx, businessOrderID)
	if err != nil {
		return payment.BusinessOrder{}, err
	}
	return payment.BusinessOrder{
		ID: order.ID, Product: string(product.ProductMingShu),
		AmountMinor: int64(order.Amount), Currency: order.Currency,
		Email: order.Email, Locale: defaultLocale(order.Locale), Status: order.Status,
	}, nil
}

func (f *MingShuFlow) OnPaid(ctx context.Context, intent payment.PaymentIntent) error {
	if _, err := f.store.MarkPaidIdempotent(ctx, intent.BusinessOrderID, intent.ProviderPaymentID); err != nil {
		return fmt.Errorf("mingshu flow: mark paid: %w", err)
	}
	if f.generator == nil {
		return nil
	}
	if err := f.generator.Run(ctx, intent.BusinessOrderID); err != nil {
		return fmt.Errorf("mingshu flow: generate report: %w", err)
	}
	if f.notifications != nil {
		token, tokenErr := httputil.NewOrderAccessToken(string(f.Product()), intent.BusinessOrderID, time.Now(), 30*24*time.Hour)
		if tokenErr != nil {
			return tokenErr
		}
		link := f.baseURL + "/mingshu/report?order_id=" + intent.BusinessOrderID +
			"&access_token=" + url.QueryEscape(token)
		return f.notifications.EnqueueOnce(ctx, intent.ID, payment.NotificationReportReady, payment.EmailMessage{
			To: intent.Email, Subject: "报告已生成 — 生命书 | liki.hk",
			Body: `<p>你的生命书报告已生成。</p><p><a href="` + link + `">点击查看</a></p>`,
		})
	}
	return nil
}

func (f *MingShuFlow) SuccessRedirect(intent payment.PaymentIntent) string {
	return "/" + defaultLocale(intent.Locale) + "/mingshu/report?order_id=" + intent.BusinessOrderID
}

func (f *MingShuFlow) FailureRedirect(intent payment.PaymentIntent) string {
	return "/" + defaultLocale(intent.Locale) + "/mingshu?order_id=" + intent.BusinessOrderID
}

type HepanFlow struct {
	store         *hepan.Store
	generator     *hepan.Generator
	notifications *payment.NotificationStore
	baseURL       string
}

func NewHepanFlow(store *hepan.Store, generator *hepan.Generator, notifications *payment.NotificationStore, baseURL string) *HepanFlow {
	return &HepanFlow{store: store, generator: generator, notifications: notifications, baseURL: baseURL}
}

func (f *HepanFlow) Product() string { return string(product.ProductHePan) }

func (f *HepanFlow) PrepareCheckout(ctx context.Context, businessOrderID string) (payment.BusinessOrder, error) {
	order, err := f.store.GetOrder(ctx, businessOrderID)
	if err != nil {
		return payment.BusinessOrder{}, err
	}
	return payment.BusinessOrder{
		ID: businessOrderID, Product: f.Product(),
		AmountMinor: int64(order.Amount), Currency: order.Currency,
		Email: order.Email, Locale: defaultLocale(order.Locale), Status: order.Status,
	}, nil
}

func (f *HepanFlow) OnPaid(ctx context.Context, intent payment.PaymentIntent) error {
	if _, err := f.store.MarkPaidIdempotent(ctx, intent.BusinessOrderID, intent.ProviderPaymentID); err != nil {
		return fmt.Errorf("hepan flow: mark paid: %w", err)
	}
	if f.generator == nil {
		return nil
	}
	if err := f.generator.Run(ctx, intent.BusinessOrderID); err != nil {
		return fmt.Errorf("hepan flow: generate report: %w", err)
	}
	if f.notifications != nil {
		token, tokenErr := httputil.NewOrderAccessToken(string(f.Product()), intent.BusinessOrderID, time.Now(), 30*24*time.Hour)
		if tokenErr != nil {
			return tokenErr
		}
		link := f.baseURL + "/hepan/report?order_id=" + intent.BusinessOrderID +
			"&access_token=" + url.QueryEscape(token)
		return f.notifications.EnqueueOnce(ctx, intent.ID, payment.NotificationReportReady, payment.EmailMessage{
			To: intent.Email, Subject: "报告已生成 — 合盘 | liki.hk",
			Body: `<p>你的合盘报告已生成。</p><p><a href="` + link + `">点击查看</a></p>`,
		})
	}
	return nil
}

func (f *HepanFlow) SuccessRedirect(intent payment.PaymentIntent) string {
	return "/" + defaultLocale(intent.Locale) + "/hepan/report?order_id=" + intent.BusinessOrderID
}

func (f *HepanFlow) FailureRedirect(intent payment.PaymentIntent) string {
	return "/" + defaultLocale(intent.Locale) + "/hepan?order_id=" + intent.BusinessOrderID
}
