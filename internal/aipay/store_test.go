package aipay

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func newAipayStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	store.nowFun = func() time.Time { return time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC) }
	store.idFun = func() string { return "receipt-1" }
	return store
}

func TestAipayStoreCreatesBillAndIdempotentFulfillment(t *testing.T) {
	ctx := context.Background()
	store := newAipayStore(t)

	bill, err := store.CreateBill(ctx, "1.00", "Liki 服务支持")
	if err != nil {
		t.Fatal(err)
	}
	if bill.Amount != "1.00" || bill.Currency != Currency || bill.ResourceID != ResourceID {
		t.Fatalf("bill = %+v", bill)
	}
	if bill.Status != StatusPending {
		t.Fatalf("status = %q, want pending", bill.Status)
	}

	paid := VerifiedPayment{
		TradeNo:    "trade-1",
		OutTradeNo: bill.OutTradeNo,
		Amount:     "1.00",
		ResourceID: ResourceID,
		Active:     true,
	}
	first, err := store.PrepareFulfillment(ctx, paid, "receipt-1")
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != StatusPaid || first.FulfillmentStatus != FulfillmentPending {
		t.Fatalf("first order state = %s/%s", first.Status, first.FulfillmentStatus)
	}
	if first.ReceiptID != "receipt-1" || first.TradeNo != "trade-1" {
		t.Fatalf("first order = %+v", first)
	}

	if err := store.MarkFulfilled(ctx, bill.OutTradeNo, "trade-1"); err != nil {
		t.Fatal(err)
	}

	second, err := store.PrepareFulfillment(ctx, paid, "should-not-replace")
	if err != nil {
		t.Fatal(err)
	}
	if second.FulfillmentStatus != FulfillmentConfirmed {
		t.Fatalf("second state = %q, want fulfilled", second.FulfillmentStatus)
	}
	if second.ReceiptID != "receipt-1" {
		t.Fatalf("receipt changed: %q", second.ReceiptID)
	}
	if err := store.MarkFulfilled(ctx, bill.OutTradeNo, "trade-1"); !errors.Is(err, ErrFulfillmentInvalid) {
		t.Fatalf("second mark = %v, want ErrFulfillmentInvalid", err)
	}
}

func TestAipayStoreRejectsMismatchedVerifiedPayment(t *testing.T) {
	ctx := context.Background()
	store := newAipayStore(t)
	bill, err := store.CreateBill(ctx, "1.00", "Liki 服务支持")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		paid VerifiedPayment
	}{
		{"unknown order", VerifiedPayment{OutTradeNo: "other", Amount: "1.00", ResourceID: ResourceID}},
		{"amount mismatch", VerifiedPayment{OutTradeNo: bill.OutTradeNo, Amount: "2.00", ResourceID: ResourceID}},
		{"resource mismatch", VerifiedPayment{OutTradeNo: bill.OutTradeNo, Amount: "1.00", ResourceID: "/other"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := store.PrepareFulfillment(ctx, test.paid, "receipt-x"); !errors.Is(err, ErrProofNotAllowed) {
				t.Fatalf("PrepareFulfillment = %v, want ErrProofNotAllowed", err)
			}
		})
	}
}

func TestAipayStoreRejectsExpiredPendingOrder(t *testing.T) {
	ctx := context.Background()
	store := newAipayStore(t)
	bill, err := store.CreateBill(ctx, "1.00", "Liki 服务支持")
	if err != nil {
		t.Fatal(err)
	}

	past := store.nowFun().Add(-time.Minute).Format(time.RFC3339)
	if _, err := store.db.ExecContext(ctx, `UPDATE aipay_orders SET pay_before=? WHERE out_trade_no=?`, past, bill.OutTradeNo); err != nil {
		t.Fatal(err)
	}
	paid := VerifiedPayment{OutTradeNo: bill.OutTradeNo, Amount: "1.00", ResourceID: ResourceID}
	if _, err := store.PrepareFulfillment(ctx, paid, "receipt-x"); !errors.Is(err, ErrProofNotAllowed) {
		t.Fatalf("PrepareFulfillment = %v, want ErrProofNotAllowed", err)
	}
}

func TestAipayStoreCleanupExpiredBills(t *testing.T) {
	ctx := context.Background()
	store := newAipayStore(t)

	// 过期 pending 单：应被清理
	expired, err := store.CreateBill(ctx, "1.00", "Liki 服务支持")
	if err != nil {
		t.Fatal(err)
	}
	past := store.nowFun().Add(-time.Hour).Format(time.RFC3339)
	if _, err := store.db.ExecContext(ctx, `UPDATE aipay_orders SET pay_before=? WHERE out_trade_no=?`, past, expired.OutTradeNo); err != nil {
		t.Fatal(err)
	}
	store.idFun = func() string { return "ot-second" }

	// 未过期 pending 单：应保留
	active, err := store.CreateBill(ctx, "2.00", "Liki 服务支持")
	if err != nil {
		t.Fatal(err)
	}

	n, err := store.CleanupExpiredBills(ctx, store.nowFun())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("cleaned = %d, want 1", n)
	}
	var count int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM aipay_orders WHERE out_trade_no=?`, active.OutTradeNo).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("unexpired bill must survive, got count=%d", count)
	}
}
