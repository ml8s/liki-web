package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/ml8s/liki-web/internal/account"
	"github.com/ml8s/liki-web/internal/adminauth"
	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/birthprofile"
	"github.com/ml8s/liki-web/internal/chat"
	"github.com/ml8s/liki-web/internal/dodo"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/feedback"
	apphttp "github.com/ml8s/liki-web/internal/http"
	"github.com/ml8s/liki-web/internal/metrics"
	"github.com/ml8s/liki-web/internal/partner"
	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/records"
	"github.com/ml8s/liki-web/internal/sales"
)

// httpWiring is the explicit composition input for the HTTP process. It is
// populated only in main; domain packages never reach back into it.
type httpWiring struct {
	config          Config
	platformDB      *sql.DB
	profilesDB      *sql.DB
	feedbackDB      *sql.DB
	likiDB          *sql.DB
	aipayDB         *sql.DB
	authStore       *auth.Store
	authConfig      auth.Config
	authHandler     *auth.Handler
	profiles        *birthprofile.Store
	feedback        *feedback.Store
	chats           *chatWiring
	payments        *paymentWiring
	partners        *partner.Store
	partnerHandler  *partner.Handler
	entitlements    *entitlement.Store
	orders          *sales.OrderStore
	subscriptions   *sales.SubscriptionStore
	currency        sales.CurrencyResolver
	catalog         *product.Catalog
	dodo            *dodo.Client
	erasureQueue    *account.ErasureStore
	partnerNotifier partner.DecisionNotifier
	adminAuthorizer *adminauth.Store
	rateLimiter     *apphttp.RateLimiter
	verifyLimiter   *apphttp.RateLimiter
}

func newHTTPHandler(wiring httpWiring) (http.Handler, *account.Handler) {
	readiness := func(ctx context.Context) error {
		for name, db := range map[string]*sql.DB{
			"platform": wiring.platformDB,
			"profiles": wiring.profilesDB,
			"feedback": wiring.feedbackDB,
			"liki":     wiring.likiDB,
			"aipay":    wiring.aipayDB,
		} {
			if err := db.PingContext(ctx); err != nil {
				return fmt.Errorf("database %s: %w", name, err)
			}
		}
		for _, expert := range chat.ExpertOrder {
			if err := checkAgentReadiness(ctx, agentOpsURL(wiring.config.AgentURLs[expert], wiring.config.AgentsOpsPort)); err != nil {
				return fmt.Errorf("expert %s: %w", expert, err)
			}
		}
		return nil
	}

	accountHandler := account.NewHandler(
		wiring.profiles,
		wiring.chats.Store,
		wiring.authStore,
		wiring.orders,
		wiring.entitlements,
		wiring.payments.Intents,
		wiring.payments.Notifications,
		wiring.partners,
		wiring.adminAuthorizer,
		wiring.erasureQueue,
		wiring.authHandler.RequireUser,
	)
	accountHandler.SetSubscriptionEraser(wiring.subscriptions)
	accountHandler.SetToolRecordEraser(wiring.chats.Records)
	accountHandler.SetRunRecordEraser(wiring.chats.RunRecords)

	var subscriptionCheckout sales.SubscriptionCheckoutCreator
	if wiring.config.DodoProductLikiChatSubscription != "" {
		subscriptionCheckout = wiring.dodo
	}

	mux := http.NewServeMux()
	handler := apphttp.RegisterRoutes(mux, apphttp.ServerDeps{
		PaymentRuntime: wiring.payments.Runtime,
		AuthHandler:    wiring.authHandler,
		PartnerHandler: wiring.partnerHandler,
		PartnerStore:   wiring.partners,
		SalesHandler: sales.NewHandler(
			wiring.orders,
			wiring.payments.Runtime,
			wiring.authHandler.RequireUser,
			func(next http.HandlerFunc) http.HandlerFunc {
				return wiring.rateLimiter.Wrap(5.0/60, 6, next)
			},
			sales.WithCurrencyResolver(wiring.currency),
			sales.WithCatalog(wiring.catalog),
			sales.WithSkillAuthorizer(wiring.entitlements),
			sales.WithSubscriptions(
				wiring.subscriptions,
				subscriptionCheckout,
				wiring.dodo,
				wiring.config.PublicBaseURL,
			),
			sales.WithSubscriptionReconciliation(
				wiring.dodo,
				wiring.payments.SubscriptionProcessor.HandleSubscriptionEvent,
			),
		),
		EntitlementHandler: entitlement.NewHandler(wiring.entitlements, wiring.authHandler.RequireUser),
		ChatHandler: chat.NewHandler(wiring.chats.Service, wiring.authHandler.RequireUser,
			func(next http.HandlerFunc) http.HandlerFunc {
				return wiring.rateLimiter.Wrap(5.0/60, 6, next)
			}),
		RecordsHandler: records.NewHandler(wiring.chats.Records, wiring.authHandler.RequireUser,
			func(next http.HandlerFunc) http.HandlerFunc {
				return wiring.rateLimiter.Wrap(20.0/60, 10, next)
			}),
		AipayHandler:         wiring.payments.AipayHandler,
		ProfileHandler:       birthprofile.NewHandler(wiring.profiles, wiring.authHandler.RequireUser),
		AccountHandler:       accountHandler,
		Entitlements:         wiring.entitlements,
		FeedbackStore:        wiring.feedback,
		FeedbackAPIKey:       wiring.config.FeedbackAPIKey,
		AdminEmail:           wiring.config.AdminEmail,
		AdminAuthorizer:      wiring.adminAuthorizer,
		AuthUsers:            wiring.authStore,
		SessionCookieName:    wiring.authConfig.CookieName,
		RunCallbackHandler:   wiring.chats.RunCallbackHandler,
		SalesStores:          sales.NewSalesStores(wiring.orders, wiring.subscriptions),
		CompensationNotifier: wiring.payments.CompensationNotifier,
		PartnerNotifier:      wiring.partnerNotifier,
		MetricsHandler:       metrics.Handler(wiring.config.MetricsToken),
		Readiness:            readiness,
		DevMode:              wiring.config.DevelopmentMode,
	}, BuildTime, Version, wiring.rateLimiter, wiring.verifyLimiter)
	return handler, accountHandler
}
