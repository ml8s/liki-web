package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/ml8s/liki-web/internal/aipay"
	"github.com/ml8s/liki-web/internal/chat"
	"github.com/ml8s/liki-web/internal/sales"
)

type Config struct {
	DataDir                          string
	AdminEmail                       string
	MetricsToken                     string
	FeedbackAPIKey                   string
	ResendAPIKey                     string
	ResendFrom                       string
	DodoAPIKey                       string
	DodoWebhookKey                   string
	DodoTestMode                     bool
	DodoProductLikiChatOneTime       string
	DodoProductLikiChatSubscription  string
	XunhuAppID                       string
	XunhuAppSecret                   string
	PaymentWebhookURL                string
	AlipayConfig                     aipay.Config
	AgentURLs                        map[string]string
	AgentInternalToken               string
	AgentsOpsPort                    string
	AgentTimeoutSeconds              int
	PublicBaseURL                    string
	JWTSecret                        string
	HTTPListenAddress                string
	DevelopmentMode                  bool
	CurrencySource                   string
	BillingRegion                    string
	ChatOneTimeAccessDays            int
	ChatOneTimePriceMinorCNY         int64
	ChatOneTimePriceMinorUSD         int64
	ChatSubscriptionPriceMinorUSD    int64
	KOLRevenueShareBasisPoints       int
	ScoutRevenueShareBasisPoints     int
	ReferralAttributionTTLDays       int
	PaymentIntentExpiryDays          int
	PaymentNotificationRetentionDays int
	DeletedProfileRetentionDays      int
	ExpiredSessionRetentionDays      int
}

func LoadConfig() Config {
	return Config{
		DataDir:                          envOr("LIKI_DATA_DIR", "/var/lib/liki"),
		AdminEmail:                       os.Getenv("LIKI_ADMIN_EMAIL"),
		MetricsToken:                     os.Getenv("LIKI_METRICS_TOKEN"),
		FeedbackAPIKey:                   os.Getenv("LIKI_FEEDBACK_API_TOKEN"),
		ResendAPIKey:                     os.Getenv("RESEND_API_KEY"),
		ResendFrom:                       envOr("LIKI_EMAIL_FROM_ADDRESS", "Liki <report@liki.email>"),
		DodoAPIKey:                       envOr("DODO_API_KEY", ""),
		DodoWebhookKey:                   envOr("DODO_WEBHOOK_KEY", ""),
		DodoTestMode:                     envOrBool("DODO_TEST_MODE", false),
		DodoProductLikiChatOneTime:       os.Getenv("DODO_PRODUCT_LIKI_CHAT_ONE_TIME"),
		DodoProductLikiChatSubscription:  os.Getenv("DODO_PRODUCT_LIKI_CHAT_SUBSCRIPTION"),
		XunhuAppID:                       envOr("XUNHU_APPID", ""),
		XunhuAppSecret:                   envOr("XUNHU_APPSECRET", ""),
		PaymentWebhookURL:                envOr("LIKI_PAYMENT_WEBHOOK_URL", strings.TrimSuffix(envOr("LIKI_PUBLIC_BASE_URL", ""), "/")+"/api/payments/webhook"),
		AlipayConfig:                     loadAlipayConfig(),
		AgentURLs:                        loadAgentURLs(),
		AgentsOpsPort:                    envOr("LIKI_AGENTS_OPS_PORT", "8085"),
		AgentInternalToken:               os.Getenv("LIKI_AGENTS_INTERNAL_TOKEN"),
		AgentTimeoutSeconds:              envOrInt("LIKI_AGENTS_TIMEOUT_SECONDS", 600),
		PublicBaseURL:                    envOr("LIKI_PUBLIC_BASE_URL", ""),
		JWTSecret:                        os.Getenv("JWT_SECRET"),
		HTTPListenAddress:                envOr("LIKI_HTTP_LISTEN_ADDRESS", ":8081"),
		DevelopmentMode:                  envOrBool("LIKI_DEVELOPMENT_MODE", false),
		CurrencySource:                   envOr("LIKI_BILLING_CURRENCY_SOURCE", sales.CurrencySourceTrustedEdge),
		BillingRegion:                    envOr("LIKI_BILLING_REGION", "global"),
		ChatOneTimeAccessDays:            envOrInt("LIKI_CHAT_ONE_TIME_ACCESS_DAYS", 30),
		ChatOneTimePriceMinorCNY:         envOrInt64("LIKI_CHAT_ONE_TIME_PRICE_CNY_MINOR_UNITS", 0),
		ChatOneTimePriceMinorUSD:         envOrInt64("LIKI_CHAT_ONE_TIME_PRICE_USD_MINOR_UNITS", 0),
		ChatSubscriptionPriceMinorUSD:    envOrInt64("LIKI_CHAT_SUBSCRIPTION_PRICE_USD_MINOR_UNITS", 0),
		KOLRevenueShareBasisPoints:       envOrInt("LIKI_CHAT_KOL_REVENUE_SHARE_BASIS_POINTS", 5000),
		ScoutRevenueShareBasisPoints:     envOrInt("LIKI_CHAT_SCOUT_REVENUE_SHARE_BASIS_POINTS", 2000),
		ReferralAttributionTTLDays:       envOrInt("LIKI_REFERRAL_ATTRIBUTION_TTL_DAYS", 30),
		PaymentIntentExpiryDays:          envOrInt("LIKI_PAYMENT_INTENT_EXPIRY_DAYS", 7),
		PaymentNotificationRetentionDays: envOrInt("LIKI_PAYMENT_NOTIFICATION_RETENTION_DAYS", 90),
		DeletedProfileRetentionDays:      envOrInt("LIKI_DELETED_PROFILE_RETENTION_DAYS", 30),
		ExpiredSessionRetentionDays:      envOrInt("LIKI_EXPIRED_SESSION_RETENTION_DAYS", 30),
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
	for name, minor := range map[string]int64{
		"LIKI_CHAT_ONE_TIME_PRICE_CNY_MINOR_UNITS":     cfg.ChatOneTimePriceMinorCNY,
		"LIKI_CHAT_ONE_TIME_PRICE_USD_MINOR_UNITS":     cfg.ChatOneTimePriceMinorUSD,
		"LIKI_CHAT_SUBSCRIPTION_PRICE_USD_MINOR_UNITS": cfg.ChatSubscriptionPriceMinorUSD,
	} {
		if minor <= 0 || minor > 1_000_000_000 {
			return fmt.Errorf("%s must be within (0, 1000000000] minor units, got %d", name, minor)
		}
	}
	for name, bps := range map[string]int{
		"LIKI_CHAT_KOL_REVENUE_SHARE_BASIS_POINTS":   cfg.KOLRevenueShareBasisPoints,
		"LIKI_CHAT_SCOUT_REVENUE_SHARE_BASIS_POINTS": cfg.ScoutRevenueShareBasisPoints,
	} {
		if bps < 0 || bps > 10000 {
			return fmt.Errorf("%s must be within [0,10000] bps, got %d", name, bps)
		}
	}
	if cfg.KOLRevenueShareBasisPoints+cfg.ScoutRevenueShareBasisPoints > 10000 {
		return fmt.Errorf("KOL+Scout revenue shares must not exceed 10000 basis points, got %d",
			cfg.KOLRevenueShareBasisPoints+cfg.ScoutRevenueShareBasisPoints)
	}
	for name, days := range map[string]int{
		"LIKI_CHAT_ONE_TIME_ACCESS_DAYS":           cfg.ChatOneTimeAccessDays,
		"LIKI_REFERRAL_ATTRIBUTION_TTL_DAYS":       cfg.ReferralAttributionTTLDays,
		"LIKI_PAYMENT_INTENT_EXPIRY_DAYS":          cfg.PaymentIntentExpiryDays,
		"LIKI_PAYMENT_NOTIFICATION_RETENTION_DAYS": cfg.PaymentNotificationRetentionDays,
		"LIKI_DELETED_PROFILE_RETENTION_DAYS":      cfg.DeletedProfileRetentionDays,
		"LIKI_EXPIRED_SESSION_RETENTION_DAYS":      cfg.ExpiredSessionRetentionDays,
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
	if err := validateHTTPURL("LIKI_PUBLIC_BASE_URL", cfg.PublicBaseURL); err != nil {
		errs = append(errs, err)
	}
	if err := validateHTTPURL("LIKI_PAYMENT_WEBHOOK_URL", cfg.PaymentWebhookURL); err != nil {
		errs = append(errs, err)
	}
	for _, expert := range chat.ExpertOrder {
		name := "LIKI_AGENTS_" + strings.ToUpper(expert) + "_URL"
		if err := validateHTTPURL(name, cfg.AgentURLs[expert]); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// loadAgentURLs resolves the per-expert runtime endpoints. LIKI_AGENTS_URL is
// a fallback for single-deployment development only; production sets one URL
// per expert service.
func loadAgentURLs() map[string]string {
	fallback := strings.TrimRight(envOr("LIKI_AGENTS_URL", ""), "/")
	return map[string]string{
		chat.ExpertNatal:      strings.TrimRight(envOr("LIKI_AGENTS_NATAL_URL", fallback), "/"),
		chat.ExpertDivination: strings.TrimRight(envOr("LIKI_AGENTS_DIVINATION_URL", fallback), "/"),
		chat.ExpertFengshui:   strings.TrimRight(envOr("LIKI_AGENTS_FENGSHUI_URL", fallback), "/"),
		chat.ExpertNaming:     strings.TrimRight(envOr("LIKI_AGENTS_NAMING_URL", fallback), "/"),
	}
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

func envOrInt64(key string, def int64) int64 {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 {
			slog.Warn("invalid integer env", "key", key, "value", value, "default", def)
			return def
		}
		return parsed
	}
	return def
}
