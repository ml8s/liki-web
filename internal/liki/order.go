package liki

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrOrderNotFound = errors.New("liki: order not found")

type OrderStatus string

const (
	OrderPending  OrderStatus = "pending"
	OrderPaid     OrderStatus = "paid"
	OrderCanceled OrderStatus = "canceled"
)

type Order struct {
	ID              string      `json:"id"`
	UserID          string      `json:"user_id"`
	Email           string      `json:"email"`
	ProductID       Product     `json:"product_id"`
	AmountMinor     int64       `json:"amount_minor"`
	Currency        string      `json:"currency"`
	Status          OrderStatus `json:"status"`
	PaymentIntentID string      `json:"payment_intent_id"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	PaidAt          *time.Time  `json:"paid_at"`
}

// OrderStore manages the chat_orders table.
type OrderStore struct {
	db      *sql.DB
	nowFunc func() time.Time
	idFunc  func() string
}

func NewOrderStore(db *sql.DB) *OrderStore {
	return &OrderStore{
		db:      db,
		nowFunc: func() time.Time { return time.Now().UTC() },
		idFunc:  uuid.NewString,
	}
}

func (s *OrderStore) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS chat_orders (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    email TEXT NOT NULL,
    product_id TEXT NOT NULL,
    amount_minor INTEGER NOT NULL,
    currency TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending','paid','canceled')),
    payment_intent_id TEXT UNIQUE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    paid_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_chat_orders_user_created
    ON chat_orders(user_id, created_at DESC);
DROP INDEX IF EXISTS idx_chat_orders_user_pending_product;
`)
	return err
}

// AnonymizeUser replaces contact email on transactional commerce records. The
// order and financial amount remain for accounting; the user becomes a
// pseudonymous reference only.
func (s *OrderStore) AnonymizeUser(ctx context.Context, userID, oldEmail, tombstone string) (int64, error) {
	if userID == "" || tombstone == "" {
		return 0, fmt.Errorf("liki: anonymize requires user and tombstone")
	}
	res, err := s.db.ExecContext(ctx, `
UPDATE chat_orders SET email=?
WHERE user_id=? OR (? != '' AND email=?)`, tombstone, userID, oldEmail, oldEmail)
	if err != nil {
		return 0, fmt.Errorf("liki: anonymize orders: %w", err)
	}
	return res.RowsAffected()
}

func (s *OrderStore) CreateOrder(ctx context.Context, userID, email string, productID Product, currency string) (Order, error) {
	userID = strings.TrimSpace(userID)
	email = strings.ToLower(strings.TrimSpace(email))
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if _, err := DefinitionFor(productID); err != nil {
		return Order{}, err
	}
	amount, err := AmountMinor(productID, currency)
	if err != nil {
		return Order{}, err
	}
	if userID == "" || email == "" {
		return Order{}, fmt.Errorf("liki: order requires user and email")
	}

	now := s.nowFunc()
	order := Order{
		ID: s.idFunc(), UserID: userID, Email: email, ProductID: productID,
		AmountMinor: amount, Currency: currency, Status: OrderPending,
		CreatedAt: now, UpdatedAt: now,
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO chat_orders
(id,user_id,email,product_id,amount_minor,currency,status,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?,?)`,
		order.ID, order.UserID, order.Email, string(order.ProductID), order.AmountMinor,
		order.Currency, string(order.Status), order.CreatedAt.Format(time.RFC3339),
		order.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return Order{}, fmt.Errorf("liki: create order: %w", err)
	}
	return order, nil
}

func (s *OrderStore) GetOrderForUser(ctx context.Context, userID, orderID string) (Order, error) {
	order, err := s.getOrder(ctx, orderID)
	if err != nil {
		return Order{}, err
	}
	if order.UserID != userID {
		return Order{}, ErrOrderNotFound
	}
	return order, nil
}

func (s *OrderStore) getOrder(ctx context.Context, orderID string) (Order, error) {
	var order Order
	var productID, status, createdAt, updatedAt string
	var intentID, paidAt sql.NullString
	err := s.db.QueryRowContext(ctx, `
SELECT id,user_id,email,product_id,amount_minor,currency,status,payment_intent_id,created_at,updated_at,paid_at
FROM chat_orders WHERE id=?`, orderID,
	).Scan(&order.ID, &order.UserID, &order.Email, &productID, &order.AmountMinor, &order.Currency,
		&status, &intentID, &createdAt, &updatedAt, &paidAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Order{}, ErrOrderNotFound
	}
	if err != nil {
		return Order{}, err
	}
	order.ProductID = Product(productID)
	order.Status = OrderStatus(status)
	order.PaymentIntentID = intentID.String
	if order.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return Order{}, fmt.Errorf("liki: parse created_at: %w", err)
	}
	if order.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return Order{}, fmt.Errorf("liki: parse updated_at: %w", err)
	}
	if paidAt.Valid {
		parsed, err := time.Parse(time.RFC3339, paidAt.String)
		if err != nil {
			return Order{}, fmt.Errorf("liki: parse paid_at: %w", err)
		}
		order.PaidAt = &parsed
	}
	return order, nil
}

func (s *OrderStore) AttachIntent(ctx context.Context, userID, orderID, intentID string) error {
	result, err := s.db.ExecContext(ctx, `
UPDATE chat_orders SET payment_intent_id=?, updated_at=?
WHERE id=? AND user_id=? AND status='pending'`,
		intentID, s.nowFunc().Format(time.RFC3339), orderID, userID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return ErrOrderNotFound
	}
	return nil
}

func (s *OrderStore) CancelOrder(ctx context.Context, userID, orderID string) error {
	result, err := s.db.ExecContext(ctx, `
UPDATE chat_orders SET status='canceled', updated_at=?
WHERE id=? AND user_id=? AND status='pending'`,
		s.nowFunc().Format(time.RFC3339), orderID, userID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return ErrOrderNotFound
	}
	return nil
}

func (s *OrderStore) MarkPaidByIntent(ctx context.Context, intentID string) (Order, bool, error) {
	now := s.nowFunc().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
UPDATE chat_orders SET status='paid', paid_at=?, updated_at=?
WHERE payment_intent_id=? AND status='pending'`, now, now, intentID)
	if err != nil {
		return Order{}, false, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return Order{}, false, nil
	}

	var order Order
	var productID, status, createdAt, updatedAt string
	var paidAt sql.NullString
	err = s.db.QueryRowContext(ctx, `
SELECT id,user_id,email,product_id,amount_minor,currency,status,payment_intent_id,created_at,updated_at,paid_at
FROM chat_orders WHERE payment_intent_id=?`, intentID,
	).Scan(&order.ID, &order.UserID, &order.Email, &productID, &order.AmountMinor, &order.Currency,
		&status, &order.PaymentIntentID, &createdAt, &updatedAt, &paidAt)
	if err != nil {
		return Order{}, false, err
	}
	order.ProductID = Product(productID)
	order.Status = OrderStatus(status)
	parsed, err := time.Parse(time.RFC3339, paidAt.String)
	if err != nil {
		return Order{}, false, err
	}
	order.PaidAt = &parsed
	if order.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return Order{}, false, err
	}
	if order.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return Order{}, false, err
	}
	return order, true, nil
}

func (s *OrderStore) ListOrdersForUser(ctx context.Context, userID string, limit int) ([]Order, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id,user_id,email,product_id,amount_minor,currency,status,payment_intent_id,created_at,updated_at,paid_at
FROM chat_orders WHERE user_id=?
ORDER BY created_at DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]Order, 0)
	for rows.Next() {
		var order Order
		var productID, status, createdAt, updatedAt string
		var intentID, paidAt sql.NullString
		if err := rows.Scan(&order.ID, &order.UserID, &order.Email, &productID, &order.AmountMinor,
			&order.Currency, &status, &intentID, &createdAt, &updatedAt, &paidAt); err != nil {
			return nil, err
		}
		order.ProductID = Product(productID)
		order.Status = OrderStatus(status)
		order.PaymentIntentID = intentID.String
		var err error
		if order.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
			return nil, err
		}
		if order.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
			return nil, err
		}
		if paidAt.Valid {
			parsed, err := time.Parse(time.RFC3339, paidAt.String)
			if err != nil {
				return nil, err
			}
			order.PaidAt = &parsed
		}
		out = append(out, order)
	}
	return out, rows.Err()
}
