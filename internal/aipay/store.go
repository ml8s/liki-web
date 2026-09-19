package aipay

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Store struct {
	db     *sql.DB
	nowFun func() time.Time
	idFun  func() string
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:     db,
		nowFun: func() time.Time { return time.Now().UTC() },
		idFun:  func() string { return uuid.NewString() },
	}
}

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS aipay_orders (
    out_trade_no TEXT PRIMARY KEY,
    trade_no TEXT NOT NULL DEFAULT '',
    amount TEXT NOT NULL,
    currency TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    goods_name TEXT NOT NULL,
    pay_before TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'paid')),
    fulfillment_status TEXT NOT NULL CHECK (fulfillment_status IN ('unfulfilled', 'pending_confirm', 'fulfilled')),
    receipt_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    paid_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_aipay_orders_fulfillment
    ON aipay_orders(fulfillment_status, updated_at);
`)
	if err != nil {
		return fmt.Errorf("aipay: migrate: %w", err)
	}
	return nil
}

func (s *Store) CreateBill(ctx context.Context, amount, goodsName string) (Bill, error) {
	now := s.nowFun()
	bill := Bill{
		OutTradeNo: s.idFun(),
		Amount:     amount,
		Currency:   Currency,
		ResourceID: ResourceID,
		GoodsName:  goodsName,
		PayBefore:  now.Local().Add(30 * time.Minute).Format(time.RFC3339),
		Status:     StatusPending,
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO aipay_orders
(out_trade_no, amount, currency, resource_id, goods_name, pay_before,
 status, fulfillment_status, created_at, updated_at)
VALUES (?,?,?,?,?,?,?, 'unfulfilled', ?, ?)`,
		bill.OutTradeNo, bill.Amount, bill.Currency, bill.ResourceID, bill.GoodsName,
		bill.PayBefore, bill.Status, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return Bill{}, fmt.Errorf("aipay: create bill: %w", err)
	}
	return bill, nil
}

// PrepareFulfillment atomically transitions a verified aipay. Retrying an
// already-confirmed order is idempotent and never replaces its receipt.
func (s *Store) PrepareFulfillment(ctx context.Context, payment VerifiedPayment, receiptID string) (Order, error) {
	now := s.nowFun().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Order{}, fmt.Errorf("aipay: begin fulfillment: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
UPDATE aipay_orders SET
  trade_no=?, status='paid', fulfillment_status='pending_confirm',
  receipt_id=COALESCE(NULLIF(receipt_id,''),?), paid_at=COALESCE(paid_at, ?), updated_at=?
WHERE out_trade_no=? AND amount=? AND currency=? AND resource_id=?
  AND status IN ('pending', 'paid')
  AND fulfillment_status IN ('unfulfilled', 'pending_confirm')
  AND (fulfillment_status='pending_confirm' OR pay_before>?)`,
		payment.TradeNo, receiptID, now, now, payment.OutTradeNo,
		payment.Amount, Currency, payment.ResourceID, now)
	if err != nil {
		return Order{}, fmt.Errorf("aipay: prepare fulfillment: %w", err)
	}

	order, err := scanOrder(func(dest ...any) error {
		return tx.QueryRowContext(ctx, `
SELECT out_trade_no, trade_no, amount, currency, resource_id, goods_name,
       pay_before, status, fulfillment_status, receipt_id, paid_at
FROM aipay_orders WHERE out_trade_no=?`, payment.OutTradeNo).Scan(dest...)
	})
	if errors.Is(err, ErrOrderNotFound) {
		return Order{}, ErrProofNotAllowed
	}
	if err != nil {
		return Order{}, err
	}
	if err := tx.Commit(); err != nil {
		return Order{}, fmt.Errorf("aipay: commit fulfillment: %w", err)
	}
	if order.Status != StatusPaid || order.TradeNo != payment.TradeNo ||
		(order.FulfillmentStatus != FulfillmentPending && order.FulfillmentStatus != FulfillmentConfirmed) {
		return Order{}, ErrProofNotAllowed
	}
	return order, nil
}

func (s *Store) MarkFulfilled(ctx context.Context, outTradeNo, tradeNo string) error {
	now := s.nowFun().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
UPDATE aipay_orders SET fulfillment_status='fulfilled', updated_at=?
WHERE out_trade_no=? AND trade_no=? AND status='paid' AND fulfillment_status='pending_confirm'`,
		now, outTradeNo, tradeNo)
	if err != nil {
		return fmt.Errorf("aipay: mark fulfilled: %w", err)
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return ErrFulfillmentInvalid
	}
	return nil
}

func scanOrder(scan func(dest ...any) error) (Order, error) {
	var order Order
	var paidAt sql.NullString
	err := scan(&order.OutTradeNo, &order.TradeNo, &order.Amount, &order.Currency,
		&order.ResourceID, &order.GoodsName, &order.PayBefore, &order.Status,
		&order.FulfillmentStatus, &order.ReceiptID, &paidAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Order{}, ErrOrderNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("aipay: scan order: %w", err)
	}
	order.PaidAt = paidAt.String
	return order, nil
}
