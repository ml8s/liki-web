package account

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sales"
	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func TestErasureAnonymizesSubscriptionContact(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "liki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	subscriptions := sales.NewSubscriptionStore(db)
	if err := subscriptions.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	pending, err := subscriptions.CreatePending(ctx, "user-1", "user@example.com", product.Chat, product.USD)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
UPDATE chat_subscriptions
SET provider_subscription_id='sub_1', status='active'
WHERE id=?`, pending.ID); err != nil {
		t.Fatal(err)
	}
	paymentEvent := "not-used"
	_ = paymentEvent
	if _, err := db.Exec(`
INSERT INTO chat_subscription_payments
(id,subscription_id,user_id,product_id,provider,provider_payment_id,amount_minor_units,currency_code,status,created_at,updated_at)
VALUES ('payment-1',?,'user-1','liki-chat','dodo','pay_1',1990,'USD','paid','2026-10-13T00:00:00Z','2026-10-13T00:00:00Z')`,
		pending.ID); err != nil {
		t.Fatal(err)
	}

	// The handler only needs the subscription erasure domain for this test;
	// other stores are covered by TestHandleErasure.
	handler := &Handler{}
	handler.SetSubscriptionEraser(subscriptions)
	cleared, err := handler.eraseSubscriptions(ctx, "user-1", "user@example.com", "erased-user@example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	if cleared != 1 {
		t.Fatalf("cleared = %d, want 1", cleared)
	}
	var email string
	if err := db.QueryRow(`SELECT email FROM chat_subscriptions WHERE id=?`, pending.ID).Scan(&email); err != nil {
		t.Fatal(err)
	}
	if email != "erased-user@example.invalid" {
		t.Fatalf("email = %q", email)
	}
}
