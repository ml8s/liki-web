package main

import "testing"

func TestLoadConfigReadsStandardBusinessContract(t *testing.T) {
	t.Setenv("LIKI_HTTP_LISTEN_ADDRESS", ":18082")
	t.Setenv("LIKI_PUBLIC_BASE_URL", "https://liki.hk")
	t.Setenv("DODO_PRODUCT_LIKI_CHAT_ONE_TIME", "pdt_one_time")
	t.Setenv("DODO_PRODUCT_LIKI_CHAT_SUBSCRIPTION", "pdt_subscription")
	t.Setenv("LIKI_CHAT_ONE_TIME_PRICE_CNY_MINOR_UNITS", "7900")
	t.Setenv("LIKI_CHAT_ONE_TIME_PRICE_USD_MINOR_UNITS", "2990")
	t.Setenv("LIKI_CHAT_SUBSCRIPTION_PRICE_USD_MINOR_UNITS", "1990")
	t.Setenv("LIKI_CHAT_ONE_TIME_ACCESS_DAYS", "30")
	t.Setenv("LIKI_CHAT_KOL_REVENUE_SHARE_BASIS_POINTS", "4000")
	t.Setenv("LIKI_CHAT_SCOUT_REVENUE_SHARE_BASIS_POINTS", "1000")

	config := LoadConfig()

	if config.HTTPListenAddress != ":18082" {
		t.Fatalf("HTTPListenAddress = %q", config.HTTPListenAddress)
	}
	if config.PublicBaseURL != "https://liki.hk" {
		t.Fatalf("PublicBaseURL = %q", config.PublicBaseURL)
	}
	if config.DodoProductLikiChatOneTime != "pdt_one_time" ||
		config.DodoProductLikiChatSubscription != "pdt_subscription" {
		t.Fatalf("Dodo products = %q / %q", config.DodoProductLikiChatOneTime, config.DodoProductLikiChatSubscription)
	}
	if config.ChatOneTimePriceMinorCNY != 7900 || config.ChatOneTimePriceMinorUSD != 2990 {
		t.Fatalf("one-time prices = %d / %d", config.ChatOneTimePriceMinorCNY, config.ChatOneTimePriceMinorUSD)
	}
	if config.ChatSubscriptionPriceMinorUSD != 1990 {
		t.Fatalf("subscription price = %d", config.ChatSubscriptionPriceMinorUSD)
	}
	if config.ChatOneTimeAccessDays != 30 {
		t.Fatalf("one-time access days = %d", config.ChatOneTimeAccessDays)
	}
	if config.KOLRevenueShareBasisPoints != 4000 || config.ScoutRevenueShareBasisPoints != 1000 {
		t.Fatalf("revenue shares = %d / %d", config.KOLRevenueShareBasisPoints, config.ScoutRevenueShareBasisPoints)
	}
}

func TestValidateBusinessConfigRejectsBadSubscriptionContract(t *testing.T) {
	config := Config{
		ChatOneTimePriceMinorCNY:         7900,
		ChatOneTimePriceMinorUSD:         2990,
		ChatSubscriptionPriceMinorUSD:    0,
		ChatOneTimeAccessDays:            30,
		KOLRevenueShareBasisPoints:       4000,
		ScoutRevenueShareBasisPoints:     1000,
		ReferralAttributionTTLDays:       30,
		PaymentIntentExpiryDays:          7,
		PaymentNotificationRetentionDays: 90,
		DeletedProfileRetentionDays:      30,
		ExpiredSessionRetentionDays:      30,
	}
	if err := validateBusinessConfig(config); err == nil {
		t.Fatal("validateBusinessConfig(bad subscription price) = nil, want error")
	}
}
