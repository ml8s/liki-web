package referral

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ml8s/liki-web/internal/metrics"
	"github.com/ml8s/liki-web/internal/payment"
)

// CommissionRefundIntake is the durable hand-off from the payment domain to
// the referral ledger. It prevents webhook retries from applying money twice
// and lets a refund arrive before the commission worker settles the sale.
type CommissionRefundIntake struct {
	ID               string
	Provider         string
	PaymentID        string
	ProviderRefundID string
	AmountMinor      int64
	Currency         string
	Status           IntakeStatus
	SkipReason       string
	Attempts         int
	RunAt            time.Time
	LastError        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// EnqueueCommissionRefund records a verified payment refund fact exactly once.
func (s *Store) EnqueueCommissionRefund(ctx context.Context, event payment.WebhookEvent, sale payment.CommissionablePayment) error {
	if event.Type != payment.EventPaymentRefunded {
		return fmt.Errorf("referral: unexpected refund event %q", event.Type)
	}
	if sale.Provider != event.Provider {
		return fmt.Errorf("referral: refund provider mismatch: sale %q event %q", sale.Provider, event.Provider)
	}
	refundID := event.Data.RefundReference()
	if sale.ID == "" || refundID == "" || event.Data.Amount <= 0 {
		return fmt.Errorf("referral: invalid refund payload for sale %q", sale.ID)
	}
	amount := event.Data.Amount
	currency := strings.ToUpper(event.Data.Currency)
	if currency == "" {
		currency = strings.ToUpper(sale.Currency)
	}
	now := formatTime(s.nowFunc())
	id := string(event.Provider) + ":" + event.Data.PaymentID
	res, err := s.db.ExecContext(ctx, `
INSERT INTO commission_refund_intents
(id,payment_provider,payment_intent_id,provider_refund_id,amount_minor,currency,
 status,attempts,run_at,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,0,?,?,?)
ON CONFLICT(payment_provider,provider_refund_id) DO NOTHING`,
		id, string(event.Provider), sale.ID, refundID, amount, currency,
		string(IntakeQueued), now, now, now)
	if err != nil {
		return fmt.Errorf("referral: enqueue refund intent: %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		metrics.Default.Inc("referral_commission_refund_intake_total", map[string]string{"status": string(IntakeQueued)})
	}
	return nil
}

func (s *Store) claimCommissionRefund(ctx context.Context, now time.Time) (CommissionRefundIntake, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CommissionRefundIntake{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var item CommissionRefundIntake
	var provider, paymentID, refundID, currency string
	var amount, attempts int64
	var status, runAt, createdAt, updatedAt string
	var skipReason, lastError sql.NullString
	err = tx.QueryRowContext(ctx, `
SELECT id,payment_provider,payment_intent_id,provider_refund_id,amount_minor,currency,
       status,attempts,run_at,skip_reason,last_error,created_at,updated_at
FROM commission_refund_intents
WHERE status=? AND run_at<=?
ORDER BY run_at LIMIT 1`, string(IntakeQueued), formatTime(now)).Scan(
		&item.ID, &provider, &paymentID, &refundID, &amount, &currency,
		&status, &attempts, &runAt, &skipReason, &lastError, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return CommissionRefundIntake{}, ErrNoRunnableCommissionIntake
	}
	if err != nil {
		return CommissionRefundIntake{}, err
	}
	res, err := tx.ExecContext(ctx, `
UPDATE commission_refund_intents SET status=?,attempts=attempts+1,updated_at=?
WHERE id=? AND status=?`, string(IntakeRunning), formatTime(now), item.ID, string(IntakeQueued))
	if err != nil {
		return CommissionRefundIntake{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return CommissionRefundIntake{}, ErrNoRunnableCommissionIntake
	}
	if err := tx.Commit(); err != nil {
		return CommissionRefundIntake{}, err
	}
	return CommissionRefundIntake{
		ID: item.ID, Provider: provider, PaymentID: paymentID,
		ProviderRefundID: refundID, AmountMinor: amount, Currency: currency,
		Status: IntakeRunning, SkipReason: skipReason.String, Attempts: int(attempts) + 1,
		RunAt: parseTime(runAt), LastError: lastError.String,
		CreatedAt: parseTime(createdAt), UpdatedAt: now,
	}, nil
}

// ProcessCommissionRefunds applies durable refund facts to beneficiary ledger
// entries. Missing commissions are retried; impossible money states fail loudly.
func (s *Store) ProcessCommissionRefunds(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 1
	}
	processed := 0
	for processed < limit {
		item, err := s.claimCommissionRefund(ctx, s.nowFunc())
		if errors.Is(err, ErrNoRunnableCommissionIntake) {
			return processed, nil
		}
		if err != nil {
			return processed, err
		}
		err = s.applyCommissionRefund(ctx, item)
		if err == nil {
			if markErr := s.markCommissionRefundSucceeded(ctx, item.ID); markErr != nil {
				return processed, markErr
			}
			metrics.Default.Inc("referral_commission_refund_applied_total", nil)
			processed++
			continue
		}
		if errors.Is(err, ErrReferralCommissionNotFound) && item.Attempts < maxIntakeAttempts {
			if markErr := s.retryCommissionRefund(ctx, item.ID, item.Attempts, err); markErr != nil {
				return processed, markErr
			}
			continue
		}
		if errors.Is(err, ErrReferralCommissionNotFound) {
			if markErr := s.skipCommissionRefund(ctx, item.ID, "commission_not_found"); markErr != nil {
				return processed, markErr
			}
			continue
		}
		if markErr := s.failCommissionRefund(ctx, item.ID, err); markErr != nil {
			return processed, markErr
		}
		slog.Warn("referral: commission refund retry", "id", item.ID, "intent", item.PaymentID, "err", err)
		metrics.Default.Inc("referral_commission_refund_worker_total", map[string]string{"result": "retry_or_failed"})
	}
	return processed, nil
}

func (s *Store) applyCommissionRefund(ctx context.Context, item CommissionRefundIntake) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var applied int64
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(1) FROM commission_entry_refunds WHERE provider_refund_id=?`,
		item.ProviderRefundID).Scan(&applied); err != nil {
		return err
	}
	if applied > 0 {
		return nil
	}

	var commissionID string
	var saleAmount, alreadyRefunded int64
	var currency, status string
	err = tx.QueryRowContext(ctx, `
SELECT id,amount_minor,refunded_minor,currency,status
FROM referral_commissions
WHERE payment_provider=? AND payment_intent_id=?`,
		item.Provider, item.PaymentID).Scan(
		&commissionID, &saleAmount, &alreadyRefunded, &currency, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrReferralCommissionNotFound
	}
	if err != nil {
		return err
	}
	if currency != item.Currency {
		return fmt.Errorf("referral: refund currency mismatch for %s/%s: got %s commission %s",
			item.Provider, item.PaymentID, item.Currency, currency)
	}
	if item.AmountMinor > saleAmount-alreadyRefunded {
		return fmt.Errorf("referral: refund exceeds remaining sale amount: refund %d remaining %d",
			item.AmountMinor, saleAmount-alreadyRefunded)
	}

	saleRemaining := saleAmount - alreadyRefunded

	type entryDelta struct {
		id, payoutID, beneficiaryID string
		role                        PartnerRole
		status                      CommissionStatus
		amount, refunded, delta     int64
	}
	rows, err := tx.QueryContext(ctx, `
SELECT id,COALESCE(payout_id,''),beneficiary_id,beneficiary_role,status,amount_minor,refunded_minor
FROM referral_commission_entries
WHERE commission_id=? ORDER BY beneficiary_role`, commissionID)
	if err != nil {
		return err
	}
	entries := make([]entryDelta, 0)
	for rows.Next() {
		var entry entryDelta
		if err := rows.Scan(&entry.id, &entry.payoutID, &entry.beneficiaryID, &entry.role,
			&entry.status, &entry.amount, &entry.refunded); err != nil {
			_ = rows.Close()
			return err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	_ = rows.Close()

	// Allocate against remaining balances, so a sequence of partial refunds
	// closes the sale exactly and cannot overdraw a beneficiary share.
	for i := range entries {
		entry := &entries[i]
		entryRemaining := entry.amount - entry.refunded
		if item.AmountMinor == saleRemaining {
			entry.delta = entryRemaining
		} else {
			numerator := entryRemaining * item.AmountMinor * 2
			entry.delta = (numerator/saleRemaining + 1) / 2
			entry.delta = min(entry.delta, entryRemaining)
		}
	}
	// Per-entry rounding is deliberate: entries cover only beneficiary
	// shares — the remainder of a partial refund belongs to the buyer and is
	// tracked by the sale-level refunded counter, so sum(deltas) does not
	// equal the refund amount.
	for _, entry := range entries {
		if entry.delta <= 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE referral_commission_entries
SET refunded_minor=refunded_minor+?,
    status=CASE
      WHEN status=? AND refunded_minor+?>=amount_minor THEN ?
      ELSE status
    END,
    updated_at=?
WHERE id=?`,
			entry.delta, string(CommissionPending), entry.delta, string(CommissionVoided),
			formatTime(s.nowFunc()), entry.id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO commission_entry_refunds
(id,entry_id,provider_refund_id,amount_minor,created_at)
VALUES (?,?,?,?,?)`,
			s.idFunc(), entry.id, item.ProviderRefundID, entry.delta,
			formatTime(s.nowFunc())); err != nil {
			return err
		}
		// Refunding a payable entry is automatic only while it remains pending.
		// Once the beneficiary has been paid, clawback is a human finance action.
		if entry.status == CommissionPaid {
			if err := s.ensureFinanceTaskTx(ctx, tx, item, entry.id, entry.beneficiaryID,
				entry.role, entry.delta); err != nil {
				return err
			}
		}
	}
	newSaleRefund := alreadyRefunded + item.AmountMinor
	sourceStatus := status
	if newSaleRefund >= saleAmount && status == string(CommissionPending) {
		sourceStatus = string(CommissionVoided)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE referral_commissions SET refunded_minor=?,status=?,updated_at=? WHERE id=?`,
		newSaleRefund, sourceStatus, formatTime(s.nowFunc()), commissionID); err != nil {
		return err
	}
	payoutIDs := make([]string, 0)
	for _, entry := range entries {
		if entry.payoutID != "" {
			payoutIDs = append(payoutIDs, entry.payoutID)
		}
	}
	for _, payoutID := range payoutIDs {
		if err := recalculatePayoutAfterRefundTx(ctx, tx, payoutID); err != nil {
			return err
		}
	}
	if err := recordReferralActionTx(ctx, tx, s.idFunc(), "system", "commission.refund_applied", "commission",
		commissionID, fmt.Sprintf("refunded_minor=%d", alreadyRefunded),
		fmt.Sprintf("refunded_minor=%d;refund_id=%s", newSaleRefund, item.ProviderRefundID), "payment.refunded"); err != nil {
		return err
	}
	return tx.Commit()
}

func recalculatePayoutAfterRefundTx(ctx context.Context, tx *sql.Tx, payoutID string) error {
	var status string
	var before int64
	if err := tx.QueryRowContext(ctx, `SELECT status,amount_minor FROM referral_payouts WHERE id=?`, payoutID).Scan(&status, &before); err != nil {
		return err
	}
	if status != string(PayoutRequested) && status != string(PayoutApproved) {
		return nil
	}
	var currency string
	var total int64
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(SUM(amount_minor-refunded_minor),0),COALESCE(MAX(currency),'')
FROM referral_commission_entries WHERE payout_id=?`, payoutID).Scan(&total, &currency); err != nil {
		return err
	}
	now := formatTime(time.Now().UTC())
	minimum := payoutMinimum(currency)
	if total <= 0 || total < minimum {
		if _, err := tx.ExecContext(ctx, `
UPDATE referral_commission_entries SET payout_id=NULL,updated_at=? WHERE payout_id=?`, now, payoutID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE referral_payouts SET status=?,rejection_reason=?,updated_at=? WHERE id=?`,
			string(PayoutRejected), "payment.refunded", now, payoutID); err != nil {
			return err
		}
		if err := recordReferralActionTx(ctx, tx, "system-"+payoutID, "system", "payout.refund_rejected",
			"payout", payoutID, status, string(PayoutRejected), "payment.refunded"); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO referral_payout_adjustments
(id,payout_id,before_minor,after_minor,reason,actor,created_at)
VALUES (?,?,?,?,?,?,?)`,
			fmt.Sprintf("adj-%s-%d", payoutID, time.Now().UnixNano()), payoutID, before, total,
			"payment.refunded", "system", now); err != nil {
			return err
		}
		return nil
	}
	if total == before {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE referral_payouts SET amount_minor=?,updated_at=? WHERE id=?`, total, now, payoutID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO referral_payout_adjustments
(id,payout_id,before_minor,after_minor,reason,actor,created_at)
VALUES (?,?,?,?,?,?,?)`,
		fmt.Sprintf("adj-%s-%d", payoutID, time.Now().UnixNano()), payoutID, before, total, "payment.refunded", "system", now)
	return err
}

func (s *Store) markCommissionRefundSucceeded(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE commission_refund_intents SET status=?,last_error=NULL,updated_at=? WHERE id=?`,
		string(IntakeSuccess), formatTime(s.nowFunc()), id)
	return err
}

func (s *Store) retryCommissionRefund(ctx context.Context, id string, attempt int, failure error) error {
	delays := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, 4 * time.Hour}
	runAt := s.nowFunc().Add(delays[min(attempt, len(delays)-1)])
	_, err := s.db.ExecContext(ctx, `
UPDATE commission_refund_intents SET status=?,run_at=?,last_error=?,updated_at=? WHERE id=?`,
		string(IntakeQueued), formatTime(runAt), failure.Error(), formatTime(s.nowFunc()), id)
	return err
}

func (s *Store) skipCommissionRefund(ctx context.Context, id, reason string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE commission_refund_intents SET status=?,skip_reason=?,updated_at=? WHERE id=?`,
		string(IntakeSkipped), reason, formatTime(s.nowFunc()), id)
	return err
}

func (s *Store) failCommissionRefund(ctx context.Context, id string, failure error) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE commission_refund_intents SET status=?,last_error=?,updated_at=? WHERE id=?`,
		string(IntakeFailed), failure.Error(), formatTime(s.nowFunc()), id)
	return err
}

var ErrReferralCommissionNotFound = errors.New("referral: commission not found")

func (s *Store) ensureFinanceTaskTx(
	ctx context.Context,
	tx *sql.Tx,
	item CommissionRefundIntake,
	entryID, beneficiaryID string,
	role PartnerRole,
	amountMinor int64,
) error {
	if amountMinor <= 0 {
		return nil
	}
	now := formatTime(s.nowFunc())
	_, err := tx.ExecContext(ctx, `
INSERT INTO finance_tasks
(id,type,provider,payment_intent_id,provider_refund_id,entry_id,
 beneficiary_id,beneficiary_role,amount_minor,currency,status,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(type,provider_refund_id,entry_id) DO NOTHING`,
		s.idFunc(), string(FinanceTaskCommissionClawback),
		item.Provider, item.PaymentID, item.ProviderRefundID, entryID,
		beneficiaryID, string(role), amountMinor, item.Currency,
		string(FinanceTaskPending), now, now)
	if err != nil {
		return fmt.Errorf("referral: ensure finance task: %w", err)
	}
	metrics.Default.Inc("referral_finance_task_created_total", map[string]string{"type": string(FinanceTaskCommissionClawback)})
	return nil
}
