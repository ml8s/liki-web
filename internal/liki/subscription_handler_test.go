package liki

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestSubscriptionEventProcessorRecordsPaymentAndAppliesLifecycle(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	entitlements := entitlement.NewStore(db)
	subscriptions := NewSubscriptionStore(db)
	for _, migrate := range []func(context.Context) error{entitlements.Migrate, subscriptions.Migrate} {
		if err := migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	pending, err := subscriptions.CreatePending(ctx, "user-1", "user@example.com", product.Chat, product.USD)
	if err != nil {
		t.Fatal(err)
	}
	periodStart := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)
	periodStartPtr := &periodStart
	periodEndPtr := &periodEnd
	observed := 0
	processor := NewSubscriptionEventProcessor(subscriptions, entitlements, "pdt_subscription", func(_ context.Context, _ payment.CommissionablePayment) error {
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
	if err := processor.HandleSubscriptionEvent(ctx, paymentEvent); err != nil {
		t.Fatal(err)
	}
	if observed != 1 {
		t.Fatalf("observer count = %d, want 1", observed)
	}

	if err := processor.HandleSubscriptionEvent(ctx, paymentEvent); err != nil {
		t.Fatal(err)
	}
	if observed != 1 {
		t.Fatalf("duplicate payment observer count = %d, want 1", observed)
	}

	active := payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     payment.EventSubscriptionActive,
		Data: payment.WebhookEventData{
			SubscriptionID: "sub_1", ProviderProductID: "pdt_subscription", Status: "active", Currency: "USD",
			PeriodStart: periodStartPtr, PeriodEnd: periodEndPtr,
			Metadata: map[string]string{"liki_subscription_key": pending.ID},
		},
	}
	if err := processor.HandleSubscriptionEvent(ctx, active); err != nil {
		t.Fatal(err)
	}
	allowed, err := entitlements.AllowedSkills(ctx, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(allowed) != 1 || allowed[0] != product.SkillConsultation {
		t.Fatalf("allowed = %#v", allowed)
	}
}

func TestSubscriptionEventProcessorRoutesSubscriptionRefund(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	entitlements := entitlement.NewStore(db)
	subscriptions := NewSubscriptionStore(db)
	for _, migrate := range []func(context.Context) error{entitlements.Migrate, subscriptions.Migrate} {
		if err := migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	pending, err := subscriptions.CreatePending(ctx, "user-1", "user@example.com", product.Chat, product.USD)
	if err != nil {
		t.Fatal(err)
	}
	if err := subscriptions.AttachCheckout(ctx, "user-1", pending.ID, "checkout-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE chat_subscriptions SET provider_subscription_id='sub_1' WHERE id=?`, pending.ID); err != nil {
		t.Fatal(err)
	}
	paid := payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     payment.EventPaymentSucceeded,
		Data: payment.WebhookEventData{
			SubscriptionID: "sub_1", PaymentID: "pay_1", Amount: 1990, Currency: "USD",
			Metadata: map[string]string{"liki_user_id": "user-1"},
		},
	}
	processor := NewSubscriptionEventProcessor(subscriptions, entitlements, "pdt_subscription", func(_ context.Context, _ payment.CommissionablePayment) error {
		return nil
	})
	refundSales := make([]payment.CommissionablePayment, 0, 1)
	processor.SetCommissionRefundObserver(func(_ context.Context, _ payment.WebhookEvent, sale payment.CommissionablePayment) error {
		refundSales = append(refundSales, sale)
		return nil
	})
	if err := processor.HandleSubscriptionEvent(ctx, paid); err != nil {
		t.Fatal(err)
	}
	refund := payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     payment.EventPaymentRefunded,
		Data: payment.WebhookEventData{
			PaymentID: "refund_1", RefundID: "refund_1", RefundedPaymentID: "pay_1", Amount: 1990, Currency: "USD",
			Metadata: map[string]string{"liki_user_id": "user-1"},
		},
	}
	if err := processor.HandleSubscriptionEvent(ctx, refund); err != nil {
		t.Fatal(err)
	}
	payments, err := subscriptions.ListPaymentsByUser(ctx, "user-1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(payments) != 1 || payments[0].Status != SubscriptionPaymentRefunded || payments[0].RefundedMinorUnits != 1990 {
		t.Fatalf("payments = %+v", payments)
	}
	if len(refundSales) != 1 || refundSales[0].ID != "pay_1" || refundSales[0].UserID != "user-1" {
		t.Fatalf("refund sales = %+v", refundSales)
	}
}

func TestSubscriptionEventProcessorRejectsUnexpectedProduct(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	entitlements := entitlement.NewStore(db)
	subscriptions := NewSubscriptionStore(db)
	for _, migrate := range []func(context.Context) error{entitlements.Migrate, subscriptions.Migrate} {
		if err := migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	pending, err := subscriptions.CreatePending(ctx, "user-1", "user@example.com", product.Chat, product.USD)
	if err != nil {
		t.Fatal(err)
	}
	processor := NewSubscriptionEventProcessor(subscriptions, entitlements, "pdt_subscription", nil)
	err = processor.HandleSubscriptionEvent(ctx, payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     payment.EventSubscriptionActive,
		Data: payment.WebhookEventData{
			SubscriptionID: "sub_wrong", ProviderProductID: "pdt_other", Status: "active",
			PeriodStart: ptrTime(time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)),
			PeriodEnd:   ptrTime(time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)),
			Metadata:    map[string]string{"liki_subscription_key": pending.ID},
		},
	})
	if err == nil {
		t.Fatal("unexpected product webhook was accepted")
	}
	stored, err := subscriptions.GetByUser(ctx, "user-1", pending.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != SubscriptionPending || stored.ProviderSubscriptionID != "" {
		t.Fatalf("stored = %+v", stored)
	}
	grants, err := entitlements.ListActiveByUser(ctx, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(grants) != 0 {
		t.Fatalf("grants = %+v", grants)
	}
}

func ptrTime(value time.Time) *time.Time { return &value }
