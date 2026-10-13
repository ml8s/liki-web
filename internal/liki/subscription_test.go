package liki

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestSubscriptionStoreLifecycleIsIdempotent(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	subscriptions := NewSubscriptionStore(db)
	entitlements := NewEntitlementStore(db)
	for _, migrate := range []func(context.Context) error{subscriptions.Migrate, entitlements.Migrate} {
		if err := migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}

	pending, err := subscriptions.CreatePending(ctx, "user-1", "user@example.com", ProductChat, CurrencyUSD)
	if err != nil {
		t.Fatal(err)
	}
	if pending.Status != SubscriptionPending || pending.Provider != payment.ProviderDodo {
		t.Fatalf("pending = %+v", pending)
	}
	if err := subscriptions.AttachCheckout(ctx, "user-1", pending.ID, "checkout-1"); err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	end := start.Add(30 * 24 * time.Hour)
	active := payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     payment.EventSubscriptionActive,
		Data: payment.WebhookEventData{
			SubscriptionID:      "sub_1",
			CustomerID:          "cus_1",
			Product:             string(ProductChat),
			Status:              "active",
			Currency:            "USD",
			PeriodStart:         &start,
			PeriodEnd:           &end,
			NextBillingAt:       &end,
			CancelAtPeriodEnd:   false,
			ProviderPeriodCount: 1,
			Metadata:            map[string]string{"liki_subscription_key": pending.ID},
		},
	}
	if err := subscriptions.ApplyWebhook(ctx, entitlements, active); err != nil {
		t.Fatal(err)
	}
	stored, err := subscriptions.GetByUser(ctx, "user-1", pending.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != SubscriptionActive || stored.ProviderSubscriptionID != "sub_1" {
		t.Fatalf("stored = %+v", stored)
	}
	grants, err := entitlements.ListActiveByUser(ctx, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(grants) != 1 || grants[0].Source != "subscription" || grants[0].SourceID != "sub_1" {
		t.Fatalf("grants = %+v", grants)
	}

	if err := subscriptions.ApplyWebhook(ctx, entitlements, active); err != nil {
		t.Fatalf("repeated active webhook: %v", err)
	}
	grants, err = entitlements.ListActiveByUser(ctx, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(grants) != 1 {
		t.Fatalf("repeated webhook created %d grants, want 1", len(grants))
	}
}

func TestSubscriptionStoreRecordsRenewalPaymentOnce(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	subscriptions := NewSubscriptionStore(db)
	if err := subscriptions.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	pending, err := subscriptions.CreatePending(ctx, "user-1", "user@example.com", ProductChat, CurrencyUSD)
	if err != nil {
		t.Fatal(err)
	}
	if err := subscriptions.AttachCheckout(ctx, "user-1", pending.ID, "checkout-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE chat_subscriptions SET provider_subscription_id='sub_1' WHERE id=?`, pending.ID); err != nil {
		t.Fatal(err)
	}

	paymentEvent := payment.WebhookEvent{
		Provider: payment.ProviderDodo,
		Type:     payment.EventPaymentSucceeded,
		Data: payment.WebhookEventData{
			SubscriptionID: "sub_1",
			PaymentID:      "pay_1",
			Amount:         1990,
			Currency:       "USD",
			Email:          "user@example.com",
			Metadata:       map[string]string{"liki_user_id": "user-1"},
		},
	}
	count := 0
	observer := func(_ context.Context, recorded payment.WebhookEvent) error {
		count++
		if recorded.Data.PaymentID != "pay_1" {
			t.Fatalf("observer payment = %#v", recorded.Data)
		}
		return nil
	}
	if err := subscriptions.RecordPayment(ctx, paymentEvent, observer); err != nil {
		t.Fatal(err)
	}
	if err := subscriptions.RecordPayment(ctx, paymentEvent, observer); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("payment observer count = %d, want 1", count)
	}
	payments, err := subscriptions.ListPaymentsByUser(ctx, "user-1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(payments) != 1 || payments[0].AmountMinorUnits != 1990 {
		t.Fatalf("payments = %+v", payments)
	}
}

func TestSubscriptionCancellationRequiresOwnership(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	subscriptions := NewSubscriptionStore(db)
	if err := subscriptions.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	pending, err := subscriptions.CreatePending(ctx, "user-1", "user@example.com", ProductChat, CurrencyUSD)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
UPDATE chat_subscriptions
SET provider_subscription_id='sub_1', status='active'
WHERE id=?`, pending.ID); err != nil {
		t.Fatal(err)
	}
	gateway := &fakeSubscriptionGateway{}
	if err := subscriptions.SetCancellation(ctx, "other-user", pending.ID, true, gateway); !errors.Is(err, ErrSubscriptionNotFound) {
		t.Fatalf("ownership error = %v, want ErrSubscriptionNotFound", err)
	}
	if gateway.cancelled {
		t.Fatal("provider cancellation called for another user's subscription")
	}
	if err := subscriptions.SetCancellation(ctx, "user-1", pending.ID, true, gateway); err != nil {
		t.Fatal(err)
	}
	if !gateway.cancelled {
		t.Fatal("provider cancellation was not called")
	}
	stored, err := subscriptions.GetByUser(ctx, "user-1", pending.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !stored.CancelAtPeriodEnd {
		t.Fatal("CancelAtPeriodEnd = false, want true")
	}
}

type fakeSubscriptionGateway struct {
	cancelled bool
}

func (g *fakeSubscriptionGateway) SetSubscriptionCancellation(_ context.Context, _ string, cancel bool) error {
	g.cancelled = cancel
	return nil
}
