package mingshu

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ml8s/liki-web/internal/httputil"
)

// OrderData is the order row returned by GetOrder.
type OrderData struct {
	ID       string
	Product  string
	Amount   int64
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
	if _, err := db.Exec(`UPDATE orders SET product='mingshu' WHERE product='lifebook'`); err != nil {
		slog.Warn("mingshu: normalize product IDs", "err", err)
	}
	return &Store{db: db}
}

// ── orders ──

func (s *Store) CreateOrder(ctx context.Context, orderID, product string, amount int64, currency, email, birth, locale string) error {
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

// AnonymizeEmail removes contact and divination-input PII, deletes generated
// report content, and retains only anonymous financial order facts.
func (s *Store) AnonymizeEmail(ctx context.Context, oldEmail, tombstone string) (int64, error) {
	if tombstone == "" {
		return 0, fmt.Errorf("mingshu: anonymize requires tombstone")
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
	res, err := tx.ExecContext(ctx, `
UPDATE orders SET email=?,birth='',updated_at=datetime('now')
WHERE email=? OR (? != '' AND email=?)`, tombstone, oldEmail, oldEmail, oldEmail)
	if err != nil {
		return 0, err
	}
	count, _ := res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}
