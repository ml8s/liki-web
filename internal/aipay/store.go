package aipay

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ml8s/liki-web/internal/money"
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
    amount_minor INTEGER NOT NULL DEFAULT 0,
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
	return s.backfillAmountMinor(ctx)
}

// backfillAmountMinor adds the canonical integer minor-unit column to
// databases created before it existed, then derives it from the provider
// wire string for every row predating the column.
func (s *Store) backfillAmountMinor(ctx context.Context) error {
	var name string
	err := s.db.QueryRowContext(ctx,
		`SELECT name FROM pragma_table_info('aipay_orders') WHERE name='amount_minor'`).Scan(&name)
	if err == nil {
		return s.recomputeAmountMinor(ctx)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("aipay: check amount_minor: %w", err)
	}
	if _, err := s.db.ExecContext(ctx,
		`ALTER TABLE aipay_orders ADD COLUMN amount_minor INTEGER NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("aipay: add amount_minor: %w", err)
	}
	return s.recomputeAmountMinor(ctx)
}

// recomputeAmountMinor derives amount_minor from the provider wire string.
// Alipay amounts are fixed 2-decimal strings, so the split-and-pad parse is
// exact (no float rounding).
func (s *Store) recomputeAmountMinor(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT out_trade_no, amount FROM aipay_orders WHERE amount_minor = 0`)
	if err != nil {
		return fmt.Errorf("aipay: scan amount_minor backlog: %w", err)
	}
	defer func() { _ = rows.Close() }()
	type backlogItem struct {
		outTradeNo string
		minor      int64
	}
	var backlog []backlogItem
	for rows.Next() {
		var outTradeNo, amount string
		if err := rows.Scan(&outTradeNo, &amount); err != nil {
			return fmt.Errorf("aipay: scan amount row: %w", err)
		}
		minor, err := money.ParseDecimalToMinor(amount)
		if err != nil {
			return fmt.Errorf("aipay: backfill %s: %w", outTradeNo, err)
		}
		if minor != 0 {
			backlog = append(backlog, backlogItem{outTradeNo, minor})
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("aipay: iterate amount rows: %w", err)
	}
	for _, item := range backlog {
		if _, err := s.db.ExecContext(ctx,
			`UPDATE aipay_orders SET amount_minor=? WHERE out_trade_no=? AND amount_minor=0`,
			item.minor, item.outTradeNo); err != nil {
			return fmt.Errorf("aipay: backfill update: %w", err)
		}
	}
	return nil
}

func (s *Store) CreateBill(ctx context.Context, amount, goodsName string) (Bill, error) {
	now := s.nowFun()
	minor, err := money.ParseDecimalToMinor(amount)
	if err != nil {
		return Bill{}, err
	}
	bill := Bill{
		OutTradeNo: s.idFun(),
		Amount:     amount,
		Currency:   Currency,
		ResourceID: ResourceID,
		GoodsName:  goodsName,
		// 落库统一 UTC：CleanupExpiredBills 以 UTC 字符串比较 pay_before，
		// 本地时区字符串会让字典序比较失真。
		PayBefore: now.Add(30 * time.Minute).Format(time.RFC3339),
		Status:    StatusPending,
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO aipay_orders
(out_trade_no, amount, amount_minor, currency, resource_id, goods_name, pay_before,
 status, fulfillment_status, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?, 'unfulfilled', ?, ?)`,
		bill.OutTradeNo, bill.Amount, minor, bill.Currency, bill.ResourceID, bill.GoodsName,
		bill.PayBefore, bill.Status, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return Bill{}, fmt.Errorf("aipay: create bill: %w", err)
	}
	return bill, nil
}

// CleanupExpiredBills removes pending bills whose redemption window
// (pay_before, 30 minutes) has passed. Paid orders and the receipt trail
// are financial records and are retained.
func (s *Store) CleanupExpiredBills(ctx context.Context, now time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
DELETE FROM aipay_orders
WHERE status='pending' AND pay_before < ?`, now.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("aipay: cleanup expired bills: %w", err)
	}
	return res.RowsAffected()
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
