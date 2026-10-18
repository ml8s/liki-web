package payment

import (
	"context"
	"testing"
	"time"
)

func TestAnonymizeNotificationsCoversCompensationQueue(t *testing.T) {
	ctx := context.Background()
	db := newRuntimeTestDB(t)
	notifications := NewNotificationStore(db)
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`
INSERT INTO payment_intents(id,business_order_id,user_id,product,provider,status,amount_minor,currency,email,created_at,updated_at)
VALUES('intent-1','order-1','user-1','liki-chat','dodo','paid',1000,'USD','user@example.com',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if err := notifications.EnqueueOnce(ctx, "intent-1", NotificationCustomerReceipt, EmailMessage{
		To: "user@example.com", Subject: "receipt", Body: "<p>receipt</p>",
	}); err != nil {
		t.Fatal(err)
	}
	if err := notifications.EnqueueOnce(ctx, "entitlement:ent-1", NotificationEntitlementCompensation, EmailMessage{
		To: "user@example.com", Subject: "compensation", Body: "<p>compensation</p>",
	}); err != nil {
		t.Fatal(err)
	}

	cleared, err := notifications.AnonymizeUser(ctx, "user-1", "user@example.com", "erased@example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	if cleared != 2 {
		t.Fatalf("cleared=%d, want 2", cleared)
	}
	var email, subject, body string
	if err := db.QueryRow(`
SELECT to_email,subject,body FROM payment_notifications WHERE payment_intent_id='entitlement:ent-1'`).
		Scan(&email, &subject, &body); err != nil {
		t.Fatal(err)
	}
	if email != "erased@example.invalid" || subject != "erased" || body != "" {
		t.Fatalf("compensation notification=%q/%q/%q", email, subject, body)
	}
}
