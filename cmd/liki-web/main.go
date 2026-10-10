package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ml8s/liki-web/internal/sqliteutil"

	"github.com/ml8s/liki-web/internal/account"
	"github.com/ml8s/liki-web/internal/adminauth"
	"github.com/ml8s/liki-web/internal/aipay"
	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/dodo"
	"github.com/ml8s/liki-web/internal/profile"
	"github.com/ml8s/liki-web/internal/referral"
	"github.com/ml8s/liki-web/internal/xunhu"

	"github.com/ml8s/liki-web/internal/email"
	"github.com/ml8s/liki-web/internal/feedback"
	apphttp "github.com/ml8s/liki-web/internal/http"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/liki"
	"github.com/ml8s/liki-web/internal/metrics"
	"github.com/ml8s/liki-web/internal/payment"
)

// Build-time variables injected via -ldflags.
var (
	BuildTime = "dev"
	Version   = "dev"
)

// maxAgentRunTimeout matches the bounded AG-UI product run deadline. The HTTP
// writer keeps an additional margin so a completed response can be flushed.
const maxAgentRunTimeout = 600 * time.Second

type Config struct {
	DataDir              string
	FeedbackAPIKey       string
	ResendAPIKey         string
	ResendFrom           string
	DodoAPIKey           string
	DodoWebhookKey       string
	DodoTestMode         bool
	DodoProductLikiChat  string
	XunhuAppID           string
	XunhuAppSecret       string
	PaymentWebhookURL    string
	AlipayConfig         aipay.Config
	EngineURL            string
	AgentURL             string
	AgentInternalToken   string
	AgentTimeoutSeconds  int
	ReturnURL            string
	JWTSecret            string
	ListenAddr           string
	DevMode              bool
	CurrencySource       string
	BillingRegion        string
	EntitlementDays      int
	PriceMinorCNY        int
	PriceMinorUSD        int
	ReferrerRateBPS      int
	RecruiterRateBPS     int
	AttributionTTLDays   int
	PendingExpireDays    int
	NotifyRetentionDays  int
	ProfileRetentionDays int
	SessionRetentionDays int
}

func LoadConfig() Config {
	return Config{
		DataDir:              envOr("LIKI_DATA_DIR", "/var/lib/liki"),
		FeedbackAPIKey:       os.Getenv("FEEDBACK_API_KEY"),
		ResendAPIKey:         os.Getenv("RESEND_API_KEY"),
		ResendFrom:           envOr("RESEND_FROM", envOr("EMAIL_FROM", "Liki <report@liki.email>")),
		DodoAPIKey:           envOr("DODO_API_KEY", ""),
		DodoWebhookKey:       envOr("DODO_WEBHOOK_KEY", ""),
		DodoTestMode:         envOrBool("DODO_TEST_MODE", false),
		DodoProductLikiChat:  os.Getenv("DODO_PRODUCT_LIKI_CHAT"),
		XunhuAppID:           envOr("XUNHU_APPID", ""),
		XunhuAppSecret:       envOr("XUNHU_APPSECRET", ""),
		PaymentWebhookURL:    envOr("PAYMENTS_WEBHOOK_URL", strings.TrimSuffix(envOr("RETURN_URL", ""), "/")+"/api/payments/webhook"),
		AlipayConfig:         loadAlipayConfig(),
		EngineURL:            envOr("LIKI_ENGINE_URL", "http://localhost:8082/jsonrpc"),
		AgentURL:             strings.TrimRight(envOr("LIKI_AGENTS_URL", ""), "/"),
		AgentInternalToken:   os.Getenv("LIKI_AGENTS_INTERNAL_TOKEN"),
		AgentTimeoutSeconds:  envOrInt("LIKI_AGENTS_TIMEOUT_SECONDS", 140),
		ReturnURL:            envOr("RETURN_URL", ""),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		ListenAddr:           envOr("LISTEN_ADDR", ":8080"),
		DevMode:              envOrBool("DEV_MODE", false),
		CurrencySource:       envOr("LIKI_CURRENCY_SOURCE", liki.CurrencySourceTrustedEdge),
		BillingRegion:        envOr("LIKI_BILLING_REGION", "global"),
		EntitlementDays:      envOrInt("LIKI_ENTITLEMENT_DAYS", 30),
		PriceMinorCNY:        envOrInt("LIKI_PRICE_CNY_MINOR", 100),
		PriceMinorUSD:        envOrInt("LIKI_PRICE_USD_MINOR", 100),
		ReferrerRateBPS:      envOrInt("LIKI_REFERRER_RATE_BPS", 5000),
		RecruiterRateBPS:     envOrInt("LIKI_RECRUITER_RATE_BPS", 2000),
		AttributionTTLDays:   envOrInt("LIKI_ATTRIBUTION_TTL_DAYS", 30),
		PendingExpireDays:    envOrInt("LIKI_PAYMENT_PENDING_EXPIRE_DAYS", 7),
		NotifyRetentionDays:  envOrInt("LIKI_NOTIFICATION_RETENTION_DAYS", 90),
		ProfileRetentionDays: envOrInt("LIKI_PROFILE_RETENTION_DAYS", 30),
		SessionRetentionDays: envOrInt("LIKI_SESSION_RETENTION_DAYS", 30),
	}
}

func loadAlipayConfig() aipay.Config {
	sellerName := envOr("ALIPAY_SELLER_NAME", "Liki")
	if path := os.Getenv("ALIPAY_CONFIG_FILE"); path != "" {
		config, err := aipay.LoadSandboxConfig(path)
		if err != nil {
			// Configuration errors must be visible while the endpoint remains
			// disabled; they must not leak credential material.
			slog.Warn("aipay disabled", "err", err)
			return aipay.Config{}
		}
		config.SellerName = sellerName
		config.Amount = os.Getenv("ALIPAY_AIPAY_AMOUNT")
		config.GoodsName = aipay.DefaultGoodsName
		return config
	}
	return aipay.Config{
		AppID:           os.Getenv("ALIPAY_APP_ID"),
		PrivateKey:      os.Getenv("ALIPAY_PRIVATE_KEY"),
		AlipayPublicKey: os.Getenv("ALIPAY_PUBLIC_KEY"),
		Gateway:         envOr("ALIPAY_GATEWAY", "https://openapi.alipay.com/gateway.do"),
		SellerID:        os.Getenv("ALIPAY_SELLER_ID"),
		SellerName:      sellerName,
		ServiceID:       os.Getenv("ALIPAY_SERVICE_ID"),
		Amount:          os.Getenv("ALIPAY_AIPAY_AMOUNT"),
		GoodsName:       aipay.DefaultGoodsName,
	}
}

func main() {
	cfg := LoadConfig()

	currencyResolver, err := liki.NewCurrencyResolver(cfg.CurrencySource, cfg.BillingRegion)
	if err != nil {
		slog.Error("invalid billing currency configuration", "err", err)
		os.Exit(1)
	}

	// Structured logging
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

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
	dataDir := cfg.DataDir
	open := func(name string) *sql.DB {
		db, err := sqliteutil.Open(filepath.Join(dataDir, name))
		if err != nil {
			slog.Error("open database", "name", name, "err", err)
			os.Exit(1)
		}
		return db
	}

	platformDB := open("platform.db")
	defer func() { _ = platformDB.Close() }()
	profilesDB := open("profiles.db")
	defer func() { _ = profilesDB.Close() }()
	feedbackDB := open("feedback.db")
	defer func() { _ = feedbackDB.Close() }()
	likiDB := open("liki.db")
	defer func() { _ = likiDB.Close() }()
	aipayDB := open("aipay.db")
	defer func() { _ = aipayDB.Close() }()

	// ── platform (auth + payment runtime) ──
	authStore := auth.NewStore(platformDB, []byte(cfg.JWTSecret))

	// ── profiles ──
	profileStore := profile.NewStore(profilesDB)

	// ── feedback ──
	feedbackStore, err := feedback.NewStore(feedbackDB)
	if err != nil {
		slog.Error("init feedback store", "err", err)
		os.Exit(1)
	}

	// Init services
	emailClient := email.New(cfg.ResendAPIKey, cfg.ResendFrom)

	dodoProducts := map[string]string{
		string(liki.ProductChat): cfg.DodoProductLikiChat,
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

	if cfg.ReturnURL == "" {
		slog.Warn("RETURN_URL not set — post-payment redirect will fail")
	}

	if err := authStore.Migrate(ctx); err != nil {
		slog.Error("init auth schema", "err", err)
		os.Exit(1)
	}

	adminAuthorizer, err := adminauth.Open(filepath.Join(dataDir, "platform.db"))
	if err != nil {
		slog.Error("init admin auth", "err", err)
		os.Exit(1)
	}
	defer func() { _ = adminAuthorizer.Close() }()
	if err := adminAuthorizer.Migrate(ctx); err != nil {
		slog.Error("init admin auth schema", "err", err)
		os.Exit(1)
	}
	if adminEmail := os.Getenv("ADMIN_EMAIL"); adminEmail != "" {
		if err := adminAuthorizer.BootstrapAdminByEmail(ctx, adminEmail); err != nil {
			slog.Error("bootstrap admin", "err", err)
			os.Exit(1)
		}
	} else {
		slog.Warn("ADMIN_EMAIL is empty; admin console capabilities are disabled")
	}

	// ── Callbacks ──
	authMailer := auth.MailerFunc(func(ctx context.Context, email, code, locale string) error {
		subject, body := "Liki login code", `<p>Your Liki login code:</p><p style="font-size:28px;font-weight:bold;letter-spacing:6px">`+code+`</p><p>The code is valid for 10 minutes.</p>`
		switch locale {
		case "zh-Hant":
			subject, body = "Liki 登入驗證碼", `<p>你的 Liki 登入驗證碼：</p><p style="font-size:28px;font-weight:bold;letter-spacing:6px">`+code+`</p><p>驗證碼 10 分鐘內有效。</p>`
		case "zh-Hans", "":
			subject, body = "Liki 登录验证码", `<p>你的 Liki 登录验证码：</p><p style="font-size:28px;font-weight:bold;letter-spacing:6px">`+code+`</p><p>验证码 10 分钟内有效。</p>`
		}
		return emailClient.SendReport(ctx, email, subject, body)
	})
	partnerDecisionEmail := func(ctx context.Context, item referral.PartnerApplication, approved bool, reason string) error {
		applicant, err := authStore.GetUser(ctx, item.UserID)
		if err != nil {
			return err
		}
		role := "KOL"
		if item.Type == referral.RoleScout {
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
				opening = "你的 " + role + " 合作申请未通过。"
			}
		}
		body := `<p>` + opening + `</p><p>` + noteLabel + html.EscapeString(reason) + `</p><p>` + next + `</p>`
		return emailClient.SendReport(ctx, applicant.Email, subject, body)
	}
	authConfig := auth.DefaultConfig()
	authConfig.SecureCookie = !cfg.DevMode
	authConfig.ExposeDevCode = cfg.DevMode
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

	entitlements := liki.NewEntitlementStore(likiDB)
	if err := entitlements.Migrate(ctx); err != nil {
		slog.Error("init entitlements", "err", err)
		os.Exit(1)
	}
	if _, err := entitlements.ExpireDue(ctx); err != nil {
		slog.Warn("expire entitlements", "err", err)
	}
	payoutSecret := os.Getenv("REFERRAL_PAYOUT_SECRET")
	if len(payoutSecret) < 32 {
		slog.Error("REFERRAL_PAYOUT_SECRET must contain at least 32 bytes")
		os.Exit(1)
	}
	referralStore, err := referral.Open(filepath.Join(dataDir, "referral.db"), []byte(payoutSecret))
	if err != nil {
		slog.Error("init referral store", "err", err)
		os.Exit(1)
	}
	defer func() { _ = referralStore.Close() }()
	if err := referralStore.Migrate(ctx); err != nil {
		slog.Error("init referral schema", "err", err)
		os.Exit(1)
	}
	referralHandler := referral.NewHandler(referralStore, authHandler.RequireUser, !cfg.DevMode)
	authHandler.SetPostLoginHook(referralHandler.PostLoginHook)
	chatStore := liki.NewChatStore(likiDB)
	if err := chatStore.Migrate(ctx); err != nil {
		slog.Error("init chat schema", "err", err)
		os.Exit(1)
	}
	chatService := liki.NewService(chatStore, entitlements, profileStore)
	if cfg.AgentURL == "" {
		slog.Error("LIKI_AGENTS_URL is required")
		os.Exit(1)
	}
	if cfg.AgentInternalToken == "" {
		slog.Error("LIKI_AGENTS_INTERNAL_TOKEN is required")
		os.Exit(1)
	}
	agentTimeout := time.Duration(cfg.AgentTimeoutSeconds) * time.Second
	if agentTimeout <= 0 || agentTimeout > maxAgentRunTimeout {
		agentTimeout = maxAgentRunTimeout
	}
	agentRunner, err := liki.NewAgentRunner(liki.AgentRunnerConfig{
		BaseURL: cfg.AgentURL,
		Token:   cfg.AgentInternalToken,
		Timeout: agentTimeout,
		Product: "liki",
	})
	if err != nil {
		slog.Error("init liki agent runner", "err", err)
		os.Exit(1)
	}
	chatService.SetAgentRunner(agentRunner)
	commerceOrders := liki.NewOrderStore(likiDB)
	if err := commerceOrders.Migrate(ctx); err != nil {
		slog.Error("init commerce schema", "err", err)
		os.Exit(1)
	}
	aipayStore := aipay.NewStore(aipayDB)
	if err := aipayStore.Migrate(ctx); err != nil {
		slog.Error("init aipay schema", "err", err)
		os.Exit(1)
	}
	var aipayGateway aipay.Gateway
	if cfg.AlipayConfig.Enabled() {
		client, err := aipay.NewClient(cfg.AlipayConfig)
		if err != nil {
			slog.Error("init alipay aipay client", "err", err)
			os.Exit(1)
		}
		aipayGateway = client
	} else {
		slog.Warn("alipay aipay is not fully configured; it is disabled")
	}
	aipayHandler, err := aipay.NewHandler(aipayStore, aipayGateway, cfg.AlipayConfig)
	if err != nil {
		slog.Error("init aipay handler", "err", err)
		os.Exit(1)
	}
	chatDuration := 30 * 24 * time.Hour

	intents := payment.NewIntentStore(platformDB)
	notifications := payment.NewNotificationStore(platformDB)
	fulfillments := payment.NewFulfillmentTaskStore(platformDB)
	events := payment.NewPaymentEventStore(platformDB)
	if err := intents.Migrate(ctx); err != nil {
		slog.Error("init payment intents", "err", err)
		os.Exit(1)
	}
	if err := notifications.Migrate(ctx); err != nil {
		slog.Error("init payment notifications", "err", err)
		os.Exit(1)
	}
	if err := fulfillments.Migrate(ctx); err != nil {
		slog.Error("init payment fulfillment tasks", "err", err)
		os.Exit(1)
	}
	if err := events.Migrate(ctx); err != nil {
		slog.Error("init payment events", "err", err)
		os.Exit(1)
	}
	dodoConfigured := cfg.DodoAPIKey != "" && cfg.DodoWebhookKey != ""
	if dodoConfigured && cfg.DodoProductLikiChat == "" {
		slog.Warn("dodo product is not configured for liki-chat; USD chat checkout will fail")
	}
	xunhuConfigured := cfg.XunhuAppID != "" && cfg.XunhuAppSecret != "" && cfg.PaymentWebhookURL != ""
	if dodoConfigured || xunhuConfigured {
		if err := validateHTTPURL("RETURN_URL", cfg.ReturnURL); err != nil {
			slog.Error("payment runtime requires an absolute public RETURN_URL", "err", err)
			os.Exit(1)
		}
	}
	if !dodoConfigured && !xunhuConfigured {
		slog.Error("no payment provider is fully configured")
		os.Exit(1)
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
	flows := map[string]payment.ProductFlow{
		string(liki.ProductChat): liki.NewFulfiller(commerceOrders, entitlements, liki.ProductChat, chatDuration),
	}
	paymentSender := paymentEmailSender{client: emailClient}
	paymentRuntime := payment.NewRuntime(gateways, flows, intents, notifications, fulfillments, events, paymentSender, payment.RuntimeOptions{
		BaseURL:    cfg.ReturnURL,
		AdminEmail: os.Getenv("ADMIN_EMAIL"),
	})
	paymentRuntime.SetFulfillmentObserver(func(ctx context.Context, intent payment.PaymentIntent) {
		if _, _, err := referralStore.EnqueueCommissionIntent(ctx, intent); err != nil {
			slog.Error("referral: enqueue fulfilled payment", "intent_id", intent.ID, "err", err)
		}
	})
	paymentRuntime.SetRefundObserver(func(ctx context.Context, event payment.WebhookEvent, intent payment.PaymentIntent) error {
		return referralStore.EnqueueCommissionRefund(ctx, event, intent)
	})
	erasureQueue := account.NewErasureStore(platformDB)
	if err := erasureQueue.Migrate(ctx); err != nil {
		slog.Error("account: migrate erasure queue", "err", err)
		os.Exit(1)
	}
	workers := new(sync.WaitGroup)
	workers.Go(func() {
		referralStore.StartCommissionWorker(ctx, platformDB, 5*time.Second)
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

	readiness := func(ctx context.Context) error {
		for name, db := range map[string]*sql.DB{
			"platform": platformDB, "profiles": profilesDB, "feedback": feedbackDB,
			"liki": likiDB, "aipay": aipayDB,
		} {
			if err := db.PingContext(ctx); err != nil {
				return fmt.Errorf("database %s: %w", name, err)
			}
		}
		if err := checkEngineReadiness(ctx, cfg.EngineURL); err != nil {
			return err
		}
		if err := checkAgentReadiness(ctx, cfg.AgentURL); err != nil {
			return err
		}
		return nil
	}

	mux := http.NewServeMux()
	accountHandler := account.NewHandler(
		profileStore,
		chatStore,
		authStore,
		commerceOrders,
		entitlements,
		intents,
		notifications,
		referralStore,
		adminAuthorizer,
		erasureQueue,
		authHandler.RequireUser,
	)

	h := apphttp.RegisterRoutes(mux, apphttp.ServerDeps{
		PaymentRuntime:  paymentRuntime,
		AuthHandler:     authHandler,
		ReferralHandler: referralHandler,
		ReferralStore:   referralStore,
		LikiHandler:     liki.NewHandler(commerceOrders, entitlements, chatService, paymentRuntime, authHandler.RequireUser, liki.WithCurrencyResolver(currencyResolver)),
		AipayHandler:    aipayHandler,
		ProfileHandler:  profile.NewHandler(profileStore, authHandler.RequireUser),
		AccountHandler:  accountHandler,
		FeedbackStore:   feedbackStore,
		FeedbackAPIKey:  cfg.FeedbackAPIKey,
		AdminEmail:      os.Getenv("ADMIN_EMAIL"),
		AdminAuthorizer: adminAuthorizer,
		PartnerNotifier: func(ctx context.Context, item referral.PartnerApplication, approved bool, reason string) {
			// The decision is already committed; delivery must not be
			// cancelled with the request context.
			ctx = context.WithoutCancel(ctx)
			if err := partnerDecisionEmail(ctx, item, approved, reason); err != nil {
				slog.Warn("send partner decision notification", "application_id", item.ID, "err", err)
			}
		},
		MetricsHandler: metrics.Handler(os.Getenv("METRICS_TOKEN")),
		Readiness:      readiness,
		DevMode:        cfg.DevMode,
	}, BuildTime, Version, rateLimiter, verifyLimiter)

	workers.Go(func() {
		erasureWorker := &account.ErasureWorker{Queue: erasureQueue, Handler: accountHandler, Interval: 30 * time.Second}
		erasureWorker.Start(ctx)
	})

	// Write timeout must accommodate LLM calls (~120s)
	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      h,
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 610 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	workers.Go(func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := authStore.Cleanup(ctx); err != nil {
					slog.Error("auth cleanup", "err", err)
				}
			}
		}
	})

	// 软删档案 30 天恢复窗口到期后物理清理（个保法删除权落地）。
	workers.Go(func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour)
				if n, err := profileStore.CleanupExpired(ctx, cutoff); err != nil {
					slog.Error("profile cleanup", "err", err)
				} else if n > 0 {
					slog.Info("profile cleanup complete", "purged", n)
				}
			}
		}
	})

	workers.Go(func() {
		paymentRuntime.StartNotificationWorker(ctx, 30*time.Second)
	})
	workers.Go(func() {
		paymentRuntime.StartFulfillmentWorker(ctx, 5*time.Second)
	})
	workers.Go(func() {
		paymentRuntime.StartReconciliationWorker(ctx, time.Minute)
	})

	// Payment retention: expire abandoned checkouts and prune sent receipts.
	workers.Go(func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				intents, notifications, err := paymentRuntime.RunRetention(ctx, 7*24*time.Hour, 90*24*time.Hour)
				if err != nil {
					slog.Error("payment retention", "err", err)
					continue
				}
				if intents > 0 || notifications > 0 {
					slog.Info("payment retention complete", "expired_intents", intents, "deleted_notifications", notifications)
				}
				if n, err := aipayStore.CleanupExpiredBills(ctx, time.Now().UTC()); err != nil {
					slog.Error("aipay bill cleanup", "err", err)
				} else if n > 0 {
					slog.Info("aipay expired bills cleaned", "count", n)
				}
			}
		}
	})

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

func requireEnv(pairs ...string) {
	for i := 0; i < len(pairs); i += 2 {
		if pairs[i+1] == "" {
			slog.Error("required environment variable not set", "var", pairs[i])
			os.Exit(1)
		}
	}
}

// validateBusinessConfig fail-fast 检查业务/合规参数：错误配置宁可不启动，
// 也不能把错误价格或比例带进资金路径。
func validateBusinessConfig(cfg Config) error {
	for name, minor := range map[string]int{
		"LIKI_PRICE_CNY_MINOR": cfg.PriceMinorCNY,
		"LIKI_PRICE_USD_MINOR": cfg.PriceMinorUSD,
	} {
		if minor <= 0 || minor > 1_000_000_000 {
			return fmt.Errorf("%s must be within (0, 1000000000] minor units, got %d", name, minor)
		}
	}
	for name, bps := range map[string]int{
		"LIKI_REFERRER_RATE_BPS":  cfg.ReferrerRateBPS,
		"LIKI_RECRUITER_RATE_BPS": cfg.RecruiterRateBPS,
	} {
		if bps < 0 || bps > 10000 {
			return fmt.Errorf("%s must be within [0,10000] bps, got %d", name, bps)
		}
	}
	if cfg.ReferrerRateBPS+cfg.RecruiterRateBPS > 10000 {
		return fmt.Errorf("referrer+recruiter rates must not exceed 10000 bps, got %d",
			cfg.ReferrerRateBPS+cfg.RecruiterRateBPS)
	}
	for name, days := range map[string]int{
		"LIKI_ENTITLEMENT_DAYS":            cfg.EntitlementDays,
		"LIKI_ATTRIBUTION_TTL_DAYS":        cfg.AttributionTTLDays,
		"LIKI_PAYMENT_PENDING_EXPIRE_DAYS": cfg.PendingExpireDays,
		"LIKI_NOTIFICATION_RETENTION_DAYS": cfg.NotifyRetentionDays,
		"LIKI_PROFILE_RETENTION_DAYS":      cfg.ProfileRetentionDays,
		"LIKI_SESSION_RETENTION_DAYS":      cfg.SessionRetentionDays,
	} {
		if days < 1 || days > 3650 {
			return fmt.Errorf("%s must be within [1,3650] days, got %d", name, days)
		}
	}
	return nil
}

func validateHTTPURL(name, value string) error {
	if value == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf("%s %q: %w", name, value, err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("%s %q must be an absolute http(s) URL", name, value)
	}
	return nil
}

func validateRuntimeURLs(cfg Config) error {
	var errs []error
	if err := validateHTTPURL("LIKI_ENGINE_URL", cfg.EngineURL); err != nil {
		errs = append(errs, err)
	}
	if err := validateHTTPURL("RETURN_URL", cfg.ReturnURL); err != nil {
		errs = append(errs, err)
	}
	if err := validateHTTPURL("PAYMENTS_WEBHOOK_URL", cfg.PaymentWebhookURL); err != nil {
		errs = append(errs, err)
	}
	if err := validateHTTPURL("LIKI_AGENTS_URL", cfg.AgentURL); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envOrBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			slog.Warn("invalid bool env", "key", key, "value", v, "default", def)
			return def
		}
		return b
	}
	return def
}

func envOrInt(key string, def int) int {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			slog.Warn("invalid integer env", "key", key, "value", value, "default", def)
			return def
		}
		return parsed
	}
	return def
}

func handleSignals(srv *http.Server) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in handleSignals", "panic", r)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("received signal, shutting down", "signal", sig.String())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("forced shutdown", "err", err)
	}
}
