package hepan

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"liki-web/internal/httputil"
)

// PersonInfo holds one person's birth data.
type PersonInfo struct {
	Name   string `json:"name"`
	Birth  string `json:"birth"`
	City   string `json:"city"`
	Gender string `json:"gender"`
}

// OrderData is the order row returned by GetOrder.
type OrderData struct {
	Product  string `json:"product"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Email    string `json:"email"`
	Provider string `json:"provider"`
	Status   string `json:"status"`
	PersonA  string `json:"person_a"`
	PersonB  string `json:"person_b"`
	Locale   string `json:"locale"`
}

// Store manages hepan orders and reports in its own database.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// ── orders ──

func (s *Store) CreateOrder(ctx context.Context, orderID, product string, amount int64, currency, email, personA, personB, locale string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO orders (id, product, amount, currency, email, person_a, person_b, locale) VALUES (?,?,?,?,?,?,?,?)`,
		orderID, product, amount, currency, email, personA, personB, locale)
	return err
}

func (s *Store) GetOrder(ctx context.Context, orderID string) (*OrderData, error) {
	row := s.db.QueryRowContext(ctx, `SELECT product, amount, currency, email, provider, status, person_a, person_b, locale FROM orders WHERE id=?`, orderID)
	var o OrderData
	err := row.Scan(&o.Product, &o.Amount, &o.Currency, &o.Email, &o.Provider, &o.Status, &o.PersonA, &o.PersonB, &o.Locale)
	if err == sql.ErrNoRows {
		return nil, httputil.ErrOrderNotFound
	}
	if err != nil {
		return nil, httputil.ErrInternal
	}
	return &o, nil
}

func (s *Store) UpdateEmail(ctx context.Context, orderID, email string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE orders SET email=? WHERE id=?`, email, orderID)
	return err
}

func (s *Store) MarkPaidIdempotent(ctx context.Context, orderID, paymentID string) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE orders SET status='paid', paid_at=? WHERE id=? AND status='pending'`,
		time.Now().UTC().Format(time.RFC3339), orderID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected() //nolint:errcheck
	return n > 0, nil
}

// ── reports ──

func (s *Store) SaveReport(ctx context.Context, orderID, content string) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO reports (order_id, content, created_at) VALUES (?,?,?)`,
		orderID, content, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) GetReport(ctx context.Context, orderID string) (string, error) {
	var content string
	err := s.db.QueryRowContext(ctx, `SELECT content FROM reports WHERE order_id=?`, orderID).Scan(&content)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("report not found: %s", orderID)
	}
	return content, err
}

// AnonymizeEmail removes contact and birth-report PII, deletes generated report
// content, and retains only anonymous financial order facts.
func (s *Store) AnonymizeEmail(ctx context.Context, oldEmail, tombstone string) (int64, error) {
	if tombstone == "" {
		return 0, fmt.Errorf("hepan: anonymize requires tombstone")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
DELETE FROM reports
WHERE order_id IN (SELECT id FROM orders WHERE email=? OR (? != '' AND email=?))`,
		oldEmail, oldEmail, oldEmail); err != nil {
		return 0, err
	}
	emptyPerson := `{"name":"","birth":"","city":"","gender":""}`
	res, err := tx.ExecContext(ctx, `
UPDATE orders
SET email=?,person_a=?,person_b=?
WHERE email=? OR (? != '' AND email=?)`,
		tombstone, emptyPerson, emptyPerson, oldEmail, oldEmail, oldEmail)
	if err != nil {
		return 0, err
	}
	count, _ := res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}
