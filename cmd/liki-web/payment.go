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

// paymentWiring owns the provider-neutral payment lifecycle and its local
// transactional stores. Provider clients stay outside this boundary.
type paymentWiring struct {
	AipayStore            *aipay.Store
	AipayHandler          *aipay.Handler
	Intents               *payment.IntentStore
	Notifications         *payment.NotificationStore
	Runtime               *payment.Runtime
	SubscriptionProcessor *sales.SubscriptionEventProcessor
	CompensationNotifier  func(context.Context, auth.User, entitlement.Entitlement, string) error
}

type paymentStores struct {
	Platform      *sql.DB
	Aipay         *sql.DB
	Orders        *sales.OrderStore
	Subscriptions *sales.SubscriptionStore
	Entitlements  *entitlement.Store
	Partners      *partner.Store
}

type paymentClients struct {
	Mailer *email.Client
	Dodo   *dodo.Client
	Xunhu  *xunhu.Client
}

type paymentPersistence struct {
	Intents       *payment.IntentStore
	Notifications *payment.NotificationStore
	Fulfillments  *payment.PaymentCompletionTaskStore
	Events        *payment.PaymentEventStore
}

type paymentRuntimeStores struct {
	Commerce    paymentStores
	Persistence paymentPersistence
}

func newPaymentWiring(
	ctx context.Context,
	config Config,
	stores paymentStores,
	clients paymentClients,
) (*paymentWiring, error) {
	wiring := &paymentWiring{}
	wiring.AipayStore = aipay.NewStore(stores.Aipay)
	if err := wiring.AipayStore.Migrate(ctx); err != nil {
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
	handler, err := aipay.NewHandler(wiring.AipayStore, gateway, config.AlipayConfig)
	if err != nil {
		return nil, fmt.Errorf("init aipay handler: %w", err)
	}
	wiring.AipayHandler = handler

	persistence := paymentPersistence{
		Intents:       payment.NewIntentStore(stores.Platform),
		Notifications: payment.NewNotificationStore(stores.Platform),
		Fulfillments:  payment.NewPaymentCompletionTaskStore(stores.Platform),
		Events:        payment.NewPaymentEventStore(stores.Platform),
	}
	for _, migration := range []interface{ Migrate(context.Context) error }{
		persistence.Intents, persistence.Notifications,
		persistence.Fulfillments, persistence.Events,
	} {
		if err := migration.Migrate(ctx); err != nil {
			return nil, err
		}
	}
	runtime, processor, err := buildPaymentRuntime(ctx, config, paymentRuntimeStores{
		Commerce:    stores,
		Persistence: persistence,
	}, clients)
	if err != nil {
		return nil, err
	}
	wiring.Intents = persistence.Intents
	wiring.Notifications = persistence.Notifications
	wiring.Runtime = runtime
	wiring.SubscriptionProcessor = processor
	wiring.CompensationNotifier = newCompensationNotifier(persistence.Notifications, config.PublicBaseURL)
	return wiring, nil
}

func buildPaymentRuntime(
	ctx context.Context,
	config Config,
	stores paymentRuntimeStores,
	clients paymentClients,
) (*payment.Runtime, *sales.SubscriptionEventProcessor, error) {
	commerce := stores.Commerce
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
		gateways[payment.ProviderDodo] = clients.Dodo
	} else {
		slog.Warn("dodo payment provider is not fully configured; it is disabled")
	}
	if xunhuConfigured {
		gateways[payment.ProviderXunhu] = clients.Xunhu
	} else {
		slog.Warn("xunhu payment provider is not fully configured; it is disabled")
	}
	chatDuration := time.Duration(config.ChatOneTimeAccessDays) * 24 * time.Hour
	flows := map[string]payment.ProductFlow{
		string(product.Chat): sales.NewProductPaymentFlow(
			commerce.Orders, commerce.Entitlements, product.Chat, chatDuration,
		),
	}
	runtime := payment.NewRuntime(gateways, flows, stores.Persistence.Intents,
		stores.Persistence.Notifications, stores.Persistence.Fulfillments,
		stores.Persistence.Events, paymentEmailSender{client: clients.Mailer},
		payment.RuntimeOptions{
			BaseURL:    config.PublicBaseURL,
			AdminEmail: config.AdminEmail,
		})
	runtime.SetPaymentCompletionObserver(func(ctx context.Context, intent payment.PaymentIntent) {
		sale := commissionablePaymentFromIntent(intent)
		if _, _, err := commerce.Partners.EnqueueCommissionPayment(ctx, sale); err != nil {
			slog.Error("referral: enqueue fulfilled payment", "payment_id", sale.ID, "err", err)
		}
	})
	runtime.SetRefundObserver(func(ctx context.Context, event payment.WebhookEvent, intent payment.PaymentIntent) error {
		return commerce.Partners.EnqueueCommissionRefund(ctx, event, commissionablePaymentFromIntent(intent))
	})
	processor := sales.NewSubscriptionEventProcessor(
		commerce.Subscriptions,
		commerce.Entitlements,
		config.DodoProductLikiChatSubscription,
		func(ctx context.Context, sale payment.CommissionablePayment) error {
			_, _, err := commerce.Partners.EnqueueCommissionPayment(ctx, sale)
			return err
		},
	)
	processor.SetCommissionRefundObserver(func(ctx context.Context, event payment.WebhookEvent, sale payment.CommissionablePayment) error {
		return commerce.Partners.EnqueueCommissionRefund(ctx, event, sale)
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
