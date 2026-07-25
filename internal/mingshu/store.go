package mingshu

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"liki-web/internal/httputil"
	"liki-web/internal/product"
)

// OrderData is the order row returned by GetOrder.
type OrderData struct {
	ID       string
	Product  string
	Amount   int
	Currency string
	Email    string
	Provider string
	Status   string
	Birth    string
	Locale   string
}

// Store manages mingshu orders and reports in its own database.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	for _, stmt := range strings.Split(Schema, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			slog.Warn("mingshu: schema init", "err", err)
		}
	}
	return &Store{db: db}
}

// ── orders ──

func (s *Store) CreateOrder(ctx context.Context, orderID, product string, amount int, currency, email, birth, locale string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO orders (id, product, amount, currency, email, birth, locale) VALUES (?,?,?,?,?,?,?)`,
		orderID, product, amount, currency, email, birth, locale)
	return err
}

func (s *Store) GetOrder(ctx context.Context, orderID string) (*OrderData, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, product, amount, currency, email, provider, status, birth, locale FROM orders WHERE id=?`, orderID)
	var o OrderData
	err := row.Scan(&o.ID, &o.Product, &o.Amount, &o.Currency, &o.Email, &o.Provider, &o.Status, &o.Birth, &o.Locale)
	if err == sql.ErrNoRows {
		return nil, httputil.ErrOrderNotFound
	}
	if err != nil {
		return nil, httputil.ErrInternal
	}
	return &o, nil
}

func (s *Store) UpdateEmail(ctx context.Context, orderID, email string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE orders SET email=?, updated_at=datetime('now') WHERE id=?`, email, orderID)
	return err
}

func (s *Store) UpdateProvider(ctx context.Context, orderID, provider string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE orders SET provider=?, updated_at=datetime('now') WHERE id=?`, provider, orderID)
	return err
}

func (s *Store) MarkPaidViaCallback(ctx context.Context, orderID string) (bool, string, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE orders SET status='paid', updated_at=datetime('now') WHERE id=? AND status='pending'`, orderID)
	if err != nil {
		return false, "", err
	}
	n, _ := res.RowsAffected() //nolint:errcheck
	if n == 0 {
		return false, "", nil
	}
	var email string
	err = s.db.QueryRowContext(ctx, `SELECT email FROM orders WHERE id=?`, orderID).Scan(&email)
	return true, email, err
}

func (s *Store) MarkPaidIdempotent(ctx context.Context, orderID, paymentID string) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE orders SET status='paid', payment_id=?, updated_at=datetime('now') WHERE id=? AND status='pending'`,
		paymentID, orderID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected() //nolint:errcheck
	return n > 0, nil
}

func (s *Store) SaveBirth(ctx context.Context, orderID, birth string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE orders SET birth=? WHERE id=?`, birth, orderID)
	return err
}

func (s *Store) GetBirth(ctx context.Context, orderID string) (string, error) {
	var birth string
	err := s.db.QueryRowContext(ctx, `SELECT birth FROM orders WHERE id=?`, orderID).Scan(&birth)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("birth not found: %s", orderID)
	}
	return birth, err
}

// ── reports ──

func (s *Store) SaveReport(ctx context.Context, orderID, content string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO reports (order_id, content, created_at) VALUES (?,?,?)`,
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

func (s *Store) GetCheckoutOrder(ctx context.Context, orderID string) (*product.CheckoutOrder, error) {
	o, err := s.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return &product.CheckoutOrder{Status: o.Status, Amount: o.Amount}, nil
}

