package liki

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestSubscriptionEventHandlerRecordsPaymentAndAppliesLifecycle(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	entitlements := NewEntitlementStore(db)
	subscriptions := NewSubscriptionStore(db)
	for _, migrate := range []func(context.Context) error{entitlements.Migrate, subscriptions.Migrate} {
		if err := migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	pending, err := subscriptions.CreatePending(ctx, "user-1", "user@example.com", ProductChat, CurrencyUSD)
	if err != nil {
		t.Fatal(err)
	}
	periodStart := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)
	periodStartPtr := &periodStart
	periodEndPtr := &periodEnd
	observed := 0
	handler := NewSubscriptionEventHandler(subscriptions, entitlements, func(_ context.Context, _ payment.WebhookEvent) error {
		observed++
		return nil
	})

	paymentEvent := payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     payment.EventPaymentSucceeded,
		Data: payment.WebhookEventData{
			SubscriptionID: "sub_1", PaymentID: "pay_1", Amount: 1990,
			Currency: "USD", Email: "user@example.com",
			Metadata: map[string]string{"liki_user_id": "user-1"},
		},
	}
	if err := handler.HandleSubscriptionEvent(ctx, paymentEvent); err != nil {
		t.Fatal(err)
	}
	if observed != 1 {
		t.Fatalf("observer count = %d, want 1", observed)
	}

	if err := handler.HandleSubscriptionEvent(ctx, paymentEvent); err != nil {
		t.Fatal(err)
	}
	if observed != 1 {
		t.Fatalf("duplicate payment observer count = %d, want 1", observed)
	}

	active := payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     payment.EventSubscriptionActive,
		Data: payment.WebhookEventData{
			SubscriptionID: "sub_1", Status: "active", Currency: "USD",
			PeriodStart: periodStartPtr, PeriodEnd: periodEndPtr,
			Metadata: map[string]string{"liki_subscription_key": pending.ID},
		},
	}
	if err := handler.HandleSubscriptionEvent(ctx, active); err != nil {
		t.Fatal(err)
	}
	allowed, err := entitlements.AllowedSkills(ctx, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(allowed) != 1 || allowed[0] != SkillLiki {
		t.Fatalf("allowed = %#v", allowed)
	}
}
