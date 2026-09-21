package naming

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"liki-web/internal/httputil"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	if db == nil {
		panic("naming: nil db")
	}
	for _, stmt := range strings.Split(Schema, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			slog.Warn("naming: schema init", "err", err)
		}
	}
	return &Store{db: db}
}

// ── orders ──

func (s *Store) CreateOrder(ctx context.Context, orderID, product string, amount int64, currency, email, locale string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO orders (order_id, product, amount, currency, email, locale) VALUES (?,?,?,?,?,?)`,
		orderID, product, amount, currency, email, locale)
	return err
}

func (s *Store) GetOrder(ctx context.Context, orderID string) (*OrderData, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT order_id, product, amount, currency, email, provider, status, locale, chat_expires_at, created_at FROM orders WHERE order_id=?`,
		orderID)
	var o OrderData
	err := row.Scan(&o.OrderID, &o.Product, &o.Amount, &o.Currency, &o.Email, &o.Provider, &o.Status, &o.Locale, &o.ChatExpiresAt, &o.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, httputil.ErrOrderNotFound
	}
	if err != nil {
		return nil, httputil.ErrInternal
	}
	return &o, nil
}

func (s *Store) UpdateEmail(ctx context.Context, orderID, email string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE orders SET email=?, updated_at=datetime('now') WHERE order_id=?`, email, orderID)
	return err
}

func (s *Store) UpdateProvider(ctx context.Context, orderID, provider string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE orders SET provider=?, updated_at=datetime('now') WHERE order_id=?`, provider, orderID)
	return err
}

func (s *Store) MarkPaidIdempotent(ctx context.Context, orderID, paymentID string) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE orders SET status='paid', payment_id=?, updated_at=datetime('now') WHERE order_id=? AND status='pending'`,
		paymentID, orderID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected() //nolint:errcheck
	return n > 0, nil
}

func (s *Store) MarkPaidViaCallback(ctx context.Context, orderID string) (bool, string, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE orders SET status='paid', updated_at=datetime('now') WHERE order_id=? AND status='pending'`, orderID)
	if err != nil {
		return false, "", err
	}
	n, _ := res.RowsAffected() //nolint:errcheck
	if n == 0 {
		return false, "", nil
	}
	var email string
	err = s.db.QueryRowContext(ctx, `SELECT email FROM orders WHERE order_id=?`, orderID).Scan(&email)
	return true, email, err
}

func (s *Store) FindActiveOrdersByEmail(ctx context.Context, email string) ([]OrderData, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT order_id, product, amount, currency, email, provider, status, locale, chat_expires_at, created_at FROM orders WHERE email=? AND status='paid' AND chat_expires_at > datetime('now') ORDER BY created_at DESC`, email)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var orders []OrderData
	for rows.Next() {
		var o OrderData
		if err := rows.Scan(&o.OrderID, &o.Product, &o.Amount, &o.Currency, &o.Email, &o.Provider, &o.Status, &o.Locale, &o.ChatExpiresAt, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (s *Store) ListOrdersByEmail(ctx context.Context, email string) ([]OrderData, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT order_id, product, amount, currency, email, provider, status, locale, chat_expires_at, created_at FROM orders WHERE email=? ORDER BY created_at DESC`, email)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var orders []OrderData
	for rows.Next() {
		var o OrderData
		if err := rows.Scan(&o.OrderID, &o.Product, &o.Amount, &o.Currency, &o.Email, &o.Provider, &o.Status, &o.Locale, &o.ChatExpiresAt, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (s *Store) CreatePaidOrder(ctx context.Context, orderID, product string, amount int64, currency, email, expiresAt, locale string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO orders (order_id, product, amount, currency, email, locale, chat_expires_at, status) VALUES (?,?,?,?,?,?,?,'paid')`,
		orderID, product, amount, currency, email, locale, expiresAt)
	return err
}

func (s *Store) OverrideChatExpiresAt(ctx context.Context, orderID, expiresAt string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE orders SET chat_expires_at=?, updated_at=datetime('now') WHERE order_id=?`, expiresAt, orderID)
	return err
}

// ── chat ──

func (s *Store) CreateChatMessage(ctx context.Context, orderID, role, content string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO chat_messages (order_id, role, content) VALUES (?,?,?)`, orderID, role, content)
	return err
}

func (s *Store) LoadChatHistory(ctx context.Context, orderID string) ([]ChatMessage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, order_id, role, content, created_at FROM chat_messages WHERE order_id=? ORDER BY id`, orderID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var msgs []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.OrderID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (s *Store) DeleteChatHistory(ctx context.Context, orderID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM chat_messages WHERE order_id=?`, orderID)
	return err
}

// ── verification codes ──

func (s *Store) CreateVerificationCode(ctx context.Context, code, email, createdBy string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO verification_codes (code, email, created_by) VALUES (?,?,?)`, code, email, createdBy)
	return err
}

func (s *Store) MarkCodeUsed(ctx context.Context, code string) (string, error) {
	var email string
	err := s.db.QueryRowContext(ctx,
		`UPDATE verification_codes SET status='used', used_at=datetime('now') WHERE code=? AND status='active' RETURNING email`, code).Scan(&email)
	if err != nil {
		return "", fmt.Errorf("code not found or already used: %s", code)
	}
	return email, nil
}

func (s *Store) ListVerificationCodes(ctx context.Context) ([]VerificationCode, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT code, status, order_id, email, created_by, created_at, used_at FROM verification_codes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var codes []VerificationCode
	for rows.Next() {
		var c VerificationCode
		if err := rows.Scan(&c.Code, &c.Status, &c.OrderID, &c.Email, &c.CreatedBy, &c.CreatedAt, &c.UsedAt); err != nil {
			return nil, err
		}
		codes = append(codes, c)
	}
	return codes, rows.Err()
}

func (s *Store) RedeemCode(ctx context.Context, params RedeemCodeParams) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("naming: redeem begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(ctx,
		`INSERT INTO orders (order_id, product, amount, currency, email, status, chat_expires_at, locale)
		 VALUES (?, ?, ?, ?, ?, 'paid', ?, ?)`,
		params.OrderID, params.Product, params.Amount, params.Currency, params.Email, params.ChatExpiresAt, params.Locale)
	if err != nil {
		return fmt.Errorf("naming: redeem create order: %w", err)
	}

	res, err := tx.ExecContext(ctx,
		`UPDATE verification_codes SET status='used', email=?, order_id=?, used_at=datetime('now') WHERE code=? AND status='active'`,
		params.Email, params.OrderID, params.Code)
	if err != nil {
		return fmt.Errorf("naming: redeem update code: %w", err)
	}
	n, _ := res.RowsAffected() //nolint:errcheck
	if n == 0 {
		var status string
		err := tx.QueryRowContext(ctx, `SELECT status FROM verification_codes WHERE code=?`, params.Code).Scan(&status)
		if err == sql.ErrNoRows {
			return fmt.Errorf("code not found: %s", params.Code)
		}
		if err != nil {
			return fmt.Errorf("naming: check code status: %w", err)
		}
		return fmt.Errorf("code status is %s (not active)", status)
	}

	return tx.Commit()
}

// ── maintenance ──

func (s *Store) CleanStale(ctx context.Context) (int, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE orders SET status='expired', updated_at=datetime('now') WHERE status='pending' AND created_at < datetime('now','-7 days')`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected() //nolint:errcheck
	return int(n), nil
}

// AnonymizeEmail pseudonymizes transaction email and deletes generated chat
// content. Paid order facts remain for accounting and tax retention.
func (s *Store) AnonymizeEmail(ctx context.Context, oldEmail, tombstone string) (int64, error) {
	if tombstone == "" {
		return 0, fmt.Errorf("naming: anonymize requires tombstone")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
DELETE FROM chat_messages
WHERE order_id IN (SELECT order_id FROM orders WHERE email=? OR (? != '' AND email=?))`,
		oldEmail, oldEmail, oldEmail); err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `
UPDATE orders SET email=?
WHERE email=? OR (? != '' AND email=?)`, tombstone, oldEmail, oldEmail, oldEmail)
	if err != nil {
		return 0, err
	}
	count, _ := res.RowsAffected()
	if _, err := tx.ExecContext(ctx, `
UPDATE verification_codes SET email=?
WHERE email=? OR (? != '' AND email=?)`, tombstone, oldEmail, oldEmail, oldEmail); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}
