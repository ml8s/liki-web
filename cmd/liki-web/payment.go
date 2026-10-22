package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/ml8s/liki-web/internal/aipay"
	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/dodo"
	"github.com/ml8s/liki-web/internal/email"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/partner"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sales"
	"github.com/ml8s/liki-web/internal/xunhu"
)

// paymentSystem owns the provider-neutral payment lifecycle and its local
// transactional stores. Provider clients stay outside this boundary.
type paymentSystem struct {
	AipayStore            *aipay.Store
	AipayHandler          *aipay.Handler
	Intents               *payment.IntentStore
	Notifications         *payment.NotificationStore
	Runtime               *payment.Runtime
	SubscriptionProcessor *sales.SubscriptionEventProcessor
	CompensationNotifier  func(context.Context, auth.User, entitlement.Entitlement, string) error
}

func newPaymentSystem(
	ctx context.Context,
	config Config,
	platformDB *sql.DB,
	aipayDB *sql.DB,
	orders *sales.OrderStore,
	subscriptions *sales.SubscriptionStore,
	entitlements *entitlement.Store,
	partners *partner.Store,
	mailer *email.Client,
	dodoClient *dodo.Client,
	xunhuClient *xunhu.Client,
) (*paymentSystem, error) {
	system := &paymentSystem{}
	system.AipayStore = aipay.NewStore(aipayDB)
	if err := system.AipayStore.Migrate(ctx); err != nil {
		return nil, fmt.Errorf("init aipay schema: %w", err)
	}
	var gateway aipay.Gateway
	if config.AlipayConfig.Enabled() {
		client, err := aipay.NewClient(config.AlipayConfig)
		if err != nil {
			return nil, fmt.Errorf("init alipay aipay client: %w", err)
		}
		gateway = client
	} else {
		slog.Warn("alipay aipay is not fully configured; it is disabled")
	}
	handler, err := aipay.NewHandler(system.AipayStore, gateway, config.AlipayConfig)
	if err != nil {
		return nil, fmt.Errorf("init aipay handler: %w", err)
	}
	system.AipayHandler = handler

	system.Intents = payment.NewIntentStore(platformDB)
	system.Notifications = payment.NewNotificationStore(platformDB)
	fulfillments := payment.NewPaymentCompletionTaskStore(platformDB)
	events := payment.NewPaymentEventStore(platformDB)
	if err := system.Intents.Migrate(ctx); err != nil {
		return nil, fmt.Errorf("init payment intents: %w", err)
	}
	if err := system.Notifications.Migrate(ctx); err != nil {
		return nil, fmt.Errorf("init payment notifications: %w", err)
	}
	if err := fulfillments.Migrate(ctx); err != nil {
		return nil, fmt.Errorf("init payment fulfillment tasks: %w", err)
	}
	if err := events.Migrate(ctx); err != nil {
		return nil, fmt.Errorf("init payment events: %w", err)
	}
	runtime, processor, err := buildPaymentRuntime(
		ctx, config, orders, subscriptions, entitlements, partners,
		system.Intents, system.Notifications, fulfillments, events,
		mailer, dodoClient, xunhuClient,
	)
	if err != nil {
		return nil, err
	}
	system.Runtime = runtime
	system.SubscriptionProcessor = processor
	system.CompensationNotifier = newCompensationNotifier(system.Notifications, config.PublicBaseURL)
	return system, nil
}

func buildPaymentRuntime(
	ctx context.Context,
	config Config,
	orders *sales.OrderStore,
	subscriptions *sales.SubscriptionStore,
	entitlements *entitlement.Store,
	partners *partner.Store,
	intents *payment.IntentStore,
	notifications *payment.NotificationStore,
	fulfillments *payment.PaymentCompletionTaskStore,
	events *payment.PaymentEventStore,
	mailer *email.Client,
	dodoClient *dodo.Client,
	xunhuClient *xunhu.Client,
) (*payment.Runtime, *sales.SubscriptionEventProcessor, error) {
	dodoConfigured := config.DodoAPIKey != "" && config.DodoWebhookKey != ""
	if dodoConfigured && config.DodoProductLikiChatOneTime == "" {
		slog.Warn("dodo one-time product is not configured for liki-chat; USD chat checkout will fail")
	}
	if dodoConfigured && config.DodoProductLikiChatSubscription == "" {
		slog.Warn("dodo subscription product is not configured for liki-chat; the subscription offer stays disabled")
	}
	xunhuConfigured := config.XunhuAppID != "" && config.XunhuAppSecret != "" && config.PaymentWebhookURL != ""
	if dodoConfigured || xunhuConfigured {
		if err := validateHTTPURL("LIKI_PUBLIC_BASE_URL", config.PublicBaseURL); err != nil {
			return nil, nil, fmt.Errorf("payment runtime requires an absolute public LIKI_PUBLIC_BASE_URL: %w", err)
		}
	}
	if !dodoConfigured && !xunhuConfigured {
		return nil, nil, fmt.Errorf("no payment provider is fully configured")
	}
	gateways := map[payment.Provider]payment.Gateway{}
	if dodoConfigured {
		gateways[payment.ProviderDodo] = dodoClient
	} else {
		slog.Warn("dodo payment provider is not fully configured; it is disabled")
	}
	if xunhuConfigured {
		gateways[payment.ProviderXunhu] = xunhuClient
	} else {
		slog.Warn("xunhu payment provider is not fully configured; it is disabled")
	}
	chatDuration := time.Duration(config.ChatOneTimeAccessDays) * 24 * time.Hour
	flows := map[string]payment.ProductFlow{
		string(product.Chat): sales.NewProductPaymentFlow(orders, entitlements, product.Chat, chatDuration),
	}
	runtime := payment.NewRuntime(gateways, flows, intents, notifications, fulfillments, events,
		paymentEmailSender{client: mailer}, payment.RuntimeOptions{
			BaseURL:    config.PublicBaseURL,
			AdminEmail: config.AdminEmail,
		})
	runtime.SetPaymentCompletionObserver(func(ctx context.Context, intent payment.PaymentIntent) {
		sale := commissionablePaymentFromIntent(intent)
		if _, _, err := partners.EnqueueCommissionPayment(ctx, sale); err != nil {
			slog.Error("referral: enqueue fulfilled payment", "payment_id", sale.ID, "err", err)
		}
	})
	runtime.SetRefundObserver(func(ctx context.Context, event payment.WebhookEvent, intent payment.PaymentIntent) error {
		return partners.EnqueueCommissionRefund(ctx, event, commissionablePaymentFromIntent(intent))
	})
	processor := sales.NewSubscriptionEventProcessor(
		subscriptions,
		entitlements,
		config.DodoProductLikiChatSubscription,
		func(ctx context.Context, sale payment.CommissionablePayment) error {
			_, _, err := partners.EnqueueCommissionPayment(ctx, sale)
			return err
		},
	)
	processor.SetCommissionRefundObserver(func(ctx context.Context, event payment.WebhookEvent, sale payment.CommissionablePayment) error {
		return partners.EnqueueCommissionRefund(ctx, event, sale)
	})
	runtime.SetSubscriptionProcessor(processor)
	return runtime, processor, nil
}

func commissionablePaymentFromIntent(intent payment.PaymentIntent) payment.CommissionablePayment {
	return payment.CommissionablePayment{
		ID: intent.ID, BusinessOrderID: intent.BusinessOrderID, UserID: intent.UserID,
		Product: intent.Product, Provider: intent.Provider,
		AmountMinor: intent.AmountMinor, Currency: intent.Currency, Email: intent.Email,
	}
}
