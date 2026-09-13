package payment

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func newEventTestStore(t *testing.T) (*sql.DB, *PaymentEventStore) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "payments.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewPaymentEventStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return db, store
}

func TestPaymentEventRecordIsIdempotent(t *testing.T) {
	ctx := context.Background()
	_, store := newEventTestStore(t)
	event := WebhookEvent{Provider: ProviderXunhu, Type: EventPaymentSucceeded,
		Data: WebhookEventData{OrderID: "pi-1", Amount: 990, Currency: "CNY", PaymentID: "pay-1"}}
	inserted, err := store.Record(ctx, "pi-1", PaymentEventWebhook, event)
	if err != nil || !inserted {
		t.Fatalf("first inserted=%v err=%v", inserted, err)
	}
	inserted, err = store.Record(ctx, "pi-1", PaymentEventWebhook, event)
	if err != nil || inserted {
		t.Fatalf("duplicate inserted=%v err=%v", inserted, err)
	}
	event.Data.Amount = 1000
	inserted, err = store.Record(ctx, "pi-1", PaymentEventWebhook, event)
	if err != nil || !inserted {
		t.Fatalf("changed inserted=%v err=%v", inserted, err)
	}
}
