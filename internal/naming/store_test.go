package naming

import (
	"context"
	"database/sql"
"testing"

	_ "modernc.org/sqlite"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewStore(db)
}

func seedOrder(t *testing.T, s *Store, orderID, email string) {
	t.Helper()
	err := s.CreateOrder(context.Background(), orderID, "naming", 19900, "CNY", email, "zh-Hans")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}

// ── CreateOrder / GetOrder ──

func TestStore_CreateAndGetOrder(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedOrder(t, s, "order-1", "a@b.co")

	o, err := s.GetOrder(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if o.Product != "naming" {
		t.Errorf("product = %q", o.Product)
	}
	if o.Amount != 19900 {
		t.Errorf("amount = %d", o.Amount)
	}
	if o.Status != "pending" {
		t.Errorf("status = %q", o.Status)
	}
	if o.Email != "a@b.co" {
		t.Errorf("email = %q", o.Email)
	}
}

func TestStore_GetOrder_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.GetOrder(context.Background(), "no-such-order")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// ── UpdateEmail / UpdateProvider ──

func TestStore_UpdateEmail(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedOrder(t, s, "order-1", "old@b.co")

	s.UpdateEmail(context.Background(), "order-1", "new@b.co") //nolint:errcheck
	o, _ := s.GetOrder(context.Background(), "order-1") //nolint:errcheck
	if o.Email != "new@b.co" {
		t.Errorf("email = %q", o.Email)
	}
}

func TestStore_UpdateProvider(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedOrder(t, s, "order-1", "a@b.co")

	s.UpdateProvider(context.Background(), "order-1", "xunhu") //nolint:errcheck
	o, _ := s.GetOrder(context.Background(), "order-1") //nolint:errcheck
	if o.Provider != "xunhu" {
		t.Errorf("provider = %q", o.Provider)
	}
}

// ── MarkPaidIdempotent ──

func TestStore_MarkPaid(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedOrder(t, s, "order-1", "a@b.co")

	paid, err := s.MarkPaidIdempotent(context.Background(), "order-1", "pay-1")
	if err != nil {
		t.Fatalf("MarkPaid: %v", err)
	}
	if !paid {
		t.Error("expected paid=true")
	}
	o, _ := s.GetOrder(context.Background(), "order-1") //nolint:errcheck
	if o.Status != "paid" {
		t.Errorf("status = %q", o.Status)
	}
}

func TestStore_MarkPaid_Idempotent(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedOrder(t, s, "order-1", "a@b.co")

	s.MarkPaidIdempotent(context.Background(), "order-1", "pay-1") //nolint:errcheck
	paid, _ := s.MarkPaidIdempotent(context.Background(), "order-1", "pay-1") //nolint:errcheck
	if paid {
		t.Error("expected paid=false for duplicate")
	}
}

// ── MarkPaidViaCallback ──

func TestStore_MarkPaidViaCallback(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedOrder(t, s, "order-1", "a@b.co")

	newPayment, email, err := s.MarkPaidViaCallback(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("MarkPaidViaCallback: %v", err)
	}
	if !newPayment {
		t.Error("expected newPayment=true")
	}
	if email != "a@b.co" {
		t.Errorf("email = %q", email)
	}
}

// ── FindActiveOrdersByEmail ──

func TestStore_FindActiveOrders(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedOrder(t, s, "o1", "a@b.co")
	s.MarkPaidIdempotent(context.Background(), "o1", "pay-1") //nolint:errcheck
	s.OverrideChatExpiresAt(context.Background(), "o1", "2099-12-31") //nolint:errcheck
	seedOrder(t, s, "o2", "a@b.co") // pending

	orders, err := s.FindActiveOrdersByEmail(context.Background(), "a@b.co")
	if err != nil {
		t.Fatalf("FindActiveOrders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 paid order, got %d", len(orders))
	}
	if orders[0].OrderID != "o1" {
		t.Errorf("order_id = %q", orders[0].OrderID)
	}
}

// ── Chat messages ──

func TestStore_ChatLifecycle(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedOrder(t, s, "o1", "a@b.co")
	s.MarkPaidIdempotent(context.Background(), "o1", "pay-1") //nolint:errcheck

	if err := s.CreateChatMessage(context.Background(), "o1", "user", "hi"); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	if err := s.CreateChatMessage(context.Background(), "o1", "assistant", "hello"); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}

	msgs, err := s.LoadChatHistory(context.Background(), "o1")
	if err != nil {
		t.Fatalf("LoadChatHistory: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "hi" {
		t.Errorf("msg[0] = %+v", msgs[0])
	}

	if err := s.DeleteChatHistory(context.Background(), "o1"); err != nil {
		t.Fatalf("DeleteChatHistory: %v", err)
	}
	msgs, _ = s.LoadChatHistory(context.Background(), "o1") //nolint:errcheck
	if len(msgs) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(msgs))
	}
}

// ── Verification codes ──

func TestStore_VerificationCodes(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	if err := s.CreateVerificationCode(context.Background(), "ABC123", "a@b.co", "admin"); err != nil {
		t.Fatalf("CreateCode: %v", err)
	}

	codes, err := s.ListVerificationCodes(context.Background())
	if err != nil {
		t.Fatalf("ListCodes: %v", err)
	}
	if len(codes) != 1 || codes[0].Code != "ABC123" {
		t.Errorf("codes = %+v", codes)
	}

	email, err := s.MarkCodeUsed(context.Background(), "ABC123")
	if err != nil {
		t.Fatalf("MarkCodeUsed: %v", err)
	}
	if email != "a@b.co" {
		t.Errorf("email = %q", email)
	}

	// Used code should fail
	_, err = s.MarkCodeUsed(context.Background(), "ABC123")
	if err == nil {
		t.Error("expected error for used code")
	}
}

// ── CreatePaidOrder ──

func TestStore_CreatePaidOrder(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	if err := s.CreatePaidOrder(context.Background(), "o1", "naming", 19900, "CNY", "a@b.co", "2099-12-31", "zh-Hans"); err != nil {
		t.Fatalf("CreatePaidOrder: %v", err)
	}

	o, _ := s.GetOrder(context.Background(), "o1") //nolint:errcheck
	if o.Status != "paid" {
		t.Errorf("status = %q", o.Status)
	}
	if o.ChatExpiresAt != "2099-12-31" {
		t.Errorf("chat_expires_at = %q", o.ChatExpiresAt)
	}
}

// ── ListOrdersByEmail ──

func TestStore_ListOrdersByEmail(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedOrder(t, s, "o1", "a@b.co")
	seedOrder(t, s, "o2", "a@b.co")

	orders, err := s.ListOrdersByEmail(context.Background(), "a@b.co")
	if err != nil {
		t.Fatalf("ListOrdersByEmail: %v", err)
	}
	if len(orders) != 2 {
		t.Errorf("expected 2, got %d", len(orders))
	}
}

// ── OverrideChatExpiresAt ──

func TestStore_OverrideChatExpiresAt(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedOrder(t, s, "o1", "a@b.co")

	if err := s.OverrideChatExpiresAt(context.Background(), "o1", "2026-12-31"); err != nil {
		t.Fatalf("OverrideChatExpiresAt: %v", err)
	}
	o, _ := s.GetOrder(context.Background(), "o1") //nolint:errcheck
	if o.ChatExpiresAt != "2026-12-31" {
		t.Errorf("chat_expires_at = %q", o.ChatExpiresAt)
	}
}

// ── CleanStale ──

func TestStore_CleanStale_MarksOldPendingAsExpired(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedOrder(t, s, "o1", "a@b.co")

	// Manually set created_at to 8 days ago
	s.db.ExecContext(context.Background(), //nolint:errcheck
		"UPDATE orders SET created_at=datetime('now','-8 days') WHERE order_id=?", "o1")

	n, err := s.CleanStale(context.Background())
	if err != nil {
		t.Fatalf("CleanStale: %v", err)
	}
	if n == 0 {
		t.Error("expected at least 1 stale order cleaned")
	}
}

// ── Edge: nil db ──

func TestStore_NewStore_NilDB(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil db")
		}
	}()
	NewStore(nil)
}


// ── GenerateVerificationCode ──

func TestStore_GenerateVerificationCode(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	code, err := s.GenerateVerificationCode(context.Background(), "admin")
	if err != nil {
		t.Fatalf("GenerateVerificationCode: %v", err)
	}
	if len(code) != 8 {
		t.Errorf("code length = %d, want 8", len(code))
	}

	// Second should succeed with different code
	code2, err := s.GenerateVerificationCode(context.Background(), "admin")
	if err != nil {
		t.Fatalf("second GenerateVerificationCode: %v", err)
	}
	if code == code2 {
		t.Error("expected different codes")
	}
}

func TestStore_BatchGenerateVerificationCodes(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	codes, err := s.BatchGenerateVerificationCodes(context.Background(), 5, "admin")
	if err != nil {
		t.Fatalf("BatchGenerate: %v", err)
	}
	if len(codes) != 5 {
		t.Fatalf("expected 5 codes, got %d", len(codes))
	}
	seen := make(map[string]bool)
	for _, c := range codes {
		if seen[c] {
			t.Errorf("duplicate code: %s", c)
		}
		seen[c] = true
		if len(c) != 8 {
			t.Errorf("code length = %d", len(c))
		}
	}
}

// ── RedeemCode ──

func TestStore_RedeemCode_Success(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	s.CreateVerificationCode(context.Background(), "TEST-CODE", "a@b.co", "admin") //nolint:errcheck

	err := s.RedeemCode(context.Background(), RedeemCodeParams{
		Code:          "TEST-CODE",
		Email:         "a@b.co",
		OrderID:       "order-1",
		Product:       "naming",
		Currency:      "CNY",
		Amount:        19900,
		ChatExpiresAt: "2099-12-31",
		Locale:        "zh-Hans",
	})
	if err != nil {
		t.Fatalf("RedeemCode: %v", err)
	}

	o, err := s.GetOrder(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if o.Status != "paid" {
		t.Errorf("status = %q, want paid", o.Status)
	}
	if o.Email != "a@b.co" {
		t.Errorf("email = %q", o.Email)
	}
}

func TestStore_RedeemCode_CodeNotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.RedeemCode(context.Background(), RedeemCodeParams{
		Code:    "NONEXISTENT",
		OrderID: "order-1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStore_RedeemCode_CodeAlreadyUsed(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	s.CreateVerificationCode(context.Background(), "USED-CODE", "a@b.co", "admin") //nolint:errcheck
	s.MarkCodeUsed(context.Background(), "USED-CODE") //nolint:errcheck

	err := s.RedeemCode(context.Background(), RedeemCodeParams{
		Code:    "USED-CODE",
		OrderID: "order-1",
	})
	if err == nil {
		t.Fatal("expected error for used code")
	}
}
