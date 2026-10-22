package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ml8s/liki-web/internal/account"
	"github.com/ml8s/liki-web/internal/adminauth"
	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/birthprofile"
	"github.com/ml8s/liki-web/internal/dodo"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/partner"
	"github.com/ml8s/liki-web/internal/sales"
	"github.com/ml8s/liki-web/internal/xunhu"

	"github.com/ml8s/liki-web/internal/email"
	"github.com/ml8s/liki-web/internal/feedback"
	apphttp "github.com/ml8s/liki-web/internal/http"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/product"
)

// Build-time variables injected via -ldflags.
var (
	BuildTime = "dev"
	Version   = "dev"
)

func main() {
	cfg := LoadConfig()

	currencyResolver, err := sales.NewCurrencyResolver(cfg.CurrencySource, cfg.BillingRegion)
	if err != nil {
		slog.Error("invalid billing currency configuration", "err", err)
		os.Exit(1)
	}

	// Structured logging
	slog.SetDefault(slog.New(httputil.NewRequestIDHandler(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
	)))

	// Validate required config
	requireEnv("JWT_SECRET", cfg.JWTSecret)
	if err := validateRuntimeURLs(cfg); err != nil {
		slog.Error("invalid runtime URL configuration", "err", err)
		os.Exit(1)
	}
	if err := validateBusinessConfig(cfg); err != nil {
		slog.Error("invalid business configuration", "err", err)
		os.Exit(1)
	}

	// ── open databases ──
	dbs, err := openAppDatabases(cfg.DataDir)
	if err != nil {
		slog.Error("open databases", "err", err)
		os.Exit(1)
	}
	defer func() { _ = dbs.Close() }()

	platformDB, profilesDB, feedbackDB := dbs.Platform, dbs.Profiles, dbs.Feedback
	likiDB, aipayDB := dbs.Liki, dbs.Aipay

	// ── platform (auth + payment runtime) ──
	authConfig := auth.DefaultConfig()
	authConfig.SecureCookie = !cfg.DevelopmentMode
	authConfig.ExposeDevCode = cfg.DevelopmentMode
	authStore := auth.NewStore(platformDB, []byte(cfg.JWTSecret), authConfig)

	// ── profiles ──
	profileStore := birthprofile.NewStore(profilesDB)

	// ── feedback ──
	feedbackStore, err := feedback.NewStore(feedbackDB)
	if err != nil {
		slog.Error("init feedback store", "err", err)
		os.Exit(1)
	}

	// Init services
	emailClient := email.New(cfg.ResendAPIKey, cfg.ResendFrom)

	dodoProducts := map[string]string{
		string(product.Chat) + ":one_time":     cfg.DodoProductLikiChatOneTime,
		string(product.Chat) + ":subscription": cfg.DodoProductLikiChatSubscription,
	}
	dodoClient := dodo.New(cfg.DodoAPIKey, cfg.DodoWebhookKey, cfg.DodoTestMode, dodoProducts)

	xunhuClient := xunhu.New(cfg.XunhuAppID, cfg.XunhuAppSecret, cfg.PaymentWebhookURL)

	// Liki Chat owns exactly the one installable skill.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Deny-by-default tool scope so an active session can reach only the
	// deterministic engine domains exposed by the Liki Chat skill.
	if err := profileStore.Migrate(ctx); err != nil {
		slog.Error("init profile schema", "err", err)
		os.Exit(1)
	}

	if cfg.PublicBaseURL == "" {
		slog.Warn("LIKI_PUBLIC_BASE_URL not set — post-payment redirect will fail")
	}

	if err := authStore.Migrate(ctx); err != nil {
		slog.Error("init auth schema", "err", err)
		os.Exit(1)
	}

	adminAuthorizer, err := adminauth.Open(filepath.Join(dbs.dir, "platform.db"))
	if err != nil {
		slog.Error("init admin auth", "err", err)
		os.Exit(1)
	}
	defer func() { _ = adminAuthorizer.Close() }()
	if err := adminAuthorizer.Migrate(ctx); err != nil {
		slog.Error("init admin auth schema", "err", err)
		os.Exit(1)
	}
	if adminEmail := cfg.AdminEmail; adminEmail != "" {
		if err := adminAuthorizer.BootstrapAdminByEmail(ctx, adminEmail); err != nil {
			slog.Error("bootstrap admin", "err", err)
			os.Exit(1)
		}
	} else {
		slog.Warn("LIKI_ADMIN_EMAIL is empty; admin console capabilities are disabled")
	}

	authMailer := newAuthMailer(emailClient, cfg.PublicBaseURL)
	partnerDecisionEmail := newPartnerDecisionNotifier(emailClient, authStore, cfg.PublicBaseURL)
	authService, err := auth.NewService(authStore, authMailer, authConfig)
	if err != nil {
		slog.Error("init auth service", "err", err)
		os.Exit(1)
	}
	authHandler, err := auth.NewHandler(authStore, authService, authConfig)
	if err != nil {
		slog.Error("init auth handler", "err", err)
		os.Exit(1)
	}

	entitlements := entitlement.NewStore(likiDB)
	if err := entitlements.Migrate(ctx); err != nil {
		slog.Error("init entitlements", "err", err)
		os.Exit(1)
	}
	if _, err := entitlements.ExpireDue(ctx); err != nil {
		slog.Warn("expire entitlements", "err", err)
	}
	subscriptionStore := sales.NewSubscriptionStore(likiDB)
	if err := subscriptionStore.Migrate(ctx); err != nil {
		slog.Error("init subscription schema", "err", err)
		os.Exit(1)
	}
	payoutSecret := os.Getenv("LIKI_REFERRAL_PAYOUT_ENCRYPTION_KEY")
	if len(payoutSecret) < 32 {
		slog.Error("LIKI_REFERRAL_PAYOUT_ENCRYPTION_KEY must contain at least 32 bytes")
		os.Exit(1)
	}
	partnerStore, err := partner.Open(filepath.Join(dbs.dir, "referral.db"), []byte(payoutSecret))
	if err != nil {
		slog.Error("init partner store", "err", err)
		os.Exit(1)
	}
	defer func() { _ = partnerStore.Close() }()
	if err := partnerStore.Migrate(ctx); err != nil {
		slog.Error("init partner schema", "err", err)
		os.Exit(1)
	}
	partnerStore.SetCommissionRates(
		cfg.KOLRevenueShareBasisPoints,
		cfg.ScoutRevenueShareBasisPoints,
	)
	partnerHandler := partner.NewHandler(partnerStore, authHandler.RequireUser, !cfg.DevelopmentMode)
	partnerHandler.SetAttributionTTL(time.Duration(cfg.ReferralAttributionTTLDays) * 24 * time.Hour)
	authHandler.SetPostLoginHook(partnerHandler.PostLoginHook)
	authHandler.SetPartnerRolesProvider(partnerStore.PartnerRoles)
	chats, err := newChatWiring(ctx, cfg, likiDB, entitlements, profileStore)
	if err != nil {
		slog.Error("init chat system", "err", err)
		os.Exit(1)
	}

	productCatalog := product.NewCatalog(product.CatalogConfig{
		OneTimePriceMinorCNY:      cfg.ChatOneTimePriceMinorCNY,
		OneTimePriceMinorUSD:      cfg.ChatOneTimePriceMinorUSD,
		SubscriptionPriceMinorUSD: cfg.ChatSubscriptionPriceMinorUSD,
	})
	salesOrders := sales.NewOrderStoreWithCatalog(likiDB, productCatalog)
	if err := salesOrders.Migrate(ctx); err != nil {
		slog.Error("init sales schema", "err", err)
		os.Exit(1)
	}
	payments, err := newPaymentWiring(ctx, cfg, paymentStores{
		Platform: platformDB, Aipay: aipayDB,
		Orders: salesOrders, Subscriptions: subscriptionStore,
		Entitlements: entitlements, Partners: partnerStore,
	}, paymentClients{
		Mailer: emailClient, Dodo: dodoClient, Xunhu: xunhuClient,
	})
	if err != nil {
		slog.Error("init payment system", "err", err)
		os.Exit(1)
	}
	aipayStore := payments.AipayStore
	paymentRuntime := payments.Runtime

	erasureQueue := account.NewErasureStore(platformDB)
	if err := erasureQueue.Migrate(ctx); err != nil {
		slog.Error("account: migrate erasure queue", "err", err)
		os.Exit(1)
	}
	workers := new(sync.WaitGroup)
	workers.Go(func() {
		partnerStore.StartCommissionWorker(ctx, platformDB, 5*time.Second)
	})

	// Validate JWT_SECRET at startup so config errors fail fast.
	if err := httputil.ValidateJWTSecret(cfg.JWTSecret); err != nil {
		slog.Error("validate JWT_SECRET", "err", err)
		os.Exit(1)
	}

	// Setup HTTP
	rateLimiter := apphttp.NewRateLimiter()
	verifyLimiter := apphttp.NewRateLimiter()
	defer rateLimiter.Stop()

	handler, accountHandler := newHTTPHandler(httpWiring{
		config:     cfg,
		platformDB: platformDB, profilesDB: profilesDB, feedbackDB: feedbackDB,
		likiDB: likiDB, aipayDB: aipayDB,
		authStore: authStore, authConfig: authConfig, authHandler: authHandler,
		profiles: profileStore, feedback: feedbackStore,
		chats: chats, payments: payments,
		partners: partnerStore, partnerHandler: partnerHandler,
		entitlements: entitlements, orders: salesOrders, subscriptions: subscriptionStore,
		currency: currencyResolver, catalog: productCatalog, dodo: dodoClient,
		erasureQueue: erasureQueue, adminAuthorizer: adminAuthorizer,
		rateLimiter: rateLimiter, verifyLimiter: verifyLimiter,
		partnerNotifier: func(ctx context.Context, item partner.PartnerApplication, approved bool, reason string) {
			// The decision is already committed; delivery must not be cancelled
			// with the request context.
			ctx = context.WithoutCancel(ctx)
			if err := partnerDecisionEmail(ctx, item, approved, reason); err != nil {
				slog.Warn("send partner decision notification", "application_id", item.ID, "err", err)
			}
		},
	})

	startErasureWorker(workers, ctx, erasureQueue, accountHandler)

	// Write timeout must accommodate LLM calls (~120s)
	srv := &http.Server{
		Addr:         cfg.HTTPListenAddress,
		Handler:      handler,
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 610 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	startAuthCleanupWorker(workers, ctx, authStore, cfg.ExpiredSessionRetentionDays)

	startProfileCleanupWorker(workers, ctx, profileStore, cfg.DeletedProfileRetentionDays)

	startPaymentWorkers(workers, ctx, paymentRuntime, aipayStore, cfg)

	// Signal handling
	go handleSignals(srv)

	slog.Info("liki listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
	slog.Info("server stopped")

	// HTTP is down; cancel background workers and wait for in-flight work
	// before the deferred database closes run.
	cancel()
	workers.Wait()

}
