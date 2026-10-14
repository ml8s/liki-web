package partner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ml8s/liki-web/internal/metrics"
)

var (
	ErrInvalidPayoutRequest     = errors.New("partner: invalid payout request")
	ErrNoWithdrawableCommission = errors.New("partner: no withdrawable commission")
	ErrMixedPayoutCurrency      = errors.New("partner: pending commissions have mixed currencies")
	ErrPayoutNotFound           = errors.New("partner: payout not found")
	ErrPayoutInvalidTransition  = errors.New("partner: payout transition is not allowed")
	ErrPayoutMinimumNotMet      = errors.New("partner: payout minimum is not met")
)

func payoutMinimum(currency string) int64 {
	if currency == "USD" {
		return 1000
	}
	return 5000
}

func (s *Store) CreatePayout(ctx context.Context, userID string, method PayoutMethod, account string) (Payout, error) {
	return s.CreatePayoutForRole(ctx, userID, RoleKOL, method, account)
}

func (s *Store) CreatePayoutForRole(ctx context.Context, userID string, role PartnerRole, method PayoutMethod, account string) (Payout, error) {
	userID = strings.TrimSpace(userID)
	method = PayoutMethod(strings.ToLower(strings.TrimSpace(string(method))))
	account = strings.TrimSpace(account)
	role = normalizeRole(role)
	if userID == "" || account == "" || (role != RoleKOL && role != RoleScout) ||
		(method != PayoutMethodAlipay && method != PayoutMethodPayPal) {
		return Payout{}, ErrInvalidPayoutRequest
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Payout{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var referrerID string
	err = tx.QueryRowContext(ctx, `
SELECT r.user_id
FROM partners r
WHERE r.user_id=? AND r.status='active'
  AND EXISTS (SELECT 1 FROM partner_roles pr WHERE pr.user_id=r.user_id AND pr.role=? AND pr.status IN ('active','retired'))`,
		userID, string(role)).Scan(&referrerID)
	if errors.Is(err, sql.ErrNoRows) {
		return Payout{}, ErrPartnerNotFound
	}
	if err != nil {
		return Payout{}, err
	}

	rows, err := tx.QueryContext(ctx, `
SELECT currency,COALESCE(SUM(amount_minor-refunded_minor),0)
FROM partner_commission_entries
WHERE beneficiary_id=? AND beneficiary_role=? AND status=? AND payout_id IS NULL
      AND amount_minor>refunded_minor
GROUP BY currency`, userID, string(role), string(CommissionPending))
	if err != nil {
		return Payout{}, err
	}
	type currencyTotal struct {
		currency string
		amount   int64
	}
	totals := make([]currencyTotal, 0)
	for rows.Next() {
		var item currencyTotal
		if err := rows.Scan(&item.currency, &item.amount); err != nil {
			_ = rows.Close()
			return Payout{}, err
		}
		totals = append(totals, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return Payout{}, err
	}
	_ = rows.Close()
	if len(totals) == 0 {
		return Payout{}, ErrNoWithdrawableCommission
	}
	if len(totals) > 1 {
		return Payout{}, ErrMixedPayoutCurrency
	}
	currency := strings.ToUpper(totals[0].currency)
	amount := totals[0].amount
	if amount <= 0 {
		return Payout{}, ErrNoWithdrawableCommission
	}
	if amount < payoutMinimum(currency) {
		return Payout{}, fmt.Errorf("%w: need %d minor units", ErrPayoutMinimumNotMet, payoutMinimum(currency))
	}

	sealedAccount, err := s.accounts.seal(account)
	if err != nil {
		return Payout{}, err
	}
	now := formatTime(s.nowFunc())
	payout := Payout{
		ID: s.idFunc(), ReferrerID: userID, Role: role,
		AmountMinor: amount, RequestedAmountMinor: amount,
		Currency: currency, Method: method, Account: account,
		Status: PayoutRequested, CreatedAt: parseTime(now), UpdatedAt: parseTime(now),
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO partner_payouts
(id,referrer_id,beneficiary_role,amount_minor,requested_amount_minor,currency,method,account,status,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		payout.ID, payout.ReferrerID, string(payout.Role), payout.AmountMinor,
		payout.RequestedAmountMinor, payout.Currency, string(payout.Method),
		sealedAccount, string(payout.Status), now, now); err != nil {
		return Payout{}, err
	}
	res, err := tx.ExecContext(ctx, `
UPDATE partner_commission_entries SET payout_id=?,updated_at=?
WHERE beneficiary_id=? AND beneficiary_role=? AND status=? AND payout_id IS NULL`,
		payout.ID, now, userID, string(role), string(CommissionPending))
	if err != nil {
		return Payout{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Payout{}, ErrNoWithdrawableCommission
	}
	metrics.Default.Inc("referral_payout_created_total", map[string]string{"method": string(payout.Method)})
	if err := recordReferralActionTx(ctx, tx, s.idFunc(), userID, "payout.requested", "payout", payout.ID,
		"", fmt.Sprintf("amount_minor=%d;role=%s", amount, role), "partner requested payout"); err != nil {
		return Payout{}, err
	}
	if err := tx.Commit(); err != nil {
		return Payout{}, err
	}
	return payout, nil
}

func (s *Store) PayoutByID(ctx context.Context, id string) (Payout, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id,referrer_id,beneficiary_role,amount_minor,requested_amount_minor,currency,
       method,account,status,COALESCE(payment_reference,''),COALESCE(rejection_reason,''),
       created_at,updated_at,paid_at
FROM partner_payouts WHERE id=?`, id)
	return s.scanPayout(row.Scan)
}

func (s *Store) ListPayoutsForUser(ctx context.Context, userID string, limit int) ([]Payout, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id,referrer_id,beneficiary_role,amount_minor,requested_amount_minor,currency,
       method,account,status,COALESCE(payment_reference,''),COALESCE(rejection_reason,''),
       created_at,updated_at,paid_at
FROM partner_payouts WHERE referrer_id=? ORDER BY created_at DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]Payout, 0)
	for rows.Next() {
		item, err := s.scanPayout(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListPayoutsForRole(ctx context.Context, userID string, role PartnerRole, limit int) ([]Payout, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id,referrer_id,beneficiary_role,amount_minor,requested_amount_minor,currency,
       method,account,status,COALESCE(payment_reference,''),COALESCE(rejection_reason,''),
       created_at,updated_at,paid_at
FROM partner_payouts
WHERE referrer_id=? AND beneficiary_role=?
ORDER BY created_at DESC LIMIT ?`, userID, string(normalizeRole(role)), limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]Payout, 0)
	for rows.Next() {
		item, err := s.scanPayout(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListPayouts(ctx context.Context, status PayoutStatus, limit int) ([]Payout, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := `
SELECT id,referrer_id,beneficiary_role,amount_minor,requested_amount_minor,currency,
       method,account,status,COALESCE(payment_reference,''),COALESCE(rejection_reason,''),
       created_at,updated_at,paid_at
FROM partner_payouts`
	args := []any{}
	if status != "" {
		query += ` WHERE status=?`
		args = append(args, string(status))
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]Payout, 0)
	for rows.Next() {
		item, err := s.scanPayout(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) DecidePayout(ctx context.Context, payoutID, actor string, approved bool, reason string) error {
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(reason) == "" {
		return errors.New("partner: actor and reason are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM partner_payouts WHERE id=?`, payoutID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPayoutNotFound
	}
	if err != nil {
		return err
	}
	if status != string(PayoutRequested) {
		return ErrPayoutInvalidTransition
	}
	now := formatTime(s.nowFunc())
	newStatus := PayoutRejected
	if approved {
		newStatus = PayoutApproved
	} else {
		if _, err := tx.ExecContext(ctx, `
UPDATE partner_commission_entries SET payout_id=NULL,updated_at=?
WHERE payout_id=? AND status=?`, now, payoutID, string(CommissionPending)); err != nil {
			return err
		}
	}
	metrics.Default.Inc("referral_payout_state_total", map[string]string{"result": string(newStatus)})
	if _, err := tx.ExecContext(ctx, `
UPDATE partner_payouts SET status=?,rejection_reason=?,updated_at=? WHERE id=?`,
		string(newStatus), reason, now, payoutID); err != nil {
		return err
	}
	if err := recordReferralActionTx(ctx, tx, s.idFunc(), actor, "payout.decided", "payout", payoutID,
		string(PayoutRequested), string(newStatus), reason); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ApprovePayout(ctx context.Context, payoutID, actor string) error {
	return s.DecidePayout(ctx, payoutID, actor, true, "approved")
}

func (s *Store) MarkPayoutPaid(ctx context.Context, payoutID, actor, paymentReference string) error {
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(paymentReference) == "" {
		return errors.New("partner: actor and payment reference are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM partner_payouts WHERE id=?`, payoutID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPayoutNotFound
	}
	if err != nil {
		return err
	}
	if status != string(PayoutApproved) {
		return ErrPayoutInvalidTransition
	}
	now := formatTime(s.nowFunc())
	metrics.Default.Inc("referral_payout_state_total", map[string]string{"result": string(PayoutPaid)})
	if _, err := tx.ExecContext(ctx, `
UPDATE partner_payouts
SET status=?,payment_reference=?,paid_at=?,updated_at=? WHERE id=?`,
		string(PayoutPaid), paymentReference, now, now, payoutID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE partner_commission_entries SET status=?,paid_at=?,updated_at=?
WHERE payout_id=? AND status=?`,
		string(CommissionPaid), now, now, payoutID, string(CommissionPending)); err != nil {
		return err
	}
	if err := recordReferralActionTx(ctx, tx, s.idFunc(), actor, "payout.marked_paid", "payout", payoutID,
		status, string(PayoutPaid), paymentReference); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) scanPayout(scan func(dest ...any) error) (Payout, error) {
	var item Payout
	var role, currency, method, status, createdAt, updatedAt string
	var paidAt sql.NullString
	if err := scan(&item.ID, &item.ReferrerID, &role, &item.AmountMinor,
		&item.RequestedAmountMinor, &currency, &method, &item.Account, &status,
		&item.PaymentReference, &item.RejectionReason, &createdAt, &updatedAt, &paidAt); err != nil {
		return Payout{}, err
	}
	item.Role = PartnerRole(role)
	item.Currency = currency
	item.Method = PayoutMethod(method)
	item.Status = PayoutStatus(status)
	item.CreatedAt = parseTime(createdAt)
	item.UpdatedAt = parseTime(updatedAt)
	if paidAt.Valid {
		t := parseTime(paidAt.String)
		item.PaidAt = &t
	}
	account, err := s.accounts.open(item.Account)
	if err != nil {
		return Payout{}, err
	}
	item.Account = account
	return item, nil
}

func recordReferralActionTx(ctx context.Context, tx *sql.Tx, id, actor, action, targetType, targetID, before, after, reason string) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO partner_admin_actions
(id,actor,action,target_type,target_id,before,after,reason,created_at)
VALUES (?,?,?,?,?,?,?,?,?)`,
		id, actor, action, targetType, targetID, before, string(after), reason, formatTime(time.Now().UTC()))
	if err != nil {
		return fmt.Errorf("partner: audit payout: %w", err)
	}
	return nil
}

// RevealPayoutAccount decrypts a payout account only for an explicitly
// authorized, audited finance operation. Payout listing returns masked values.
func (s *Store) RevealPayoutAccount(ctx context.Context, payoutID, actor, reason string) (string, error) {
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(reason) == "" {
		return "", errors.New("partner: actor and reason are required")
	}
	var sealed string
	err := s.db.QueryRowContext(ctx, `SELECT account FROM partner_payouts WHERE id=?`, payoutID).Scan(&sealed)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrPayoutNotFound
	}
	if err != nil {
		return "", err
	}
	account, err := s.accounts.open(sealed)
	if err != nil {
		return "", err
	}
	if err := s.recordAdminAction(ctx, actor, "payout.account_revealed", "payout", payoutID,
		"masked", "revealed", reason); err != nil {
		return "", err
	}
	return account, nil
}
